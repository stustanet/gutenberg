package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
)

const (
	langDE = iota
	langEN
)

var uplimit *RateLimiter

var (
	tplIndexDE  *template.Template
	tplIndexEN  *template.Template
	tplResultDE *template.Template
	tplResultEN *template.Template
)

// listenSocket returns a Listener to the first socket passed by systemd
func listenSocket() (net.Listener, error) {
	envPid := os.Getenv("LISTEN_PID")
	if envPid == "" {
		return nil, errors.New("LISTEN_PID not set. Check systemd socket status")
	}
	pid, err := strconv.Atoi(envPid)
	if err != nil {
		return nil, err
	}

	if pid != os.Getpid() {
		return nil, errors.New("Listen PID does not match")
	}

	if os.Getenv("LISTEN_FDS") != "1" {
		return nil, errors.New("Expected 1 socket activation fd, got " +
			os.Getenv("LISTEN_FDS"))
	}

	const fd = 3 // first systemd socket
	syscall.CloseOnExec(fd)
	return net.FileListener(os.NewFile(fd, ""))
}

func index(w http.ResponseWriter, r *http.Request) {
	// the net/http router sucks
	if r.RequestURI != "/" {
		http.NotFound(w, r)
		return
	}

	lang := langDE
	switch r.Host {
	case hostEN:
		lang = langEN
	case hostDE:
		//
	default:
		http.Redirect(w, r, "http://"+hostDE, http.StatusFound)
		return
	}

	// config and content for the main template
	data := struct {
		Code          int
		Class         string
		Main          bool
		MaxFileSize   uint
		Error         string
		ResultContent template.HTML
	}{Code: 200, Main: true, MaxFileSize: MaxFileSize}

	// DO NOT CHANGE THE ORDER!
	// we use fallthrough here!
	switch r.Method {
	case "POST":
		// check if it is an AJAX request or not. Yes, we are that sneaky.
		if r.FormValue("MAX_FILE_SIZE") != "" {
			// ugh... no AJAX. We have to buffer the result in order to
			// embed it in the index template
			rc, code := bufferUpload(w, r, lang)
			if code == http.StatusOK {
				data.ResultContent = template.HTML(rc)
			} else {
				data.Code = code
				data.Error = http.StatusText(code)
			}
			data.Main = false
		} else {
			// AJAX request
			if code := upload(w, r, lang); code >= 400 {
				http.Error(w,
					http.StatusText(code),
					code,
				)
			}
			return
		}

		fallthrough

	case "GET":
		if data.Code != http.StatusOK {
			w.WriteHeader(data.Code)
		}
		switch lang {
		case langEN:
			if err := tplIndexEN.Execute(w, data); err != nil {
				fmt.Println(err)
				http.Error(w,
					http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError,
				)
				return
			}
		//case langDE:
		default:
			if err := tplIndexDE.Execute(w, data); err != nil {
				fmt.Println(err)
				http.Error(w,
					http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError,
				)
				return
			}
		}

	default:
		http.Error(w,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed,
		)
		return
	}
}

func bufferUpload(w http.ResponseWriter, r *http.Request, lang int) (string, int) {
	rb := NewResponseBuffer(w)
	code := upload(rb, r, lang)
	return string(rb.Bytes()), code
}

func upload(w http.ResponseWriter, r *http.Request, lang int) int {
	var j Job // contains info about the print job
	var err error

	// simple rate limiting
	if ip, _, _ := net.SplitHostPort(r.RemoteAddr); !uplimit.Request(ip) {
		return 429 // TooManyRequests
	}

	// limit file size
	if r.ContentLength > MaxFileSize {
		return http.StatusRequestEntityTooLarge
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxFileSize)

	// validate form input
	if r.ParseMultipartForm(MaxFileSize) != nil {
		return http.StatusBadRequest
	}

	j.IP = r.RemoteAddr
	j.BW = (r.FormValue("bw") == "bw")
	j.Password = r.FormValue("password")

	switch r.FormValue("duplex") {
	case "no":
		j.Duplex = simplex
	case "long":
		j.Duplex = duplexLong
	case "short":
		j.Duplex = duplexShort
	default:
		return http.StatusBadRequest
	}

	j.Copies, err = strconv.Atoi(r.FormValue("copies"))
	if err != nil || j.Copies < 1 || j.Copies > 99 {
		fmt.Println(err)
		return http.StatusBadRequest
	}

	file, fileheader, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		return http.StatusBadRequest
	}
	j.Name = fileheader.Filename

	// save file in upload folder
	f, err := ioutil.TempFile("upload/", "")
	if err == nil {
		err = f.Chmod(0660)
	}
	if err != nil {
		file.Close()
		fmt.Println(err)
		return http.StatusInternalServerError
	}
	defer f.Close()
	io.Copy(f, file)
	file.Close()
	j.File = f.Name()

	// async. get office hours
	var officeHours []OfficeHour
	ohDone := make(chan bool)
	go NextOfficeHours(&officeHours, ohDone)

	// get basic PDF info
	if pdfInfo(&j); j.Err != nil {
		if j.Err == ErrPasswordRequired {
			fmt.Fprintln(w, "Incorrect Password!")
			return http.StatusOK
		}

	} else {
		// calculate price
		pdfPkpgcounter(&j)
	}

	// save print job to DB
	if saveJob(&j) != nil {
		fmt.Println(err)
		return http.StatusInternalServerError
	}

	j.Total = j.Price * float64(j.Copies)

	// Wait for office hours goroutine
	<-ohDone

	// execute tpl
	type tplOfficeHour struct {
		WeekDay   string
		Day       string
		Month     string
		TimeStart string
		TimeEnd   string
	}
	type tplData struct {
		Job         *Job
		OfficeHours []tplOfficeHour
	}
	data := tplData{&j, make([]tplOfficeHour, len(officeHours), len(officeHours))}
	for i, oh := range officeHours {
		wd := oh.Start.Format("Monday")
		mo := oh.Start.Format("January")

		if lang == langDE {
			switch wd {
			case "Monday":
				wd = "Montag"
			case "Tuesday":
				wd = "Dienstag"
			case "Wednesday":
				wd = "Mittwoch"
			case "Thursday":
				wd = "Donnerstag"
			case "Friday":
				wd = "Freitag"
			case "Saturday":
				wd = "Samstag"
			case "Sunday":
				wd = "Sonntag"
			}

			switch mo {
			case "January":
				mo = "Januar"
			case "February":
				mo = "February"
			case "March":
				mo = "M&auml;rz"
			//case "April":
			case "May":
				mo = "Mai"
			case "June":
				mo = "Juni"
			case "July":
				mo = "Juli"
				//case "August":
				//case "September":
			case "October":
				mo = "Oktober"
			//case "November":
			case "December":
				mo = "Dezember"
			}

		}

		data.OfficeHours[i] = tplOfficeHour{
			WeekDay:   wd,
			Day:       oh.Start.Format("02"),
			Month:     mo,
			TimeStart: oh.Start.Format("15:04"),
			TimeEnd:   oh.End.Format("15:04"),
		}
	}

	switch lang {
	case langEN:
		if err := tplResultEN.Execute(w, data); err != nil {
			fmt.Println(err)
			return http.StatusInternalServerError
		}
	//case langDE:
	default:
		if err := tplResultDE.Execute(w, data); err != nil {
			fmt.Println(err)
			return http.StatusInternalServerError
		}
	}

	return http.StatusOK
}

func main() {
	listener, err := listenSocket()
	if err != nil {
		panic(err)
	}

	// init Upload Request Ratelimiter
	uplimit = NewRateLimiter()

	// init templates
	tplIndexDE, err = template.ParseFiles("tpl/index_de.html")
	if err != nil {
		panic(err)
	}
	tplIndexEN, err = template.ParseFiles("tpl/index_en.html")
	if err != nil {
		panic(err)
	}
	tplResultDE, err = template.ParseFiles("tpl/result_de.html")
	if err != nil {
		panic(err)
	}
	tplResultEN, err = template.ParseFiles("tpl/result_en.html")
	if err != nil {
		panic(err)
	}

	http.Handle("/assets/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.ico")
	})
	http.HandleFunc("/", index)
	http.Serve(listener, nil)
}
