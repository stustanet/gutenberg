package main

import (
	"io/ioutil"
	"log"
	"os"
	"time"
)

func startCleaner(config *Config) {
	// cleanup once per hour
	ticker := time.NewTicker(time.Hour * 1)
	go func() {
		for t := range ticker.C {
			cleanup(t, config)
		}
	}()
}

func cleanup(t time.Time, config *Config) {
	files, err := ioutil.ReadDir(config.UploadPath)
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
		if int(t.Sub(file.ModTime())) > config.PruneUploads {
			log.Println("DELETE", file.Name(), os.Remove(config.UploadPath+file.Name()))
		}
	}
}
