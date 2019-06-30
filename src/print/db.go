package main

import (
	"database/sql"
	"log"
	"math/rand"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func connectDB(dsn string) {
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func genPIN() int {
	const max = 999999
	const min = 100000
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func saveJob(j *Job) (err error) {
	duplex := "simplex"
	switch j.Duplex {
	case duplexLong:
		duplex = "long"
	case duplexShort:
		duplex = "short"
	}

	for {
		j.PIN = genPIN()
		_, err = db.Exec(
			"INSERT INTO job (file_id, pin, ip_address, bw, cyan, magenta, yellow, key, duplex, format, pages, sheets, price, copies, error) "+
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)",
			j.File[7:], j.PIN, j.IP, j.BW, j.CMYK.Cyan, j.CMYK.Magenta, j.CMYK.Yellow, j.CMYK.Key, duplex, j.Format, j.Pages, j.Sheets, j.Price, j.Copies, j.Err.Error(),
		)
		if err == nil || !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return
		}
	}
}
