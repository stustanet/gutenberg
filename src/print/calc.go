package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	simplex     = 0
	duplexLong  = 1
	duplexShort = 2
)

type Job struct {
	File     string   // name of stored file
	Name     string   // original file name
	Password string   // PDF password (optional)
	IP       string   // submitter IP address
	CMYK     Coverage // ink coverage
	PIN      int      // print job PIN (generated)
	Duplex   int8
	Format   string   // A5, A4, A3
	BW       bool
	Pages    int
	Sheets   int
	Copies   int
	Price    float64       // per copy
	Total    float64       // total amount
	Runtime  time.Duration // calculation run time
	Err      error
}

type Coverage struct {
	Cyan    float64
	Magenta float64
	Yellow  float64
	Key     float64
}

func (c *Coverage) Price() float64 {
	return ((c.Cyan + c.Magenta + c.Yellow) * config.PriceColor) +
		(c.Key * config.PriceBlack)
}

func (c *Coverage) Print(w io.Writer, page int) {
	fmt.Fprintf(w,
		"%04d: C %.6f  M %.6f  Y %.6f  K %.6f  =  %.6f € \n",
		page,
		c.Cyan,
		c.Magenta,
		c.Yellow,
		c.Key,
		c.Price(),
	)
}

var (
	ErrInvalidFormat    = errors.New("invalid format")
	ErrPasswordRequired = errors.New("password required")
)

func pdfInfo(j *Job) {
	var cmd *exec.Cmd
	if j.Password == "" {
		cmd = exec.Command("pdfinfo", j.File)
	} else {
		cmd = exec.Command("pdfinfo", "-upw", j.Password, j.File)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		j.Err = err
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		j.Err = err
		return
	}
	if err = cmd.Start(); err != nil {
		j.Err = err
		return
	}

	rdr := bufio.NewReader(stdout)
	lines := 0
	for {
		var line []byte
		line, _, err = rdr.ReadLine()
		if err != nil {
			if err != io.EOF {
				j.Err = err
				return
			}
			break
		}

		lines++
		if len(line) <= 16 {
			// TODO: err?
			continue
		}

		switch line[0] {
		/*case 'T':
		  if bytes.Compare(line[:6], []byte("Title:")) == 0 {
		      info.title = string(line[16:])
		  }*/
		case 'P':
			if bytes.Compare(line[:6], []byte("Pages:")) == 0 {
				j.Pages, j.Err = strconv.Atoi(string(line[16:]))
				if j.Err != nil {
					return
				}
			}
			/*case 'E':
			  if bytes.Compare(line[:10], []byte("Encrypted:")) == 0 {
			      info.encrypted = (bytes.Compare(line[16:], []byte("no")) == 0)
			      fmt.Fprintf(w, "Encrypted: %s\n", line[16:])
			  }*/
		}
	}

	if lines == 0 {
		rdr = bufio.NewReader(stderr)
		for {
			var line []byte
			line, _, err = rdr.ReadLine()
			if err != nil {
				if err != io.EOF {
					j.Err = err
					return
				}
				break
			}

			// Some systems return "Command Line Error: Incorrect password",
			// others just "Error: Incorrect password"
			if len(line) >= 25 && bytes.Compare(
				line[len(line)-25:],
				[]byte("Error: Incorrect password"),
			) == 0 {
				stderr.Close()
				j.Err = ErrPasswordRequired
				return
			}
		}
	} else {
		stderr.Close()
	}

	j.Err = cmd.Wait()
	return
}

func pdfPkpgcounter(j *Job) {
	// colorspace arg
	cs := "-cCMYK"
	if j.BW {
		cs = "-cBW"
	}

	start := time.Now()
	cmd := exec.Command("pkpgcounter",
		cs,
		//      "-r150",
		j.File,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		j.Err = err
		return
	}
	if err = cmd.Start(); err != nil {
		j.Err = err
		return
	}
	num := 0
	rdr := bufio.NewReader(stdout)
	var sum Coverage
	for {
		var line []byte
		line, _, err = rdr.ReadLine()
		if err != nil {
			if err != io.EOF {
				j.Err = err
				return
			}
			break
		}
		num++

		var c Coverage
		if j.BW {
			if c, err = inkCoverBW(line); err != nil {
				j.Err = err
				return
			}
		} else {
			if c, err = inkCoverCMYK(line); err != nil {
				j.Err = err
				return
			}

			sum.Cyan += c.Cyan
			sum.Magenta += c.Magenta
			sum.Yellow += c.Yellow
		}

		sum.Key += c.Key

		//c.print(w, num)
		//flush()
	}
	j.CMYK = sum

	if err = cmd.Wait(); err != nil {
		j.Err = err
		return
	}

	// calculation run time
	j.Runtime = time.Since(start)

	if j.Duplex == simplex {
		j.Sheets = num
	} else {
		j.Sheets = (num + 1) / 2
	}

	j.Price = sum.Price() + // ink
		(float64(num) * config.PriceFuser) + // fuser
		(float64(j.Sheets) * config.PriceSheet) // paper
	j.Total = j.Price * float64(j.Copies)
}

/*
func pdfPrice(w io.Writer, flush func(), filename, password string) (price float64, err error) {
	var sum coverage

	start := time.Now()

	// See http://www.guug.de/uptimes/2013-2/uptimes_2013-02.pdf
	// lower resolution => higher price, faster calculation
	cmd := exec.Command("gs",
		"-q",
		"-o",
		"-",
		"-sDEVICE=inkcov",
		//      "-sColorConversionStrategy=Gray",
		//      "-dProcessColorModel=/DeviceGray",
		"-r150",
		"-sPDFPassword="+password,
		filename,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}
	num := 0
	rdr := bufio.NewReader(stdout)
	for {
		var line []byte
		line, _, err = rdr.ReadLine()
		if err != nil {
			if err != io.EOF {
				return
			}
			break
		}
		num++

		var c coverage
		if c, err = inkCover(line); err != nil {
			return
		}

		sum.cyan += c.cyan
		sum.magenta += c.magenta
		sum.yellow += c.yellow
		sum.key += c.key

		c.print(w, num)
		flush()
	}
	if err = cmd.Wait(); err != nil {
		return
	}

	elapsed := time.Since(start)
	fmt.Fprintf(w, "Time elapsed: %s \n", elapsed)

	return
}*/

func inkCover(line []byte) (c Coverage, err error) {
	// Format: " 0.00000  0.00000  0.00000  0.02231 CMYK OK"
	if len(line) == 43 && bytes.Compare(line[35:], []byte(" CMYK OK")) == 0 {
		c.Cyan, err = strconv.ParseFloat(string(line[1:8]), 64)
		c.Cyan *= 100
		if err != nil {
			return
		}

		c.Magenta, err = strconv.ParseFloat(string(line[10:17]), 64)
		c.Magenta *= 100
		if err != nil {
			return
		}

		c.Yellow, err = strconv.ParseFloat(string(line[19:26]), 64)
		c.Yellow *= 100
		if err != nil {
			return
		}

		c.Key, err = strconv.ParseFloat(string(line[28:35]), 64)
		c.Key *= 100
		return
	}

	return c, ErrInvalidFormat
}

func inkCoverBW(line []byte) (c Coverage, err error) {
	// Format: "B :   7.422240%"
	if len(line) == 15 && bytes.Compare(line[:4], []byte("B : ")) == 0 {
		c.Key, err = strconv.ParseFloat(
			strings.TrimLeft(string(line[4:14]), " "),
			64,
		)
		return
	}

	return c, ErrInvalidFormat
}

func inkCoverCMYK(line []byte) (c Coverage, err error) {
	// Format: "C :   0.507586%      M :   0.594638%      Y :   0.990822%      K :   6.804527%"
	if len(line) == 78 && bytes.Compare(line[:4], []byte("C : ")) == 0 {
		c.Cyan, err = strconv.ParseFloat(
			strings.TrimLeft(string(line[4:14]), " "),
			64,
		)
		if err != nil {
			return
		}

		c.Magenta, err = strconv.ParseFloat(
			strings.TrimLeft(string(line[25:35]), " "),
			64,
		)
		if err != nil {
			return
		}

		c.Yellow, err = strconv.ParseFloat(
			strings.TrimLeft(string(line[46:56]), " "),
			64,
		)
		if err != nil {
			return
		}

		c.Key, err = strconv.ParseFloat(
			strings.TrimLeft(string(line[67:77]), " "),
			64,
		)
		return
	}

	return c, ErrInvalidFormat
}

func convertGray(w io.Writer, filename, password string) error {
	start := time.Now()
	err := exec.Command("gs",
		"-q",
		"-sOutputFile=output.pdf", // TODO: output name
		"-sDEVICE=pdfwrite",
		"-dNumRenderingThreads=2",
		//      "-sPAPERSIZE=a4",
		"-sColorConversionStrategy=Gray",
		"-sColorConversionStrategyForImages=Gray",
		"-dProcessColorModel=/DeviceGray",
		"-dPDFSETTINGS=/printer",
		"-dOverrideICC",
		//      "-dPDFUseOldCMS=false",
		//      "-dDOINTERPOLATE",
		//      "-dCompatibilityLevel=1.4",
		"-dAutoRotatePages=/None",
		"-dHaveTransparency=false",
		"-dNOPAUSE",
		"-dBATCH",
		"-dSAFER",
		//      "-dPARANOIDSAFER",
		//      "-sPDFPassword="+password,
		"-r150",
		filename,
	).Run()

	elapsed := time.Since(start)
	fmt.Fprintf(w, "Time elapsed: %s \n", elapsed)
	return err
}
