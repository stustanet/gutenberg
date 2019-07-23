package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"

	"github.com/julienschmidt/systemd"
)

const (
	langDE = iota
	langEN
)

var uplimit *RateLimiter

var tmpl *template.Template

var config *Config

type httpRedirector struct{}

var _ http.Handler = &httpRedirector{}

func (h *httpRedirector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Host {
	case config.HostEN:
		http.Redirect(w, r, "https://"+config.HostEN, http.StatusFound)
	default:
		http.Redirect(w, r, "https://"+config.HostDE, http.StatusFound)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	// the net/http router sucks
	if r.RequestURI != "/" {
		http.NotFound(w, r)
		return
	}

	lang := langDE
	switch r.Host {
	case config.HostEN:
		lang = langEN
	case config.HostDE:
		//
	default:
		http.Redirect(w, r, "https://"+config.HostDE, http.StatusFound)
		return
	}

	var haspaOpen bool
	haspaDone := make(chan bool)
	go IsHaspaOpen(&haspaOpen, haspaDone)

	// async. get office hours
	var officeHours []OfficeHour
	ohDone := make(chan bool)
	go NextOfficeHours(&officeHours, ohDone)

	<-ohDone
	<-haspaDone

	// execute tpl
	type tplOfficeHour struct {
		WeekDay   string
		Day       string
		Month     string
		TimeStart string
		TimeEnd   string
	}

	// config and content for the main template
	data := struct {
		Code          int
		Class         string
		Main          bool
		HaspaOpen     bool
		MaxFileSize   uint
		Error         string
		OfficeHours   []tplOfficeHour
		ResultContent template.HTML
	}{
		Code:        200,
		Main:        true,
		MaxFileSize: uint(config.MaxFileSize),
		HaspaOpen:   haspaOpen,
		OfficeHours: make([]tplOfficeHour, len(officeHours), len(officeHours)),
	}

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
			if err := tmpl.ExecuteTemplate(w, "index_en.html", data); err != nil {
				fmt.Println("get EN", err)
				http.Error(w,
					http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError,
				)
				return
			}
		//case langDE:
		default:
			if err := tmpl.ExecuteTemplate(w, "index_de.html", data); err != nil {
				fmt.Println("get DE", err)
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
	if r.ContentLength > int64(config.MaxFileSize) {
		return http.StatusRequestEntityTooLarge
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(config.MaxFileSize))

	// validate form input
	if r.ParseMultipartForm(int64(config.MaxFileSize)) != nil {
		return http.StatusBadRequest
	}

	j.IP = r.RemoteAddr
	j.BW = (r.FormValue("bw") == "bw")
	j.Password = r.FormValue("password")

	switch r.FormValue("format") {
	case "A5", "A4", "A3":
		j.Format = r.FormValue("format")
	case "":
		j.Format = "A4"
	default:
		return http.StatusBadRequest
	}

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
		fmt.Println("upload file:", err)
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
		fmt.Println("save upload:", err)
		return http.StatusInternalServerError
	}
	defer f.Close()
	io.Copy(f, file)
	file.Close()
	j.File = f.Name()

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
		fmt.Println("saveJob", err)
		return http.StatusInternalServerError
	}

	j.Total = j.Price * float64(j.Copies)

	type tplData struct {
		Job *Job
	}
	data := tplData{&j}

	switch lang {
	case langEN:
		if err := tmpl.ExecuteTemplate(w, "result_en.html", data); err != nil {
			fmt.Println("tpl EN:", err)
			return http.StatusInternalServerError
		}
	//case langDE:
	default:
		if err := tmpl.ExecuteTemplate(w, "result_de.html", data); err != nil {
			fmt.Println("tpl DE:", err)
			return http.StatusInternalServerError
		}
	}

	return http.StatusOK
}

func main() {
	noSocket := flag.Bool("no-socket", false, "Do not run under a socket.")
	flag.Parse()

	// TODO: passable config file
	//config = getConfig("/etc/ssn/gutenberg/admin-config.json")
	config, _ = getConfig("../../src/print/config.json")

	connectDB(config.Dsn)

	// get sockets passed by systemd
	var httpListener net.Listener
	var httpsListener net.Listener

	if !*noSocket {
		sockets, err := systemd.ListenWithNames()
		if err != nil {
			panic(err)
		}
		if len(sockets) != 2 {
			panic(fmt.Sprintf("expected 2 sockets, got %d", len(sockets)))
		}
		for i, socket := range sockets {
			switch name := socket.Name(); name {
			case "http":
				httpListener, err = socket.Listener()
				if err != nil {
					panic(err)
				}
			case "https":
				httpsListener, err = socket.Listener()
				if err != nil {
					panic(err)
				}
			default:
				panic(fmt.Sprintf("unexpected socket name %s (i=%d)", name, i))
			}
		}

		// redirect http requests to https
		go http.Serve(httpListener, new(httpRedirector))
	}

	// init Upload Request Ratelimiter
	uplimit = NewRateLimiter()

	// init templates
	tmpl = template.Must(template.ParseGlob("tpl/*.html"))

	http.Handle("/assets/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.ico")
	})
	http.HandleFunc("/", index)

	if *noSocket {
		http.ListenAndServe(":8000", nil)
	} else {
		http.ServeTLS(httpsListener, nil, config.TlsCert, config.TlsPrivKey)
	}
}
