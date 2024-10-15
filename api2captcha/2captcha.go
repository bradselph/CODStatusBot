package api2captcha

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	BaseURL       = "https://2captcha.com"
	DefaultSoftId = 4676
)

type (
	Request struct {
		Params map[string]string
		Files  map[string]string
	}

	Client struct {
		BaseURL          *url.URL
		ApiKey           string
		SoftId           int
		Callback         string
		DefaultTimeout   int
		RecaptchaTimeout int
		PollingInterval  int
		httpClient       *http.Client
	}

	ReCaptcha struct {
		SiteKey    string
		Url        string
		Invisible  bool
		Enterprise bool
		Version    string
		Action     string
		DataS      string
		Score      float64
	}
)

var (
	ErrNetwork = errors.New("api2captcha: Network failure")
	ErrApi     = errors.New("api2captcha: API error")
	ErrTimeout = errors.New("api2captcha: Request timeout")
)

func NewClient(apiKey string) *Client {
	base, _ := url.Parse(BaseURL)
	return &Client{
		BaseURL:          base,
		ApiKey:           apiKey,
		SoftId:           DefaultSoftId,
		DefaultTimeout:   120,
		PollingInterval:  10,
		RecaptchaTimeout: 600,
		httpClient:       &http.Client{},
	}
}

func (c *Client) res(req Request) (*string, error) {

	rel := &url.URL{Path: "/res.php"}
	uri := c.BaseURL.ResolveReference(rel)

	req.Params["key"] = c.ApiKey
	c.httpClient.Timeout = time.Duration(c.DefaultTimeout) * time.Second

	var resp *http.Response = nil

	values := url.Values{}
	for key, val := range req.Params {
		values.Add(key, val)
	}
	uri.RawQuery = values.Encode()

	var err error = nil
	resp, err = http.Get(uri.String())
	if err != nil {
		return nil, ErrNetwork
	}

	defer resp.Body.Close()
	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	data := body.String()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrApi
	}

	if strings.HasPrefix(data, "ERROR_") {
		return nil, ErrApi
	}

	return &data, nil
}

func (c *Client) resAction(action string) (*string, error) {
	req := Request{
		Params: map[string]string{"action": action},
	}

	return c.res(req)
}

func (c *Client) Send(req Request) (string, error) {

	rel := &url.URL{Path: "/in.php"}
	uri := c.BaseURL.ResolveReference(rel)

	req.Params["key"] = c.ApiKey

	c.httpClient.Timeout = time.Duration(c.DefaultTimeout) * time.Second

	var resp *http.Response = nil
	if req.Files != nil && len(req.Files) > 0 {

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for name, path := range req.Files {
			file, err := os.Open(path)
			if err != nil {
				return "", err
			}
			defer file.Close()

			part, err := writer.CreateFormFile(name, filepath.Base(path))
			if err != nil {
				return "", err
			}
			_, err = io.Copy(part, file)
		}

		for key, val := range req.Params {
			_ = writer.WriteField(key, val)
		}

		err := writer.Close()
		if err != nil {
			return "", err
		}

		request, err := http.NewRequest("POST", uri.String(), body)
		if err != nil {
			return "", err
		}

		request.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err = c.httpClient.Do(request)
		if err != nil {
			return "", ErrNetwork
		}
	} else {
		values := url.Values{}
		for key, val := range req.Params {
			values.Add(key, val)
		}

		var err error = nil
		resp, err = http.PostForm(uri.String(), values)
		if err != nil {
			return "", ErrNetwork
		}
	}

	defer resp.Body.Close()
	body := &bytes.Buffer{}
	_, err := body.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}
	data := body.String()

	if resp.StatusCode != http.StatusOK {
		return "", ErrApi
	}

	if strings.HasPrefix(data, "ERROR_") {
		return "", ErrApi
	}

	if !strings.HasPrefix(data, "OK|") {
		return "", ErrApi
	}

	return data[3:], nil
}

func (c *Client) Solve(req Request) (string, string, error) {
	if c.Callback != "" {
		_, ok := req.Params["pingback"]
		if !ok {
			// set default pingback
			req.Params["pingback"] = c.Callback
		}
	}

	pingback, hasPingback := req.Params["pingback"]
	if pingback == "" {
		delete(req.Params, "pingback")
		hasPingback = false
	}

	_, ok := req.Params["soft_id"]
	if c.SoftId != 0 && !ok {
		req.Params["soft_id"] = strconv.FormatInt(int64(c.SoftId), 10)
	}

	id, err := c.Send(req)
	if err != nil {
		return "", "", err
	}

	// don't wait for result if Callback is used
	if hasPingback {
		return "", id, nil
	}

	timeout := c.DefaultTimeout
	if req.Params["method"] == "userrecaptcha" {
		timeout = c.RecaptchaTimeout
	}

	token, err := c.WaitForResult(id, timeout, c.PollingInterval)
	if err != nil {
		return "", "", err
	}

	return token, id, nil
}

func (c *Client) WaitForResult(id string, timeout int, interval int) (string, error) {

	start := time.Now()
	now := start
	for now.Sub(start) < (time.Duration(timeout) * time.Second) {

		time.Sleep(time.Duration(interval) * time.Second)

		code, err := c.GetResult(id)
		if err == nil && code != nil {
			return *code, nil
		}

		// ignore network errors
		if err != nil && err != ErrNetwork {
			return "", err
		}

		now = time.Now()
	}

	return "", ErrTimeout
}

func (c *Client) GetResult(id string) (*string, error) {
	req := Request{
		Params: map[string]string{"action": "get", "id": id},
	}

	data, err := c.res(req)
	if err != nil {
		return nil, err
	}

	if *data == "CAPCHA_NOT_READY" {
		return nil, nil
	}

	if !strings.HasPrefix(*data, "OK|") {
		return nil, ErrApi
	}

	reply := (*data)[3:]
	return &reply, nil
}

func (c *Client) GetBalance() (float64, error) {
	data, err := c.resAction("getbalance")
	if err != nil {
		return 0.0, err
	}

	return strconv.ParseFloat(*data, 64)
}

func (c *Client) Report(id string, correct bool) error {
	req := Request{
		Params: map[string]string{"id": id},
	}
	if correct {
		req.Params["action"] = "reportgood"
	} else {
		req.Params["action"] = "reportbad"
	}

	_, err := c.res(req)
	return err
}

func (req *Request) SetProxy(proxyType string, uri string) {
	req.Params["proxytype"] = proxyType
	req.Params["proxy"] = uri
}

func (req *Request) SetSoftId(softId int) {
	req.Params["soft_id"] = strconv.FormatInt(int64(softId), 10)
}

func (req *Request) SetCallback(callback string) {
	req.Params["pingback"] = callback
}

func (c *ReCaptcha) ToRequest() Request {
	req := Request{
		Params: map[string]string{"method": "userrecaptcha"},
	}
	if c.SiteKey != "" {
		req.Params["googlekey"] = c.SiteKey
	}
	if c.Url != "" {
		req.Params["pageurl"] = c.Url
	}
	if c.Invisible {
		req.Params["invisible"] = "1"
	}
	if c.Enterprise {
		req.Params["enterprise"] = "1"
	}
	if c.Version != "" {
		req.Params["version"] = c.Version
	}
	if c.Action != "" {
		req.Params["action"] = c.Action
	}
	if c.DataS != "" {
		req.Params["data-s"] = c.DataS
	}
	if c.Score != 0 {
		req.Params["min_score"] = strconv.FormatFloat(c.Score, 'f', -1, 64)
	}

	return req
}
