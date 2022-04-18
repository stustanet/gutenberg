package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
    "os"
	"os/exec"
	"regexp"
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
	Format   string // A5, A4, A3
	BW       bool
	Pages    int
	Sheets   int
	Copies   int
	Rotated  bool
	Price    float64       // per copy
	Total    float64       // total amount
	Runtime  time.Duration // calculation run time
	Created  time.Time
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
	ErrInvalidPdfinfo   = errors.New("invalid pdf info output")
	pageSizeRegex       = regexp.MustCompile(`^Page size:\s+(?P<width>\d+\.?\d*) x (?P<height>\d+\.?\d*) pts( \((?P<format>\w+)\))?.*$`)
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
			if bytes.HasPrefix(line, []byte("Pages:")) {
				j.Pages, j.Err = strconv.Atoi(string(line[16:]))
				if j.Err != nil {
					return
				}
			} else if bytes.HasPrefix(line, []byte("Page size:")) {
				var match = pageSizeRegex.FindStringSubmatch(string(line))
				if len(match) == 0 {
					j.Err = ErrInvalidPdfinfo
					return
				}
				var width, _ = strconv.ParseFloat(match[1], 32)
				var height, _ = strconv.ParseFloat(match[2], 32)
				var widthToHeight = width / height

				if widthToHeight > 1.0 {
					j.Rotated = true
				}

				if match[4] != "" {
					j.Format = match[4]
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

func calcCost(j *Job) {
    if j.BW {
        err := convertGray(j.File, j.Password)

        if err != nil {
            j.Err = err
            return
        }
        os.Rename(j.File+"_gray.pdf", j.File)
    }
    ghostscript_ink_cov(j)
    

	j.Price = j.CMYK.Price() + // ink
		(float64(j.Pages) * config.PriceFuser) + // fuser
		(float64(j.Sheets) * config.PriceSheet) // paper
	j.Total = j.Price * float64(j.Copies)

}

func ghostscript_ink_cov(j *Job) {
    cmd := exec.Command("gs",
                        "-dSAFER",
                        "-dNOPAUSE",
                        "-dBATCH",
                        "-d",
                        "-q",
                        "-o-",
                        "-sDEVICE=ink_cov",
                //      "-sPDFPassword="+password,
                        j.File)
    start := time.Now()
    out, err := cmd.Output()
    if err != nil {
        j.Err = err
        return
    }
    
    num := 0

    var sum Coverage
    scanner := bufio.NewScanner(strings.NewReader(string(out)))
    for scanner.Scan() {
        var pageCov Coverage
        // Format: 1.000000 1.000000 1.000000 1.000000 CMYK OK
        line := scanner.Text()
        num++
        tokens := strings.Fields(line)

        pageCov.Cyan, err = strconv.ParseFloat(tokens[0], 64)
        if err != nil {
            j.Err = err
            return
        }

        pageCov.Magenta, err = strconv.ParseFloat(tokens[1], 64)
        if err != nil {
            j.Err = err
            return
        }
        
        pageCov.Yellow, err = strconv.ParseFloat(tokens[2], 64)
        if err != nil {
            j.Err = err
            return
        }
        
        pageCov.Key, err = strconv.ParseFloat(tokens[3], 64)
        if err != nil {
            j.Err = err
            return
        }
        
        sum.Cyan += pageCov.Cyan
        sum.Magenta += pageCov.Magenta
        sum.Yellow += pageCov.Yellow
        sum.Key += pageCov.Key
            // TODO: WEITERMACHEN
    }

    j.CMYK = sum
    j.Runtime = time.Since(start)

    
	if j.Duplex == simplex {
		j.Sheets = num
	} else {
		j.Sheets = (num + 1) / 2
	}

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
		j.File)

	out, err := cmd.Output()
	if err != nil {
		j.Err = err
		return
	}

	num := 0

	var sum Coverage
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
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
	}

	j.CMYK = sum

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

func inkCoverBW(line string) (c Coverage, err error) {
	// Format: "B :   7.422240%"
	if len(line) == 15 && strings.Compare(line[:4], "B : ") == 0 {
		c.Key, err = strconv.ParseFloat(
			strings.TrimLeft(string(line[4:14]), " "),
			64,
		)
		return
	}

	return c, ErrInvalidFormat
}

func inkCoverCMYK(line string) (c Coverage, err error) {
	// Format: "C :   0.507586%      M :   0.594638%      Y :   0.990822%      K :   6.804527%"
	if len(line) == 78 && strings.Compare(line[:4], "C : ") == 0 {
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

func convertGray(filename, password string) error {
	err := exec.Command("gs",
		"-q",
		"-sOutputFile="+filename+"_gray.pdf", // TODO: output name
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

	return err
}
