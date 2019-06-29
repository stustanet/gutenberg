package main

import (
	"database/sql"
	"errors"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func connectDB() {
	var err error
	// TODO: different user for admin
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func listJobs() (jobs []Job) {
	rows, err := db.Query("SELECT pin, bw, duplex, pages, sheets, price, copies, date, error FROM job ORDER BY id DESC")
	if err != nil {
		//fmt.Fprintln(w, err)
		return
	}

	job := Job{}

	//fmt.Fprintln(w, "<body><table><thead><tr><th>PIN</th><th>B/W</th><th>Duplex</th><th>Pages</th><th>Sheets</th><th>Price</th><th>Copies</th><th>Total</th><th>Date</th><th>Error</th><th>Action</th></tr></thead><tbody>")
	defer rows.Close()
	var pin, duplex string
	var pages, sheets, copies int
	var bw bool
	var price float64
	var date time.Time
	var error sql.NullString
	for rows.Next() {
		err := rows.Scan(&pin, &bw, &duplex, &pages, &sheets, &price, &copies, &date, &error)
		if err != nil {
			//fmt.Fprintln(w, err)
			return
		}

		job.BW = bw
		job.Copies = copies
		job.Duplex = duplex
		job.Pin = pin
		job.Pages = pages
		job.Sheets = sheets
		job.Created = date
		job.Total = price * float64(copies)

		//fmt.Fprintf(w, "<tr><th>%s</th><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%.02f &euro;</td><td>%d</td><td>%.02f &euro;</td><td>%s</td><td>%s</td><td><button class=\"print\" onclick=\"print('%s', '%.02f')\"><img src=\"/assets/img/print.png\" alt=\"Print\" /></button></td></tr>",
		//	pin, checkmark(bw), duplex, pages, sheets, price, copies, total, date.Format("2006-01-02 15:04:05"), error.String, pin, total,
		//)
	}
	if err = rows.Err(); err != nil {
		//fmt.Fprintln(w, err)
		return
	}

	return jobs
}

func listJobsDetail() (jobs []Job) {
	rows, err := db.Query("SELECT pin, file_id, ip_address, bw, cyan, magenta, yellow, key, duplex, pages, sheets, price, copies, date, error FROM job ORDER BY id DESC")
	//if err != nil {
	//	fmt.Fprintln(w, err)
	//	return
	//}

	job := Job{}

	//fmt.Fprintln(w, "<body><table><thead><tr><th>PIN</th><th>File</th><th>IP</th><th>B/W</th><th>Cyan</th><th>Magenta</th><th>Yellow</th><th>Key</th><th>Duplex</th><th>Pages</th><th>Sheets</th><th>Price</th><th>Copies</th><th>Date</th><th>Error</th></tr></thead><tbody>")
	defer rows.Close()
	var pin, file, ip, duplex string
	var pages, sheets, copies int
	var bw bool
	var cyan, magenta, yellow, key, price float64
	var date time.Time
	var error sql.NullString
	for rows.Next() {
		err := rows.Scan(&pin, &file, &ip, &bw, &cyan, &magenta, &yellow, &key, &duplex, &pages, &sheets, &price, &copies, &date, &error)
		if err != nil {
			//fmt.Fprintln(w, err)
			return
		}

		job.BW = bw
		job.Copies = copies
		job.Duplex = duplex
		job.Pages = pages
		job.Price = price
		job.Pin = pin
		job.Created = date
		job.File = file
		job.Sheets = sheets
		job.Ip = ip
		job.CMYK.Cyan = cyan
		job.CMYK.Magenta = magenta
		job.CMYK.Yellow = yellow
		job.CMYK.Key = key

		jobs = append(jobs, job)

		//fmt.Fprintf(w,
		//	"<tr><th>%s</th><td>%s</td><td>%s</td><td>%t</td><td>%f</td><td>%f</td><td>%f</td><td>%f</td><td>%s</td><td>%d</td><td>%d</td><td>%.02f &euro;</td><td>%d</td><td>%s</td><td>%s</td></tr>",
		//	pin, file, ip, bw, cyan, magenta, yellow, key, duplex, pages, sheets, price, copies, date.Format("2006-01-02 15:04:05"), error.String,
		//)
	}
	if err = rows.Err(); err != nil {
		//fmt.Fprintln(w, err)
		return
	}

	return jobs
}

func listLog() (logs []Log) {
	rows, err := db.Query("SELECT internal, bw, duplex, pages, sheets, price, copies, create_date, print_date, error FROM log ORDER BY id DESC")
	if err != nil {
		//fmt.Fprintln(w, err)
		return
	}

	log := Log{}

	//fmt.Fprintln(w, "<!doctype html><html><head><title>Logged Jobs</title><link rel=\"stylesheet\" href=\"/assets/css/admin.css\"></head>")
	//fmt.Fprintln(w, "<body><table><thead><tr><th>Internal</th><th>B/W</th><th>Duplex</th><th>Pages</th><th>Sheets</th><th>Price</th><th>Copies</th><th>Total</th><th>Create Date</th><th>Print Date</th><th>Error</th></tr></thead><tbody>")
	defer rows.Close()
	var duplex string
	var pages, sheets, copies int
	var internal, bw bool
	var price float64
	var createDate, printDate time.Time
	var error sql.NullString
	for rows.Next() {
		err := rows.Scan(&internal, &bw, &duplex, &pages, &sheets, &price, &copies, &createDate, &printDate, &error)
		if err != nil {
			//fmt.Fprintln(w, err)
			return
		}

		log.BW = bw
		log.Duplex = duplex
		log.Pages = pages
		log.Sheets = sheets
		log.Copies = copies
		log.Created = createDate
		log.Printed = printDate
		log.Internal = internal
		log.Total = price * float64(copies)

		logs = append(logs, log)
		//fmt.Fprintf(w,
		//	"<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%.02f &euro;</td><td>%d</td><td>%.02f &euro;</td><td>%s</td><td>%s</td><td>%s</td></tr>",
		//	checkmark(internal), checkmark(bw), duplex, pages, sheets, price, copies, (price * float64(copies)), createDate.Format("2006-01-02 15:04:05"), printDate.Format("2006-01-02 15:04:05"), error.String,
		//)
	}

	if err = rows.Err(); err != nil {
		//fmt.Fprintln(w, err)
		return
	}

	return logs
}

func getJob(pin string) (*Job, error) {
	var j Job
	row := db.QueryRow("SELECT file_id, bw, cyan, magenta, yellow, key, duplex, pages, sheets, price, copies, date FROM job WHERE pin = $1", pin)
	err := row.Scan(&j.File, &j.BW, &j.CMYK.Cyan, &j.CMYK.Magenta, &j.CMYK.Yellow, &j.CMYK.Key, &j.Duplex, &j.Pages, &j.Sheets, &j.Price, &j.Copies, &j.Created)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.New("no job with this ID")
		}
		return nil, err
	}

	return &j, nil
}

func saveLog(j *Job) (err error) {
	_, err = db.Exec(
		"INSERT INTO log (internal, bw, cyan, magenta, yellow, key, duplex, pages, sheets, price, copies, create_date, print_date, error) "+
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)",
		j.Internal, j.BW, j.CMYK.Cyan, j.CMYK.Magenta, j.CMYK.Yellow, j.CMYK.Key, j.Duplex, j.Pages, j.Sheets, j.Price, j.Copies, j.Created, j.Printed, j.Err,
	)
	return
}

func deleteOldJobs() (err error) {
	_, err = db.Exec("DELETE FROM job WHERE date < NOW() - INTERVAL '7 days'")
	return
}
