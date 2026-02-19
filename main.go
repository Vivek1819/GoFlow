package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type Job struct {
	ID      int                    `json:"id"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
	Status  string                 `json:"status"`
}

var jobs []Job
var jobIDCounter int
var mu sync.Mutex

func startWorker() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		mu.Lock()

		for i := range jobs {
			if jobs[i].Status == "pending" {
				log.Println("Processing job:", jobs[i].ID)
				jobs[i].Status = "completed"
			}
		}

		mu.Unlock()
	}
}

func main() {
	go startWorker()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/jobs", jobsHandler)

	log.Println("Starting GoFlow API on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodPost:
		var req Job

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		mu.Lock()

		jobIDCounter++
		req.ID = jobIDCounter
		req.Status = "pending"
		jobs = append(jobs, req)

		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)

	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		json.NewEncoder(w).Encode(jobs)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
