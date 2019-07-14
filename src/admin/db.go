package main

import (
	"database/sql"
	"errors"
	_ "github.com/lib/pq"
	"log"
)

var db *sql.DB

func connectDB(dsn string) {
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

func listJobs() (jobs []Job, err error) {
	rows, err := db.Query("SELECT pin, bw, duplex, format, pages, sheets, price, copies, date, error FROM job ORDER BY id DESC")
	if err != nil {
		return
	}

	job := Job{}

	defer rows.Close()
	var error sql.NullString
	for rows.Next() {
		err = rows.Scan(&job.Pin, &job.BW, &job.Duplex, &job.Format, &job.Pages, &job.Sheets, &job.Price, &job.Copies, &job.Created, &error)
		if err != nil {
			return
		}

		job.Total = job.Price * float64(job.Copies)
		job.Err = errors.New(error.String)

		jobs = append(jobs, job)
	}
	if err = rows.Err(); err != nil {
		return
	}

	return jobs, err
}

func listJobsDetail() (jobs []Job, err error) {
	rows, err := db.Query("SELECT pin, file_id, ip_address, bw, cyan, magenta, yellow, key, duplex, format, pages, sheets, price, copies, date, error FROM job ORDER BY id DESC")
	if err != nil {
		return
	}

	job := Job{}

	defer rows.Close()
	var error sql.NullString
	for rows.Next() {
		err = rows.Scan(&job.Pin, &job.File, &job.Ip, &job.BW, &job.CMYK.Cyan, &job.CMYK.Magenta, &job.CMYK.Yellow, &job.CMYK.Key, &job.Duplex, &job.Format, &job.Pages, &job.Sheets, &job.Price, &job.Copies, &job.Created, &error)
		if err != nil {
			return
		}

		job.Total = job.Price * float64(job.Copies)
		job.Err = errors.New(error.String)

		jobs = append(jobs, job)
	}
	if err = rows.Err(); err != nil {
		return
	}

	return jobs, err
}

func listLog() (logs []Log, err error) {
	rows, err := db.Query("SELECT internal, bw, duplex, format, pages, sheets, price, copies, create_date, print_date, error FROM log ORDER BY id DESC")
	if err != nil {
		return
	}

	logTmp := Log{}

	defer rows.Close()
	var error sql.NullString
	for rows.Next() {
		err = rows.Scan(&logTmp.Internal, &logTmp.BW, &logTmp.Duplex, &logTmp.Format, &logTmp.Pages, &logTmp.Sheets, &logTmp.Price, &logTmp.Copies, &logTmp.Created, &logTmp.Printed, &error)
		if err != nil {
			return
		}

		logTmp.Total = logTmp.Price * float64(logTmp.Copies)
		logTmp.Err = errors.New(error.String)

		logs = append(logs, logTmp)
	}

	if err = rows.Err(); err != nil {
		return
	}

	return logs, err
}

func getJob(pin string) (*Job, error) {
	var j Job
	row := db.QueryRow("SELECT file_id, bw, cyan, magenta, yellow, key, duplex, format, pages, sheets, price, copies, date FROM job WHERE pin = $1", pin)
	err := row.Scan(&j.File, &j.BW, &j.CMYK.Cyan, &j.CMYK.Magenta, &j.CMYK.Yellow, &j.CMYK.Key, &j.Duplex, &j.Format, &j.Pages, &j.Sheets, &j.Price, &j.Copies, &j.Created)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.New("no job with this ID")
		}
		return nil, err
	}

	return &j, nil
}

func saveLog(j *Job) (err error) {
	error := ""
	if j.Err != nil {
		error = j.Err.Error()
	} else {
		_, err = db.Exec(
			"INSERT INTO log (internal, bw, cyan, magenta, yellow, key, duplex, format, pages, sheets, price, copies, create_date, print_date, error) "+
				"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)",
			j.Internal, j.BW, j.CMYK.Cyan, j.CMYK.Magenta, j.CMYK.Yellow, j.CMYK.Key, j.Duplex, j.Format, j.Pages, j.Sheets, j.Price, j.Copies, j.Created, j.Printed, error,
		)
	}
	return
}

func deleteOldJobs() (err error) {
	_, err = db.Exec("DELETE FROM job WHERE date < NOW() - INTERVAL '7 days'")
	return
}
