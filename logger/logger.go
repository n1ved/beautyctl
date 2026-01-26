package logger

import (
	"log"
	"os"
)

var (
	File *os.File
    Log  *log.Logger
)

func Init(path string) error {
	var err error
	File, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	Log = log.New(File, "[BEAUTYCTL] ", log.LstdFlags|log.Lshortfile)
	return nil
}

func Close() {
	if File != nil {
		File.Close()
	}
}

func Printf(format string, v ...interface{}) {
    if Log != nil {
        Log.Printf(format, v...)
    }
}

func Println(v ...interface{}) {
    if Log != nil {
        Log.Println(v...)
    }
}
