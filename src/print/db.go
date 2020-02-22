package main

import (
	"database/sql"
	"fmt"
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
		error := ""
		if j.Err != nil {
			error = j.Err.Error()
		}
		filename := strings.Split(j.File, "/")

		_, err = db.Exec(
			"INSERT INTO job (file_id, pin, ip_address, bw, cyan, magenta, yellow, key, duplex, format, pages, sheets, price, copies, rotated, date, error) "+
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)",
			filename[len(filename)-1], j.PIN, j.IP, j.BW, j.CMYK.Cyan, j.CMYK.Magenta, j.CMYK.Yellow, j.CMYK.Key, duplex, j.Format, j.Pages, j.Sheets, j.Price, j.Copies, j.Rotated, j.Created, error,
		)
		if err != nil {
			fmt.Println("Save Job error:", err.Error())
		}
		if err == nil || !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return
		}
	}
}
