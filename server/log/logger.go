package logger

import (
	"log"
	"os"
	"time"
)

func LogThis(message string) {
	t := time.Now()
	file, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := file.WriteString( t.Format("2006-01-02 15:04:05") + ": " + message + "\n"); err != nil {
		log.Fatal(err)
	}
	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
	log.Println(t.Format("2006-01-02 15:04:05") + ": " + message)
}