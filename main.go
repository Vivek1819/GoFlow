package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"io"

	_ "github.com/lib/pq"
)

type Job struct {
	ID      int                    `json:"id"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
	Status  string                 `json:"status"`
	RunAt   time.Time              `json:"run_at"`
}

var db *sql.DB

const (
	maxRetries = 3
	baseDelay  = 5 * time.Second
)

const processingTimeout = 30 * time.Second

func recoverStuckJobs() {
	result, err := db.Exec(`
		UPDATE jobs
		SET status = 'pending',
		    updated_at = NOW()
		WHERE status = 'processing'
		AND updated_at < NOW() - ($1 || ' seconds')::interval
	`, int(processingTimeout.Seconds()))

	if err != nil {
		log.Println("Recovery failed:", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Recovered %d stuck jobs\n", rowsAffected)
	}
}

// ==================== WORKER ====================

func startWorker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Worker %d] Shutting down...\n", workerID)
			return
		default:
		}

		var id int

		err := db.QueryRow(`
			UPDATE jobs
			SET status = 'processing',
			    updated_at = NOW()
			WHERE id = (
				SELECT id FROM jobs
				WHERE status = 'pending'
				AND retry_count < $1
				AND run_at <= NOW()
				ORDER BY id
				LIMIT 1
				FOR UPDATE SKIP LOCKED
			)
			RETURNING id;
		`, maxRetries).Scan(&id)

		if err == sql.ErrNoRows {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if err != nil {
			log.Println("Claim error:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		processJob(workerID, id)
	}
}

func processJob(workerID int, id int) {
	var job Job
	var payloadBytes []byte

	err := db.QueryRow(`
		SELECT id, type, payload, status, run_at
		FROM jobs
		WHERE id = $1
	`, id).Scan(&job.ID, &job.Type, &payloadBytes, &job.Status, &job.RunAt)

	if err != nil {
		log.Println("Fetch error:", err)
		return
	}

	err = json.Unmarshal(payloadBytes, &job.Payload)
	if err != nil {
		log.Println("Unmarshal error:", err)
		return
	}

	log.Printf("[Worker %d] Executing job %d\n", workerID, job.ID)

	start := time.Now()

	statusCode, responseBody, execErr := executeJob(job)

	duration := time.Since(start).Milliseconds()

	// 🔴 If execution failed
	if execErr != nil {

		_, _ = db.Exec(`
			UPDATE jobs
			SET last_error = $2,
			    response_status = $3,
			    response_body = $4,
			    execution_time_ms = $5,
			    updated_at = NOW()
			WHERE id = $1
		`, job.ID, execErr.Error(), statusCode, responseBody, duration)

		handleRetry(workerID, job, execErr)
		return
	}

	// 🟢 If execution succeeded
	_, err = db.Exec(`
		UPDATE jobs
		SET status = 'completed',
		    response_status = $2,
		    response_body = $3,
		    execution_time_ms = $4,
		    last_error = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`, job.ID, statusCode, responseBody, duration)

	if err != nil {
		log.Println("Completion update failed:", err)
	}
}

// ==================== EXECUTION ====================

func executeJob(job Job) (int, []byte, error) {
	switch job.Type {
	case "http_request":
		return executeHTTPRequest(job.Payload)
	default:
		return 0, nil, fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func executeHTTPRequest(payload map[string]interface{}) (int, []byte, error) {
	url, ok := payload["url"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing url")
	}

	method := "GET"
	if m, ok := payload["method"].(string); ok {
		method = m
	}

	var bodyBytes []byte
	if body, ok := payload["body"]; ok {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	responseBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return resp.StatusCode, responseBytes,
			fmt.Errorf("http status %d", resp.StatusCode)
	}

	return resp.StatusCode, responseBytes, nil
}

// ==================== DB INIT ====================

func initDB() {
	connStr := "host=127.0.0.1 port=5433 user=goflow password=goflowpass dbname=goflowdb sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS jobs (
	id SERIAL PRIMARY KEY,
	type TEXT NOT NULL,
	payload JSONB,
	status TEXT NOT NULL,
	retry_count INT DEFAULT 0,
	run_at TIMESTAMPTZ DEFAULT NOW(),
	last_error TEXT,
	response_status INT,
	response_body JSONB,
	execution_time_ms INT,
	created_at TIMESTAMP DEFAULT NOW(),
	updated_at TIMESTAMP DEFAULT NOW()
);
	`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	createReadyIndex := `
	CREATE INDEX IF NOT EXISTS idx_jobs_ready
	ON jobs (status, run_at);
	`
	_, err = db.Exec(createReadyIndex)
	if err != nil {
		log.Fatal("Failed to create ready index:", err)
	}

	log.Println("Database ready")
}

func handleRetry(workerID int, job Job, execErr error) {
	log.Println("Execution failed:", execErr)

	var retryCount int
	err := db.QueryRow(`
		SELECT retry_count FROM jobs WHERE id = $1
	`, job.ID).Scan(&retryCount)

	if err != nil {
		log.Println("Retry fetch failed:", err)
		return
	}

	if retryCount+1 >= maxRetries {
		_, err = db.Exec(`
			UPDATE jobs
			SET status = 'failed',
			    retry_count = retry_count + 1,
			    updated_at = NOW()
			WHERE id = $1
		`, job.ID)

		if err != nil {
			log.Println("Failed to mark job failed:", err)
		}
		return
	}

	nextDelay := baseDelay * time.Duration(1<<retryCount)

	log.Printf("[Worker %d] Retrying job %d in %v\n",
		workerID, job.ID, nextDelay)

	_, err = db.Exec(`
		UPDATE jobs
		SET status = 'pending',
		    retry_count = retry_count + 1,
		    run_at = NOW() + ($2 || ' seconds')::interval,
		    updated_at = NOW()
		WHERE id = $1
	`, job.ID, int(nextDelay.Seconds()))

	if err != nil {
		log.Println("Failed scheduling retry:", err)
	}
}

func startRecoveryLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Recovery] Shutting down...")
			return
		case <-ticker.C:
			recoverStuckJobs()
		}
	}
}

// ==================== API ====================

func main() {
	initDB()
	recoverStuckJobs()

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	workerCount := 5

	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go startWorker(ctx, wg, i)
	}

	wg.Add(1)
	go startRecoveryLoop(ctx, wg)

	// Start HTTP server in goroutine
	server := &http.Server{
		Addr: ":8080",
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/jobs", jobsHandler)

	go func() {
		log.Println("Server running on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutdown signal received")

	// Stop workers
	cancel()

	// Gracefully stop HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)

	// Wait for workers
	wg.Wait()

	log.Println("Graceful shutdown complete")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodPost:
		var req Job

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.RunAt.IsZero() {
			req.RunAt = time.Now().UTC()
		}

		req.Status = "pending"

		payloadJSON, err := json.Marshal(req.Payload)
		if err != nil {
			http.Error(w, "Payload error", http.StatusInternalServerError)
			return
		}

		err = db.QueryRow(`
			INSERT INTO jobs (type, payload, status, run_at)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, req.Type, payloadJSON, req.Status, req.RunAt).Scan(&req.ID)

		if err != nil {
			http.Error(w, "Insert failed", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(req)

	case http.MethodGet:
		rows, err := db.Query(`
			SELECT id, type, payload, status, run_at
			FROM jobs
			ORDER BY id
		`)
		if err != nil {
			http.Error(w, "Query failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var jobs []Job

		for rows.Next() {
			var job Job
			var payloadBytes []byte

			err := rows.Scan(&job.ID, &job.Type, &payloadBytes, &job.Status, &job.RunAt)
			if err != nil {
				http.Error(w, "Scan failed", http.StatusInternalServerError)
				return
			}

			json.Unmarshal(payloadBytes, &job.Payload)
			jobs = append(jobs, job)
		}

		json.NewEncoder(w).Encode(jobs)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
