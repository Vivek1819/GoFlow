package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
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

// -------------------- WORKER --------------------

func startWorker() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		for {
			var id int

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
				log.Println("Failed to claim job:", err)
				break
			}

			log.Println("Processing job:", id)

			time.Sleep(1 * time.Second)

			// Simulate random failure
			if rand.Intn(2) == 0 {
				log.Println("Job failed:", id)

				_, err = db.Exec(`
					UPDATE jobs
					SET status = CASE 
						WHEN retry_count + 1 >= 3 THEN 'failed'
						ELSE 'pending'
					END,
					retry_count = retry_count + 1,
					updated_at = NOW()
					WHERE id = $1
				`, id)

				if err != nil {
					log.Println("Failed to update failed job:", err)
				}

				continue
			}

			// Success
			_, err = db.Exec(`
				UPDATE jobs
				SET status = 'completed',
				    updated_at = NOW()
				WHERE id = $1
			`, id)

			if err != nil {
				log.Println("Failed to complete job:", err)
			}
		}
	}
}

// -------------------- DB INIT --------------------

func initDB() {
	connStr := "host=127.0.0.1 port=5433 user=goflow password=goflowpass dbname=goflowdb sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	log.Println("Connected to Postgres successfully")

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS jobs (
		id SERIAL PRIMARY KEY,
		type TEXT NOT NULL,
		payload JSONB,
		status TEXT NOT NULL,
		retry_count INT DEFAULT 0,
		run_at TIMESTAMP DEFAULT NOW(),
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);
	`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal("Failed to create jobs table:", err)
	}

	createIndexQuery := `
	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	`

	_, err = db.Exec(createIndexQuery)
	if err != nil {
		log.Fatal("Failed to create index:", err)
	}

	log.Println("Jobs table ready")
}

// -------------------- MAIN --------------------

func main() {
	initDB()
	rand.Seed(time.Now().UnixNano())

	go startWorker()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/jobs", jobsHandler)

	log.Println("Starting GoFlow API on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// -------------------- HANDLERS --------------------

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

		req.Status = "pending"

		// Default run_at to now if not provided
		if req.RunAt.IsZero() {
			req.RunAt = time.Now()
		}

		payloadJSON, err := json.Marshal(req.Payload)
		if err != nil {
			http.Error(w, "Failed to process payload", http.StatusInternalServerError)
			return
		}

		query := `
		INSERT INTO jobs (type, payload, status, run_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
		`

		err = db.QueryRow(query, req.Type, payloadJSON, req.Status, req.RunAt).Scan(&req.ID)
		if err != nil {
			http.Error(w, "Failed to insert job", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(req)

	case http.MethodGet:
		rows, err := db.Query(`
			SELECT id, type, payload, status, run_at
			FROM jobs
			ORDER BY id ASC
		`)
		if err != nil {
			http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var jobs []Job

		for rows.Next() {
			var job Job
			var payloadBytes []byte

			err := rows.Scan(&job.ID, &job.Type, &payloadBytes, &job.Status, &job.RunAt)
			if err != nil {
				http.Error(w, "Failed to scan job", http.StatusInternalServerError)
				return
			}

			err = json.Unmarshal(payloadBytes, &job.Payload)
			if err != nil {
				http.Error(w, "Failed to parse payload", http.StatusInternalServerError)
				return
			}

			jobs = append(jobs, job)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}