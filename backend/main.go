package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Target struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // http|tcp
	Addr     string `json:"addr"`
	Interval int    `json:"intervalSec"`
	Timeout  int    `json:"timeoutSec"`
}

type Status struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Addr      string    `json:"addr"`
	Up        bool      `json:"up"`
	LatencyMs int64     `json:"latencyMs"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checkedAt"`
	Uptime24h float64   `json:"uptime24h"`
}

type ProbeRecord struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Up        bool      `json:"up"`
	LatencyMs int64     `json:"latencyMs"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checkedAt"`
}

var (
	mu       sync.RWMutex
	statuses = map[string]Status{}
	db       *sql.DB
)

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatal(err)
	}

	targets := loadTargets()
	for _, t := range targets {
		if t.Interval <= 0 {
			t.Interval = 15
		}
		if t.Timeout <= 0 {
			t.Timeout = 6
		}
		go runProbeLoop(t)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.RLock()
		list := make([]Status, 0, len(statuses))
		for _, s := range statuses {
			list = append(list, s)
		}
		mu.RUnlock()
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		_ = json.NewEncoder(w).Encode(map[string]any{"items": list, "count": len(list)})
	})

	http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if strings.TrimSpace(name) == "" {
			http.Error(w, "missing name", http.StatusBadRequest)
			return
		}
		limit := 100
		rows, err := db.Query(`SELECT id,name,up,latency_ms,message,checked_at FROM probe_records WHERE name=? ORDER BY id DESC LIMIT ?`, name, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		out := make([]ProbeRecord, 0)
		for rows.Next() {
			var rec ProbeRecord
			var upInt int
			if err := rows.Scan(&rec.ID, &rec.Name, &upInt, &rec.LatencyMs, &rec.Message, &rec.CheckedAt); err == nil {
				rec.Up = upInt == 1
				out = append(out, rec)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": out, "count": len(out)})
	})

	http.HandleFunc("/api/targets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": targets})
	})

	addr := ":8080"
	log.Println("backend listening on", addr)
	log.Fatal(http.ListenAndServe(addr, cors(http.DefaultServeMux)))
}

func runProbeLoop(t Target) {
	tick := time.NewTicker(time.Duration(t.Interval) * time.Second)
	defer tick.Stop()
	checkAndStore(t)
	for range tick.C {
		checkAndStore(t)
	}
}

func loadTargets() []Target {
	raw := os.Getenv("TARGETS_JSON")
	if strings.TrimSpace(raw) == "" {
		return []Target{
			{Name: "Google", Type: "http", Addr: "https://www.google.com", Interval: 15, Timeout: 6},
			{Name: "GitHub", Type: "http", Addr: "https://github.com", Interval: 20, Timeout: 6},
			{Name: "Cloudflare DNS", Type: "tcp", Addr: "1.1.1.1:53", Interval: 15, Timeout: 4},
		}
	}
	var t []Target
	if err := json.Unmarshal([]byte(raw), &t); err != nil || len(t) == 0 {
		log.Printf("invalid TARGETS_JSON, fallback defaults: %v", err)
		return []Target{{Name: "Google", Type: "http", Addr: "https://www.google.com", Interval: 15, Timeout: 6}}
	}
	return t
}

func checkAndStore(t Target) {
	s := Status{Name: t.Name, Type: t.Type, Addr: t.Addr, CheckedAt: time.Now()}
	start := time.Now()
	err := error(nil)

	switch strings.ToLower(t.Type) {
	case "http", "https":
		err = probeHTTP(t.Addr, t.Timeout)
	case "tcp":
		err = probeTCP(t.Addr, t.Timeout)
	default:
		err = errors.New("unsupported type")
	}

	s.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		s.Up = false
		s.Message = err.Error()
	} else {
		s.Up = true
		s.Message = "ok"
	}

	_ = insertProbeRecord(s)
	s.Uptime24h = calcUptime24h(t.Name)

	mu.Lock()
	statuses[t.Name] = s
	mu.Unlock()
}

func probeHTTP(url string, timeoutSec int) error {
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return errors.New(resp.Status)
	}
	return nil
}

func probeTCP(addr string, timeoutSec int) error {
	conn, err := net.DialTimeout("tcp", addr, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func initDB() (*sql.DB, error) {
	path := os.Getenv("DB_PATH")
	if strings.TrimSpace(path) == "" {
		path = "./probe.db"
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	schema := `
CREATE TABLE IF NOT EXISTS probe_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	up INTEGER NOT NULL,
	latency_ms INTEGER NOT NULL,
	message TEXT,
	checked_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_probe_name_time ON probe_records(name, checked_at);
`
	_, err = db.Exec(schema)
	return db, err
}

func insertProbeRecord(s Status) error {
	up := 0
	if s.Up {
		up = 1
	}
	_, err := db.Exec(`INSERT INTO probe_records(name,up,latency_ms,message,checked_at) VALUES (?,?,?,?,?)`, s.Name, up, s.LatencyMs, s.Message, s.CheckedAt)
	return err
}

func calcUptime24h(name string) float64 {
	since := time.Now().Add(-24 * time.Hour)
	var total, up int
	_ = db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(up),0) FROM probe_records WHERE name=? AND checked_at>=?`, name, since).Scan(&total, &up)
	if total == 0 {
		return 100
	}
	return (float64(up) / float64(total)) * 100
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
