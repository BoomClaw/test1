package main

import (
  "encoding/json"
  "log"
  "net"
  "net/http"
  "os"
  "strings"
  "sync"
  "time"
)

type Target struct {
  Name string `json:"name"`
  Type string `json:"type"` // http|tcp
  Addr string `json:"addr"`
}

type Status struct {
  Name      string    `json:"name"`
  Type      string    `json:"type"`
  Addr      string    `json:"addr"`
  Up        bool      `json:"up"`
  LatencyMs int64     `json:"latencyMs"`
  Message   string    `json:"message"`
  CheckedAt time.Time `json:"checkedAt"`
}

var (
  mu      sync.RWMutex
  statuses = map[string]Status{}
)

func main() {
  targets := loadTargets()
  interval := 15 * time.Second

  for _, t := range targets {
    go func(tt Target) {
      tick := time.NewTicker(interval)
      defer tick.Stop()
      checkAndStore(tt)
      for range tick.C {
        checkAndStore(tt)
      }
    }(t)
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
    _ = json.NewEncoder(w).Encode(map[string]any{"items": list, "count": len(list)})
  })

  addr := ":8080"
  log.Println("backend listening on", addr)
  log.Fatal(http.ListenAndServe(addr, cors(http.DefaultServeMux)))
}

func loadTargets() []Target {
  raw := os.Getenv("TARGETS_JSON")
  if strings.TrimSpace(raw) == "" {
    return []Target{
      {Name: "Google", Type: "http", Addr: "https://www.google.com"},
      {Name: "Cloudflare DNS", Type: "tcp", Addr: "1.1.1.1:53"},
    }
  }
  var t []Target
  if err := json.Unmarshal([]byte(raw), &t); err != nil || len(t) == 0 {
    log.Printf("invalid TARGETS_JSON, fallback defaults: %v", err)
    return []Target{{Name: "Google", Type: "http", Addr: "https://www.google.com"}}
  }
  return t
}

func checkAndStore(t Target) {
  s := Status{Name: t.Name, Type: t.Type, Addr: t.Addr, CheckedAt: time.Now()}
  start := time.Now()

  switch strings.ToLower(t.Type) {
  case "http", "https":
    client := &http.Client{Timeout: 6 * time.Second}
    req, _ := http.NewRequest(http.MethodGet, t.Addr, nil)
    resp, err := client.Do(req)
    if err != nil {
      s.Up = false
      s.Message = err.Error()
    } else {
      _ = resp.Body.Close()
      s.Up = resp.StatusCode < 500
      s.Message = resp.Status
    }
  case "tcp":
    conn, err := net.DialTimeout("tcp", t.Addr, 4*time.Second)
    if err != nil {
      s.Up = false
      s.Message = err.Error()
    } else {
      s.Up = true
      s.Message = "connected"
      _ = conn.Close()
    }
  default:
    s.Up = false
    s.Message = "unsupported type"
  }

  s.LatencyMs = time.Since(start).Milliseconds()
  mu.Lock()
  statuses[t.Name] = s
  mu.Unlock()
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
