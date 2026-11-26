package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ConfigReq struct {
	BlockAfter  int    `json:"block_after"`
	BlockType   string `json:"block_type"` // "403", "429", "5xx", "soft_block", "none"
	BlockHealth bool   `json:"block_health"`
}

type StatsResp struct {
	RequestsReceived int    `json:"requests_received"`
	IndicesSeen      []int  `json:"indices_seen"`
	Blocked          bool   `json:"blocked"`
	BlockCount       int    `json:"block_count"`
	BlockType        string `json:"block_type"`
}

type TargetState struct {
	mu               sync.Mutex
	blockAfter       int
	blockType        string
	blockHealth      bool
	requestsReceived int
	indicesSeen      []int
	blocked          bool
	blockCount       int
}

var state = TargetState{
	blockAfter:  9999999,
	blockType:   "none",
	blockHealth: false,
	indicesSeen: make([]int, 0),
}

func main() {
	mux := http.NewServeMux()

	// 1. GET /scan/{index}
	mux.HandleFunc("/scan/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			idx = 0
		}

		state.mu.Lock()
		defer state.mu.Unlock()

		now := time.Now().Format("2006-01-02 15:04:05.000")

		// Check if blocking threshold reached
		if state.requestsReceived >= state.blockAfter && state.blockType != "none" {
			state.blocked = true

			switch state.blockType {
			case "403":
				state.blockCount++
				log.Printf("[%s] BLOCKED (403) GET /scan/%d | valid_reqs: %d | block_count: %d",
					now, idx, state.requestsReceived, state.blockCount)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"forbidden","action":"block"}`))
				return

			case "soft_block":
				state.blockCount++
				log.Printf("[%s] BLOCKED (soft_block) GET /scan/%d | valid_reqs: %d | block_count: %d",
					now, idx, state.requestsReceived, state.blockCount)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("cf-mitigated", "challenge")
				w.WriteHeader(http.StatusOK)
				html := `<!DOCTYPE html><html><head><title>Just a moment...</title></head>` +
					`<body><div id="challenge-platform">Please verify you are human before proceeding.</div></body></html>`
				_, _ = w.Write([]byte(html))
				return

			case "429":
				// Return 429 intermittently so rate limit triggers backoff while allowing eventual completion
				if state.blockCount < 5 || state.blockCount%2 == 0 {
					state.blockCount++
					log.Printf("[%s] RATE_LIMITED (429) GET /scan/%d | valid_reqs: %d | block_count: %d",
						now, idx, state.requestsReceived, state.blockCount)
					w.Header().Set("Retry-After", "30")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"error":"rate_limited","retry_after":30}`))
					return
				}
				state.blockCount++

			case "5xx":
				// Return 503 intermittently so outage triggers backoff while allowing eventual completion
				if state.blockCount < 5 || state.blockCount%2 == 0 {
					state.blockCount++
					log.Printf("[%s] OUTAGE (503) GET /scan/%d | valid_reqs: %d | block_count: %d",
						now, idx, state.requestsReceived, state.blockCount)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(`{"error":"service_unavailable"}`))
					return
				}
				state.blockCount++
			}
		}

		// Valid accepted scan request: increment requestsReceived and record index
		state.requestsReceived++
		state.indicesSeen = append(state.indicesSeen, idx)

		log.Printf("[%s] ACCEPTED (200) GET /scan/%d | valid_reqs: %d", now, idx, state.requestsReceived)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"index":%d}`, idx)))
	})

	// 2. GET /health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		now := time.Now().Format("2006-01-02 15:04:05.000")
		if state.blockHealth {
			log.Printf("[%s] HEALTH BLOCKED (403)", now)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"status":"blocked","reason":"health_blocked"}`))
			return
		}

		log.Printf("[%s] HEALTH OK (200)", now)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// 3. POST /control/configure
	mux.HandleFunc("/control/configure", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var cfg ConfigReq
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		state.mu.Lock()
		state.blockAfter = cfg.BlockAfter
		state.blockType = cfg.BlockType
		state.blockHealth = cfg.BlockHealth
		state.mu.Unlock()

		log.Printf("[CONTROL] Configured: block_after=%d, block_type=%s, block_health=%v",
			cfg.BlockAfter, cfg.BlockType, cfg.BlockHealth)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"configured":true}`))
	})

	// 4. POST /control/unblock
	mux.HandleFunc("/control/unblock", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		state.blockAfter = 9999999
		state.blockType = "none"
		state.blocked = false
		state.blockHealth = false
		state.mu.Unlock()

		log.Printf("[CONTROL] Unblocked target")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"unblocked":true}`))
	})

	// 5. GET /control/stats
	mux.HandleFunc("/control/stats", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		copiedIndices := make([]int, len(state.indicesSeen))
		copy(copiedIndices, state.indicesSeen)

		stats := StatsResp{
			RequestsReceived: state.requestsReceived,
			IndicesSeen:      copiedIndices,
			Blocked:          state.blocked,
			BlockCount:       state.blockCount,
			BlockType:        state.blockType,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats)
	})

	// 6. POST /control/reset
	mux.HandleFunc("/control/reset", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		state.requestsReceived = 0
		state.indicesSeen = make([]int, 0)
		state.blocked = false
		state.blockCount = 0
		state.mu.Unlock()

		log.Printf("[CONTROL] Reset stats and counters")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"reset":true}`))
	})

	log.Println("Starting testlab_blocking_target on :8090...")
	if err := http.ListenAndServe(":8090", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
