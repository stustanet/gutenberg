package main

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

type Config struct {
	LpPath       string    `json:"lpPath"`
	UploadPath   string    `json:"uploadPath"`
	PruneUploads int       `json:"pruneUploads"`
	Dsn          string    `json:"dsn"`
	Printers     []Printer `json:"printers"`
}

type Printer struct {
	Name      string   `json:"name"`
	Host      string   `json:"host"`
	Instance  string   `json:"instance"`
	OptionsA5 []string `json:"optionsA5"`
	OptionsA4 []string `json:"optionsA4"`
	OptionsA3 []string `json:"optionsA3"`
}

func getConfig(configFile string) (config *Config, err error) {
	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	config = new(Config)
	err = json.Unmarshal([]byte(file), config)

	// correct times such that config.json uses seconds, but go uses nanoseconds
	config.PruneUploads = config.PruneUploads * int(time.Second)

	return config, err
}
