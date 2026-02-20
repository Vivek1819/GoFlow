package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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

// ==================== WORKER ====================

func startWorker(workerID int) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		for {
			var id int

			// Claim one job atomically
			err := db.QueryRow(`
				UPDATE jobs
				SET status = 'processing',
				    updated_at = NOW()
				WHERE id = (
					SELECT id FROM jobs
					WHERE status = 'pending'
					AND retry_count < 3
					AND run_at <= NOW()
					ORDER BY id
					LIMIT 1
					FOR UPDATE SKIP LOCKED
				)
				RETURNING id;
			`).Scan(&id)

			if err == sql.ErrNoRows {
				break
			}

			if err != nil {
				log.Println("Claim error:", err)
				break
			}

			// Fetch full job
			var job Job
			var payloadBytes []byte

			err = db.QueryRow(`
				SELECT id, type, payload, status, run_at
				FROM jobs
				WHERE id = $1
			`, id).Scan(&job.ID, &job.Type, &payloadBytes, &job.Status, &job.RunAt)

			if err != nil {
				log.Println("Fetch error:", err)
				continue
			}

			err = json.Unmarshal(payloadBytes, &job.Payload)
			if err != nil {
				log.Println("Unmarshal error:", err)
				continue
			}

			log.Printf("[Worker %d] Executing job %d\n", workerID, job.ID)

			err = executeJob(job)

			if err != nil {
				log.Println("Execution failed:", err)

				_, err = db.Exec(`
					UPDATE jobs
					SET status = CASE 
						WHEN retry_count + 1 >= 3 THEN 'failed'
						ELSE 'pending'
					END,
					retry_count = retry_count + 1,
					updated_at = NOW()
					WHERE id = $1
				`, job.ID)

				if err != nil {
					log.Println("Retry update failed:", err)
				}

				continue
			}

			// Success
			_, err = db.Exec(`
				UPDATE jobs
				SET status = 'completed',
				    updated_at = NOW()
				WHERE id = $1
			`, job.ID)

			if err != nil {
				log.Println("Completion update failed:", err)
			}
		}
	}
}

// ==================== EXECUTION ====================

func executeJob(job Job) error {
	switch job.Type {

	case "http_request":
		return executeHTTPRequest(job.Payload)

	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func executeHTTPRequest(payload map[string]interface{}) error {
	url, ok := payload["url"].(string)
	if !ok {
		return fmt.Errorf("missing url")
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
			return err
		}
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	return nil
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
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);
	`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Database ready")
}

// ==================== API ====================

func main() {
	initDB()

	workerCount := 5

	for i := 1; i <= workerCount; i++ {
		go startWorker(i)
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/jobs", jobsHandler)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
