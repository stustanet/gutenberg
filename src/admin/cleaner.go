package main

import (
	"io/ioutil"
	"log"
	"os"
	"time"
)

func startCleaner() {
	// cleanup once per hour
	ticker := time.NewTicker(time.Hour * 1)
	go func() {
		for t := range ticker.C {
			cleanup(t)
		}
	}()
}

func cleanup(t time.Time) {
	files, err := ioutil.ReadDir(uploadPath)
	if err != nil {
		log.Println(err)
		return
	}

	if err = deleteOldJobs(); err != nil {
		log.Println(err)
		return
	}

	for _, file := range files {
		// Delete file older than x days
		if t.Sub(file.ModTime()) > pruneUploads {
			log.Println("DELETE", file.Name(), os.Remove(uploadPath+file.Name()))
		}
	}
}
