package log

import (
	"log"
    "os"
)

var (
	logInfo  *log.Logger
	logError *log.Logger
)

func Info(fmt string) {
    logInfo.Print(fmt);
}

func Infof(fmt string, data ...interface{}) {
    logInfo.Printf(fmt, data);
}

func Error(fmt string) {
    logError.Print(fmt);
}

func Errorf(fmt string, data ...interface{}) {
    logError.Printf(fmt, data);
}

func init() {
	logInfo = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmsgprefix)
	logError = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lmsgprefix)
}
