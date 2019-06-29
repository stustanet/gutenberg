package main

import (
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
)

func listenSocket() (net.Listener, error) {
	pid, err := strconv.Atoi(os.Getenv("LISTEN_PID"))
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

func checkmark(b bool) string {
	if b {
		return "&#x2713;"
	}
	return "&#x2717;"
}

func detail(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/job_list_detail.html"))

	data := listJobsDetail()

	err := tmpl.Execute(w, data)
	if err != nil {
		// do something
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/job_list.html"))

	data := listJobs()

	err := tmpl.Execute(w, data)
	if err != nil {
		// do something
	}
}

func logs(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/log_list.html"))

	data := listLog()

	err := tmpl.Execute(w, data)
	if err != nil {
		// do something
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
	job.Internal = r.FormValue("internal") == "1"

	if err = printJob(w, job); err != nil {
		log.Println("Error:", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	connectDB()
	startCleaner()

	//listener, err := listenSocket()
	//if err != nil {
	//	panic(err)
	//}

	http.Handle("/assets/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.ico")
	})
	http.HandleFunc("/detail", detail)
	http.HandleFunc("/log", logs)
	http.HandleFunc("/print", print)
	http.HandleFunc("/", index)
	//http.Serve(listener, nil)
	http.ListenAndServe(":8080", nil)
}
