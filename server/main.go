package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Domain        string
	Cipher        string
	NodeName      string
	AdminSecret   string
	DBPath        string
	ManagerAddr   string
	UserPortStart int
}

func loadConfig() Config {
	c := Config{
		Domain:        os.Getenv("SS_DOMAIN"),
		Cipher:        getEnvOr("SS_CIPHER", "aes-256-gcm"),
		NodeName:      getEnvOr("SS_NAME", "Tokyo"),
		AdminSecret:   os.Getenv("ADMIN_SECRET"),
		DBPath:        getEnvOr("DB_PATH", "/data/sub.db"),
		ManagerAddr:   getEnvOr("MANAGER_ADDR", "ssserver:6001"),
		UserPortStart: 40200,
	}
	if v := os.Getenv("SS_USER_PORT_START"); v != "" {
		fmt.Sscanf(v, "%d", &c.UserPortStart)
	}
	if c.Domain == "" {
		log.Fatal("SS_DOMAIN is required")
	}
	if c.AdminSecret == "" {
		log.Fatal("ADMIN_SECRET is required")
	}
	return c
}

func getEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generatePassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func main() {
	cfg := loadConfig()

	db, err := initDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer db.Close()

	mgr := NewManager(cfg.ManagerAddr)
	h := &handler{cfg: cfg, db: db, mgr: mgr}

	// sync active tokens to ssserver on startup
	go h.syncToManager()

	// poll traffic stats every 5 minutes, enforce quota
	go func() {
		for range time.Tick(5 * time.Minute) {
			h.pollStats()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/sub/", h.handleSub)
	mux.HandleFunc("/admin/tokens", h.adminAuth(h.handleTokens))
	mux.HandleFunc("/admin/tokens/", h.adminAuth(h.handleTokenByID))

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

type handler struct {
	cfg Config
	db  *DB
	mgr *Manager
}

// syncToManager re-registers all active tokens with ssserver.
// Called on startup to recover from ssserver restarts.
func (h *handler) syncToManager() {
	tokens, err := h.db.ActiveTokens()
	if err != nil {
		log.Printf("sync: list tokens: %v", err)
		return
	}
	for _, t := range tokens {
		if err := h.mgr.AddServer(t.ServerPort, t.Password, h.cfg.Cipher); err != nil {
			log.Printf("sync: add port %d: %v", t.ServerPort, err)
		}
	}
	log.Printf("sync: registered %d active tokens", len(tokens))
}

func (h *handler) pollStats() {
	stats, err := h.mgr.Stats()
	if err != nil {
		log.Printf("stats poll: %v", err)
		return
	}
	if err := h.db.UpdateStats(stats); err != nil {
		log.Printf("stats update: %v", err)
		return
	}
	// enforce quota: remove port from ssserver for over-limit tokens
	tokens, err := h.db.ActiveTokens()
	if err != nil {
		return
	}
	for _, t := range tokens {
		if t.QuotaGB != nil && float64(t.UsedBytes) >= *t.QuotaGB*1e9 {
			if err := h.mgr.RemoveServer(t.ServerPort); err != nil {
				log.Printf("quota enforce: remove port %d: %v", t.ServerPort, err)
			} else {
				log.Printf("quota exceeded: removed port %d (%s)", t.ServerPort, t.Name)
			}
		}
	}
}

func (h *handler) adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := r.Header.Get("X-Admin-Secret")
		if secret == "" {
			secret = r.URL.Query().Get("secret")
		}
		if secret != h.cfg.AdminSecret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *handler) handleSub(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[2] != "clash.yaml" {
		http.NotFound(w, r)
		return
	}
	token := parts[1]

	t, err := h.db.GetActiveToken(token)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if t.QuotaGB != nil && float64(t.UsedBytes) >= *t.QuotaGB*1e9 {
		http.Error(w, "quota exceeded", http.StatusForbidden)
		return
	}

	yaml := renderClash(h.cfg, t.Password, t.ServerPort)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="clash.yaml"`)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprint(w, yaml)
}

func (h *handler) handleTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tokens, err := h.db.ListTokens()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		type tokenResp struct {
			ID         string   `json:"id"`
			Name       string   `json:"name"`
			Token      string   `json:"token"`
			SubURL     string   `json:"sub_url"`
			ServerPort int      `json:"server_port"`
			QuotaGB    *float64 `json:"quota_gb"`
			UsedBytes  int64    `json:"used_bytes"`
			CreatedAt  string   `json:"created_at"`
			ExpiresAt  *string  `json:"expires_at"`
			Active     bool     `json:"active"`
		}
		resp := make([]tokenResp, len(tokens))
		for i, t := range tokens {
			resp[i] = tokenResp{
				ID:         t.ID,
				Name:       t.Name,
				Token:      t.Token,
				SubURL:     fmt.Sprintf("https://%s/sub/%s/clash.yaml", h.cfg.Domain, t.Token),
				ServerPort: t.ServerPort,
				QuotaGB:    t.QuotaGB,
				UsedBytes:  t.UsedBytes,
				CreatedAt:  t.CreatedAt,
				ExpiresAt:  t.ExpiresAt,
				Active:     t.Active,
			}
		}
		writeJSON(w, http.StatusOK, resp)

	case http.MethodPost:
		var req struct {
			Name       string   `json:"name"`
			ExpiryDays *int     `json:"expires_days"`
			QuotaGB    *float64 `json:"quota_gb"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		token, err := generateToken()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		password, err := generatePassword()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		port, err := h.db.NextPort(h.cfg.UserPortStart)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := h.mgr.AddServer(port, password, h.cfg.Cipher); err != nil {
			http.Error(w, "failed to register with ssserver: "+err.Error(), http.StatusInternalServerError)
			return
		}
		t, err := h.db.CreateToken(req.Name, token, password, port, req.ExpiryDays, req.QuotaGB)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"token":       t.Token,
			"server_port": t.ServerPort,
			"sub_url":     fmt.Sprintf("https://%s/sub/%s/clash.yaml", h.cfg.Domain, t.Token),
		})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *handler) handleTokenByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}
	token := parts[2]

	switch r.Method {
	case http.MethodPatch:
		var req struct {
			QuotaGB *float64 `json:"quota_gb"` // null = unlimited
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := h.db.UpdateQuota(token, req.QuotaGB); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// re-register port if quota was raised or removed (port may have been taken down)
		if t, err := h.db.GetActiveToken(token); err == nil {
			_ = h.mgr.AddServer(t.ServerPort, t.Password, h.cfg.Cipher)
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		t, err := h.db.GetActiveToken(token)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := h.db.RevokeToken(token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := h.mgr.RemoveServer(t.ServerPort); err != nil {
			log.Printf("warn: remove port %d: %v", t.ServerPort, err)
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
