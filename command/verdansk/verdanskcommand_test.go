package verdansk

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/bradselph/CODStatusBot/models"
	"github.com/bwmarrin/discordgo"
)

func Test_canUserRunVerdanskCommand(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := canUserRunVerdanskCommand(tt.args.userID)
			if got != tt.want {
				t.Errorf("canUserRunVerdanskCommand() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("canUserRunVerdanskCommand() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_updateUserRateLimit(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateUserRateLimit(tt.args.userID)
		})
	}
}

func Test_trackVerdanskDownload(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trackVerdanskDownload(tt.args.userID)
		})
	}
}

func Test_getTotalDownloadsToday(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTotalDownloadsToday(); got != tt.want {
				t.Errorf("getTotalDownloadsToday() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandVerdansk(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CommandVerdansk(tt.args.s, tt.args.i)
		})
	}
}

func TestHandleMethodSelection(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleMethodSelection(tt.args.s, tt.args.i)
		})
	}
}

func Test_showActivisionIDModal(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			showActivisionIDModal(tt.args.s, tt.args.i)
		})
	}
}

func Test_showAccountSelection(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			showAccountSelection(tt.args.s, tt.args.i)
		})
	}
}

func TestHandleAccountSelection(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleAccountSelection(tt.args.s, tt.args.i)
		})
	}
}

func TestHandleActivisionIDModal(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleActivisionIDModal(tt.args.s, tt.args.i)
		})
	}
}

func Test_getActivisionIDFromAccount(t *testing.T) {
	type args struct {
		account models.Account
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getActivisionIDFromAccount(tt.args.account)
			if (err != nil) != tt.wantErr {
				t.Errorf("getActivisionIDFromAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getActivisionIDFromAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processVerdanskStats(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		i            *discordgo.InteractionCreate
		activisionID string
		account      *models.Account
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processVerdanskStats(tt.args.s, tt.args.i, tt.args.activisionID, tt.args.account)
		})
	}
}

func Test_fetchPlayerPreferences(t *testing.T) {
	type args struct {
		client          *http.Client
		encodedGamerTag string
	}
	tests := []struct {
		name    string
		args    args
		want    *PlayerPreferences
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchPlayerPreferences(tt.args.client, tt.args.encodedGamerTag)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchPlayerPreferences() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchPlayerPreferences() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fetchPlayerStats(t *testing.T) {
	type args struct {
		client          *http.Client
		encodedGamerTag string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]StatValue
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchPlayerStats(tt.args.client, tt.args.encodedGamerTag)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchPlayerStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetchPlayerStats() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_downloadImages(t *testing.T) {
	type args struct {
		client      *http.Client
		stats       map[string]StatValue
		outputDir   string
		concurrency int
	}
	tests := []struct {
		name    string
		args    args
		want    []ImageDownload
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := downloadImages(tt.args.client, tt.args.stats, tt.args.outputDir, tt.args.concurrency)
			if (err != nil) != tt.wantErr {
				t.Errorf("downloadImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("downloadImages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createZip(t *testing.T) {
	type args struct {
		images  []ImageDownload
		zipName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := createZip(tt.args.images, tt.args.zipName); (err != nil) != tt.wantErr {
				t.Errorf("createZip() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_cleanupTempFiles(t *testing.T) {
	type args struct {
		uniqueID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTempFiles(tt.args.uniqueID)
		})
	}
}

func TestInitCleanupRoutine(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitCleanupRoutine()
		})
	}
}

func Test_cleanupAllTempFiles(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupAllTempFiles()
		})
	}
}

func Test_cleanupOldZipFiles(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupOldZipFiles()
		})
	}
}

func Test_doPreflightRequest(t *testing.T) {
	type args struct {
		client    *http.Client
		targetURL string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := doPreflightRequest(tt.args.client, tt.args.targetURL); (err != nil) != tt.wantErr {
				t.Errorf("doPreflightRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_readResponseBody(t *testing.T) {
	type args struct {
		resp *http.Response
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readResponseBody(tt.args.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("readResponseBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readResponseBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatStatName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatStatName(tt.args.name); got != tt.want {
				t.Errorf("formatStatName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_respondToInteraction(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		i       *discordgo.InteractionCreate
		message string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respondToInteraction(tt.args.s, tt.args.i, tt.args.message)
		})
	}
}

func Test_sendFollowupMessage(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		i       *discordgo.InteractionCreate
		message string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendFollowupMessage(tt.args.s, tt.args.i, tt.args.message)
		})
	}
}

func Test_storeVerdanskStatsWithAccount(t *testing.T) {
	type args struct {
		account *models.Account
		stats   map[string]StatValue
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := storeVerdanskStatsWithAccount(tt.args.account, tt.args.stats); (err != nil) != tt.wantErr {
				t.Errorf("storeVerdanskStatsWithAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_createVerdanskAPIRequest(t *testing.T) {
	type args struct {
		method string
		url    string
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Request
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createVerdanskAPIRequest(tt.args.method, tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("createVerdanskAPIRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createVerdanskAPIRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateVerdanskAPIResponse(t *testing.T) {
	type args struct {
		body         []byte
		expectedType string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateVerdanskAPIResponse(tt.args.body, tt.args.expectedType); (err != nil) != tt.wantErr {
				t.Errorf("validateVerdanskAPIResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_sortStatImages(t *testing.T) {
	type args struct {
		images []ImageDownload
		stats  map[string]StatValue
	}
	tests := []struct {
		name string
		args args
		want []ImageDownload
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sortStatImages(tt.args.images, tt.args.stats); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortStatImages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getImageCategory(t *testing.T) {
	type args struct {
		statName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getImageCategory(tt.args.statName); got != tt.want {
				t.Errorf("getImageCategory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_organizeImagesForDisplay(t *testing.T) {
	type args struct {
		images    []ImageDownload
		stats     map[string]StatValue
		maxImages int
	}
	tests := []struct {
		name string
		args args
		want []ImageDownload
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := organizeImagesForDisplay(tt.args.images, tt.args.stats, tt.args.maxImages); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("organizeImagesForDisplay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateVerdanskStatsEmbed(t *testing.T) {
	type args struct {
		activisionID string
		stats        map[string]StatValue
	}
	tests := []struct {
		name string
		args args
		want *discordgo.MessageEmbed
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateVerdanskStatsEmbed(tt.args.activisionID, tt.args.stats); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("generateVerdanskStatsEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_retryVerdanskRequest(t *testing.T) {
	type args struct {
		client     *http.Client
		method     string
		url        string
		maxRetries int
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Response
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := retryVerdanskRequest(tt.args.client, tt.args.method, tt.args.url, tt.args.maxRetries)
			if (err != nil) != tt.wantErr {
				t.Errorf("retryVerdanskRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("retryVerdanskRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_canStartNewDownload(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canStartNewDownload(tt.args.userID); got != tt.want {
				t.Errorf("canStartNewDownload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_incrementActiveDownloads(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incrementActiveDownloads(tt.args.userID)
		})
	}
}

func Test_decrementActiveDownloads(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrementActiveDownloads(tt.args.userID)
		})
	}
}

func Test_markAccountWithVerdanskStats(t *testing.T) {
	type args struct {
		userID       string
		activisionID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markAccountWithVerdanskStats(tt.args.userID, tt.args.activisionID)
		})
	}
}

func Test_loadVerdanskConfiguration(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadVerdanskConfiguration()
		})
	}
}

func TestStartVerdanskCleanupRoutine(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			StartVerdanskCleanupRoutine()
		})
	}
}

func Test_createVerdanskSession(t *testing.T) {
	type args struct {
		userID       string
		activisionID string
		accountID    uint
	}
	tests := []struct {
		name string
		args args
		want *VerdanskSession
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createVerdanskSession(tt.args.userID, tt.args.activisionID, tt.args.accountID); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createVerdanskSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateSessionState(t *testing.T) {
	type args struct {
		userID   string
		newState string
		errorMsg string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateSessionState(tt.args.userID, tt.args.newState, tt.args.errorMsg)
		})
	}
}

func Test_logVerdanskJobCompletion(t *testing.T) {
	type args struct {
		userID        string
		operationType string
		success       bool
		duration      time.Duration
		details       string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logVerdanskJobCompletion(tt.args.userID, tt.args.operationType, tt.args.success, tt.args.duration, tt.args.details)
		})
	}
}

func Test_cleanupSession(t *testing.T) {
	type args struct {
		userID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupSession(tt.args.userID)
		})
	}
}

func Test_optimizeJPEG(t *testing.T) {
	type args struct {
		imageData []byte
		quality   int
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := optimizeJPEG(tt.args.imageData, tt.args.quality)
			if (err != nil) != tt.wantErr {
				t.Errorf("optimizeJPEG() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("optimizeJPEG() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processImageBatch(t *testing.T) {
	type args struct {
		images []ImageDownload
	}
	tests := []struct {
		name    string
		args    args
		want    []ImageDownload
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processImageBatch(tt.args.images)
			if (err != nil) != tt.wantErr {
				t.Errorf("processImageBatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processImageBatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sendProgressUpdate(t *testing.T) {
	type args struct {
		s       *discordgo.Session
		i       *discordgo.InteractionCreate
		message string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendProgressUpdate(tt.args.s, tt.args.i, tt.args.message)
		})
	}
}

func Test_createVerdanskProgressBar(t *testing.T) {
	type args struct {
		current   int
		total     int
		barLength int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createVerdanskProgressBar(tt.args.current, tt.args.total, tt.args.barLength); got != tt.want {
				t.Errorf("createVerdanskProgressBar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateVerdanskSummary(t *testing.T) {
	type args struct {
		stats map[string]StatValue
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateVerdanskSummary(tt.args.stats); got != tt.want {
				t.Errorf("generateVerdanskSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_verifyVerdanskAvailability(t *testing.T) {
	type args struct {
		activisionID string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := verifyVerdanskAvailability(tt.args.activisionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyVerdanskAvailability() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("verifyVerdanskAvailability() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getActivisionAccountInfo(t *testing.T) {
	type args struct {
		client       *http.Client
		activisionID string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getActivisionAccountInfo(tt.args.client, tt.args.activisionID)
			if (err != nil) != tt.wantErr {
				t.Errorf("getActivisionAccountInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getActivisionAccountInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sendVerdanskErrorEmbed(t *testing.T) {
	type args struct {
		s           *discordgo.Session
		i           *discordgo.InteractionCreate
		title       string
		description string
		errorDetail string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendVerdanskErrorEmbed(tt.args.s, tt.args.i, tt.args.title, tt.args.description, tt.args.errorDetail)
		})
	}
}

func Test_isVerdanskAPIAvailable(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVerdanskAPIAvailable(); got != tt.want {
				t.Errorf("isVerdanskAPIAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_waitForRateLimit(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			waitForRateLimit()
		})
	}
}

func Test_enrichStatData(t *testing.T) {
	type args struct {
		stats map[string]StatValue
	}
	tests := []struct {
		name string
		args args
		want map[string]StatValue
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := enrichStatData(tt.args.stats); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("enrichStatData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getStatStringValue(t *testing.T) {
	type args struct {
		stats map[string]StatValue
		keys  []string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getStatStringValue(tt.args.stats, tt.args.keys...)
			if got != tt.want {
				t.Errorf("getStatStringValue() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getStatStringValue() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getStatIntValue(t *testing.T) {
	type args struct {
		stats map[string]StatValue
		keys  []string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStatIntValue(tt.args.stats, tt.args.keys...); got != tt.want {
				t.Errorf("getStatIntValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleVerdanskCommand(t *testing.T) {
	type args struct {
		s *discordgo.Session
		i *discordgo.InteractionCreate
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			HandleVerdanskCommand(tt.args.s, tt.args.i)
		})
	}
}

func Test_notifyUnsupportedMessage(t *testing.T) {
	type args struct {
		s            *discordgo.Session
		i            *discordgo.InteractionCreate
		activisionID string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifyUnsupportedMessage(tt.args.s, tt.args.i, tt.args.activisionID)
		})
	}
}

func Test_getVerdanskFeatureEmbed(t *testing.T) {
	tests := []struct {
		name string
		want *discordgo.MessageEmbed
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getVerdanskFeatureEmbed(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getVerdanskFeatureEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_enrichVerdanskFilenames(t *testing.T) {
	type args struct {
		images []ImageDownload
	}
	tests := []struct {
		name string
		args args
		want []ImageDownload
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := enrichVerdanskFilenames(tt.args.images); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("enrichVerdanskFilenames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_groupImagesByCategory(t *testing.T) {
	type args struct {
		images []ImageDownload
	}
	tests := []struct {
		name string
		args args
		want map[string][]ImageDownload
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := groupImagesByCategory(tt.args.images); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupImagesByCategory() = %v, want %v", got, tt.want)
			}
		})
	}
}
