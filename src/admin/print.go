package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"
	"time"
)

type Job struct {
	File     string
	CMYK     Coverage // ink coverage
	Duplex   string
	BW       bool
	Internal bool
	Pages    int
	Sheets   int
	Copies   int
	Price    float64 // per copy
	Total    float64 // total amount
	Created  time.Time
	Printed  time.Time
	Err      error
}

type Coverage struct {
	Cyan    float64
	Magenta float64
	Yellow  float64
	Key     float64
}

func printJob(w io.Writer, j *Job) (err error) {
	// Simplex / Duplex option
	var duplex string
	switch j.Duplex {
	case "short":
		duplex = "Duplex=DuplexTumble"
	case "long":
		duplex = "Duplex=DuplexNoTumble"
	default:
		duplex = "Duplex=None"
	}

	// Color Model
	color := "ColorModel=CMYK"
	if j.BW {
		color = "ColorModel=Gray" // Printer specific!
	}

	n := j.Copies
	if n < 1 {
		n = 1
	}
	cmd := exec.Command(lpPath, "-d", printer, "-n", strconv.Itoa(n), "-o", "Collate=True", "-o", color, "-o", duplex, uploadPath+j.File)

	// Pipe stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	// Pipe stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	// Start command
	if err = cmd.Start(); err != nil {
		return
	}

	// Read stdout from pipe
	out, err := ioutil.ReadAll(stdout)
	if err != nil {
		return
	}

	// Read stderr from pipe
	errout, err := ioutil.ReadAll(stderr)
	if err != nil {
		return
	}

	// Wait for command to finish
	err = cmd.Wait()
	if len(errout) > 1 {
		err = errors.New(string(errout))
	}
	j.Err = err
	j.Printed = time.Now()

	// Save print job to log
	if err1 := saveLog(j); err1 != nil {
		fmt.Fprintln(w, err1.Error())
	}

	fmt.Fprintln(w, string(out))
	return
}
