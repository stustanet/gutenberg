package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type OfficeHour struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

var (
	officeHoursCache  []OfficeHour
	officeHoursUpdate time.Time
	officeHoursMu     sync.RWMutex
)

// Saves the office hours to the given pointer and signals through chan done
// when it is done (it might need to fetch from the network first).
// This function should be called from a separate goroutine.
func NextOfficeHours(p *[]OfficeHour, done chan bool) {
	officeHoursMu.RLock()
	if int(time.Since(officeHoursUpdate)) < config.OfficeHoursMaxAge {
		*p = officeHoursCache
		officeHoursMu.RUnlock()
		done <- true
		return
	}

	// Cache is outdated. Update...
	officeHoursMu.RUnlock()
	officeHoursMu.Lock()

	// Rule #2: Always double tap.
	// The cache might have changed while we waited for the lock
	if int(time.Since(officeHoursUpdate)) < config.OfficeHoursMaxAge {
		*p = officeHoursCache
		officeHoursMu.Unlock()
		done <- true
		return
	}

	updateOfficeHours()

	*p = officeHoursCache
	officeHoursMu.Unlock()
	done <- true
}

func updateOfficeHours() {
	// Fetch appointments.json
	resp, err := http.Get(config.OfficeHoursURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Parse JSON
	var data []struct {
		Start int64
		End   int64
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return
	}

	// Unix timestamps to time.Time
	res := make([]OfficeHour, len(data), len(data))
	for i := range data {
		res[i].Start = time.Unix(data[i].Start, 0)
		res[i].End = time.Unix(data[i].End, 0)
	}
	officeHoursCache = res
}
