package main

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

type Config struct {
	PriceBlack float64 `json:"priceBlack"`
	// Our cost for color toner (same for all colors) is about 190 € for 7000
	// pages based on 5% toner usage; this means 0.0271 € for a 5% covered page.
	// We will use 0.040 € instead, since it seems like the printer uses
	// slightly more toner than calculated.
	// Therefore, price per 1% page coverage:
	PriceColor float64 `json:"priceColor"`
	// additional cost per page for the fuser and waste toner
	PriceFuser float64 `json:"priceFuser"`
	// price per sheet of paper
	PriceSheet  float64 `json:"priceSheet"`
	UploadPath  string  `json:"uploadPath"`
	MaxFileSize int     `json:"maxFileSize"`
	HostDE      string  `json:"hostDE"`
	HostEN      string  `json:"hostEN"`
	// DSN for Postgres DB
	Dsn        string `json:"dsn"`
	TlsPrivKey string `json:"tlsPrivKey"`
	TlsCert    string `json:"tlsCert"`
	// Hackerspace status URL
	HaspaStatusURL    string `json:"haspaStatusURL"`
	HaspaStatusMaxAge int    `json:"haspaStatusMaxAge"`
	// Office Hours JSON URL
	OfficeHoursURL string `json:"officeHoursURL"`
	// max age in seconds
	OfficeHoursMaxAge int `json:"officeHoursMaxAge"`
}

func getConfig(configFile string) (config *Config, err error) {
	file, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	config = new(Config)
	err = json.Unmarshal([]byte(file), config)

	// correct times such that config.json uses seconds, but go uses nanoseconds
	config.OfficeHoursMaxAge = config.OfficeHoursMaxAge * int(time.Second)

	return config, err
}
