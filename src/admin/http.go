package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var (
	templates map[string]*template.Template
	config    *Config
)

func listenSocket() (net.Listener, error) {
	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
	if err != nil {
		return nil, err
	}

	if pid != os.Getpid() {
		return nil, errors.New("Listen PID does not match.")
	}

	if os.Getenv("LISTEN_FDS") != "1" {
		return nil, errors.New("Expected 1 socket activation fd, got " +
			os.Getenv("LISTEN_FDS"))
	}

	const fd = 3 // first systemd socket
	syscall.CloseOnExec(fd)
	return net.FileListener(os.NewFile(fd, ""))
}

func isAllowedIP(w http.ResponseWriter, r *http.Request) bool {
	for _, ip := range config.AllowedIPs {
		if ip == strings.Split(r.RemoteAddr, ":")[0] {
			return true
		}
	}

	http.Error(w, "Permission Denied - Untrusted Host", http.StatusForbidden)
	return false
}

func detail(w http.ResponseWriter, r *http.Request) {
	if !isAllowedIP(w, r) {
		return
	}
	jobs, err := listJobsDetail()

	type Data struct {
		WithSearch bool
		Jobs       []Job
	}

	data := Data{true, jobs}

	err = renderTemplate(w, "job_list_detail.html", data)
	if err != nil {
		fmt.Print(err)
	}
}

func jobs(w http.ResponseWriter, r *http.Request) {
	if !isAllowedIP(w, r) {
		return
	}
	jobs, err := listJobs()

	type Data struct {
		WithSearch    bool
		Jobs          []Job
		Printers      []Printer
		FormatOptions []string
	}

	data := Data{true, jobs, config.Printers, formats}

	err = renderTemplate(w, "job_list.html", data)
	if err != nil {
		fmt.Print(err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	type Data struct {
		WithSearch    bool
		Job           Job
		Printers      []Printer
		FormatOptions []string
		Result        bool
		Err           error
	}

	data := Data{}
	data.WithSearch = false

	if r.Method == "GET" {
		data.Result = false

		err := renderTemplate(w, "job_lookup.html", data)
		if err != nil {
			fmt.Print(err)
		}
	} else if r.Method == "POST" {
		pin := r.FormValue("pin")
		if len(pin) < 1 {
			http.Error(w, "Job PIN missing", http.StatusBadRequest)
			return
		}

		job, err := getJob(pin)
		if err != nil {
			data.Err = errors.New("Job not found for PIN " + pin)
			err = renderTemplate(w, "job_lookup.html", data)
		} else {
			data.Result = true
			data.Job = *job
			fmt.Println(job.Pin, job.Price)
			data.Printers = config.Printers
			data.FormatOptions = formats

			err = renderTemplate(w, "job_lookup.html", data)
			if err != nil {
				fmt.Print(err)
			}
		}
	} else {
		http.Error(w, "NOPE", http.StatusMethodNotAllowed)
		return
	}
}

func logs(w http.ResponseWriter, r *http.Request) {
	if !isAllowedIP(w, r) {
		return
	}
	logs, err := listLog()

	type Data struct {
		WithSearch bool
		Logs       []Log
	}

	data := Data{true, logs}

	err = renderTemplate(w, "log_list.html", data)
	if err != nil {
		fmt.Print(err)
	}
}

func print(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "NOPE", http.StatusMethodNotAllowed)
		return
	}

	pin := r.FormValue("pin")
	if len(pin) < 1 {
		http.Error(w, "Job PIN missing", http.StatusBadRequest)
		return
	}

	job, err := getJob(pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println(r.FormValue("internal"))
	job.Internal = r.FormValue("internal") == "true"

	format := r.FormValue("format")
	if format == formatA5 || format == formatA4 || format == formatA3 {
		job.Format = format
	} else {
		http.Error(w, "Invalid Format", http.StatusBadRequest)
		return
	}

	printerName := r.FormValue("printer")
	var printer *Printer
	validPrinter := false
	for i := range config.Printers {
		if config.Printers[i].Name == printerName {
			validPrinter = true
			printer = &config.Printers[i]
			break
		}
	}

	if !validPrinter {
		http.Error(w, "Invalid Printer", http.StatusBadRequest)
		return
	}

	if err = printJob(w, job, printer, config); err != nil {
		log.Println("Error:", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		_, _ = fmt.Fprintf(w, "Printed successfully")
	}
}

// Load templates on program initialisation
func initTemplates() {
	templates = make(map[string]*template.Template)

	templatesDir := "tpl/"

	layouts, err := filepath.Glob(templatesDir + "layouts/*.html")
	if err != nil {
		log.Fatal(err)
	}

	pages, err := filepath.Glob(templatesDir + "*.html")
	if err != nil {
		log.Fatal(err)
	}

	// Generate our templates map from our layouts/ and includes/ directories
	for _, page := range pages {
		files := append(layouts, page)
		templates[filepath.Base(page)] = template.Must(template.ParseFiles(files...))
	}

}

// renderTemplate is a wrapper around template.ExecuteTemplate.
func renderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	// Ensure the template exists in the map.
	tmpl, ok := templates[name]
	if !ok {
		return fmt.Errorf("The template %s does not exist.\n", name)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, "base.html", data)
}

func main() {
	noSocket := flag.Bool("no-socket", false, "Do not run as a socket")
	configFile := flag.String("config", "/etc/ssn/gutenberg/admin-config.json", "Path to config file")
	flag.Parse()

	var err error
	config, err = getConfig(*configFile)
	if err != nil {
		panic(err)
	}

	initTemplates()

	connectDB(config.Dsn)
	startCleaner(config)

	var listener net.Listener

	if !*noSocket {
		var err error
		listener, err = listenSocket()
		if err != nil {
			panic(err)
		}
	}

	http.Handle("/assets/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.ico")
	})
	http.HandleFunc("/detail", detail)
	http.HandleFunc("/logs", logs)
	http.HandleFunc("/print", print)
	http.HandleFunc("/jobs", jobs)
	http.HandleFunc("/", index)

	if *noSocket {
		http.ListenAndServe(":8080", nil)
	} else {
		http.Serve(listener, nil)
	}
}
