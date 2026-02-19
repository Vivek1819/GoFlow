package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type JobRequest struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

type JobResponse struct {
	Message string      `json:"message"`
	Job     JobRequest  `json:"job"`
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/jobs", jobsHandler)

	log.Println("Starting GoFlow API on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var jobReq JobRequest

	err := json.NewDecoder(r.Body).Decode(&jobReq)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := JobResponse{
		Message: "Job received successfully",
		Job:     jobReq,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
