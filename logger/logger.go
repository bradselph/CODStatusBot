package logger

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	Log           *logrus.Logger
	logFile       *os.File
	lastRotation  time.Time
	rotationMutex sync.Mutex
)

func InitializeLogger() error {

	Log = logrus.New()
	Log.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	logDir := "logs/"

	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		return err
	}

	rotateLog()
	go checkRotation()

	return nil
}

func init() {
	Log = logrus.New()
	Log.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
}

func rotateLog() {
	rotationMutex.Lock()
	defer rotationMutex.Unlock()

	if logFile != nil {
		err := logFile.Close()
		if err != nil {
			Log.WithError(err).Error("Failed to close log file")
		}
	}

	logDir := "logs/"

	logFileName := filepath.Join(logDir, time.Now().Format("2006-01-02")+".txt")
	newLogFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		Log.WithError(err).Fatal("Failed to open new log file")
	}

	logFile = newLogFile
	mw := io.MultiWriter(os.Stdout, logFile)
	Log.SetOutput(mw)
	lastRotation = time.Now()
}

func checkRotation() {
	for {
		time.Sleep(1 * time.Hour)

		now := time.Now()
		if now.YearDay() != lastRotation.YearDay() {
			rotateLog()
			Log.Info("Log file rotated")
		}
	}
}
