package utils

import (
	"log"
	"os"
)

var Logger *log.Logger

func init() {
	Logger = log.New(os.Stdout, "[CentralServer] ", log.LstdFlags|log.Lshortfile)
}

func Info(msg string) {
	Logger.Println("INFO: " + msg)
}

func Error(msg string) {
	Logger.Println("ERROR: " + msg)
}
