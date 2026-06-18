package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Domain        string
	Cipher        string
	NodeName      string
	AdminSecret   string // used only to seed the owner account on first run
	JWTSecret     []byte
	DBPath        string
	ManagerAddr   string
	UserPortStart int
}

func loadConfig() Config {
	c := Config{
		Domain:      os.Getenv("SS_DOMAIN"),
		Cipher:      getEnvOr("SS_CIPHER", "aes-256-gcm"),
		NodeName:    getEnvOr("SS_NAME", "Tokyo"),
		AdminSecret: os.Getenv("ADMIN_SECRET"),
		DBPath:      getEnvOr("DB_PATH", "/data/sub.db"),
		ManagerAddr: getEnvOr("MANAGER_ADDR", "ssserver:6001"),
		UserPortStart: 40200,
	}
	if v := os.Getenv("SS_USER_PORT_START"); v != "" {
		fmt.Sscanf(v, "%d", &c.UserPortStart)
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = c.AdminSecret // fall back to ADMIN_SECRET if not set
	}
	c.JWTSecret = []byte(jwtSecret)
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

	// seed owner account on first run
	if n, _ := db.AdminCount(); n == 0 {
		if err := db.CreateAdmin("admin", cfg.AdminSecret, true); err != nil {
			log.Fatalf("seed admin: %v", err)
		}
		log.Println("created owner account: admin")
	}

	mgr := NewManager(cfg.ManagerAddr)
	h := &handler{cfg: cfg, db: db, mgr: mgr, statBase: make(map[int]int64)}

	go h.syncToManager()
	go func() {
		for range time.Tick(5 * time.Minute) {
			h.pollStats()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/sub/", h.handleSub)
	mux.HandleFunc("/admin/login", h.handleLogin)
	mux.HandleFunc("/admin/tokens", h.jwtAuth(false, h.handleTokens))
	mux.HandleFunc("/admin/tokens/", h.jwtAuth(false, h.handleTokenByID))
	mux.HandleFunc("/admin/admins", h.jwtAuth(true, h.handleAdmins))
	mux.HandleFunc("/admin/admins/", h.jwtAuth(true, h.handleAdminByID))

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

type handler struct {
	cfg        Config
	db         *DB
	mgr        *Manager
	statBaseMu sync.Mutex
	statBase   map[int]int64
}

func (h *handler) syncToManager() {
	tokens, err := h.db.ActiveTokens()
	if err != nil {
		log.Fatalf("sync: list tokens: %v", err)
	}
	for _, t := range tokens {
		if err := h.mgr.AddServer(t.ServerPort, t.Password, h.cfg.Cipher); err != nil {
			log.Fatalf("sync: add port %d: %v", t.ServerPort, err)
		}
	}
	log.Printf("sync: registered %d active tokens", len(tokens))
	if base, err := h.mgr.Stats(); err == nil {
		h.statBaseMu.Lock()
		h.statBase = base
		h.statBaseMu.Unlock()
		log.Printf("sync: baseline stats captured for %d ports", len(base))
	}
}

func (h *handler) pollStats() {
	stats, err := h.mgr.Stats()
	if err != nil {
		log.Printf("stats poll: %v", err)
		return
	}

	h.statBaseMu.Lock()
	increments := make(map[int]int64, len(stats))
	for port, raw := range stats {
		increments[port] = raw - h.statBase[port]
		if increments[port] < 0 {
			increments[port] = 0
		}
	}
	h.statBaseMu.Unlock()

	if err := h.db.AddStats(increments); err != nil {
		log.Printf("stats update: %v", err)
		// don't update baseline — keep it so the increments are retried next poll
		// continue to enforce quota using stale used_bytes
	} else {
		h.statBaseMu.Lock()
		h.statBase = stats
		h.statBaseMu.Unlock()
	}

	tokens, err := h.db.ActiveTokens()
	if err != nil {
		return
	}
	for _, t := range tokens {
		if t.QuotaGB != nil && float64(t.UsedBytes) >= *t.QuotaGB*1e9 {
			if err := h.mgr.RemoveServer(t.ServerPort); err != nil {
				log.Printf("quota enforce: remove port %d: %v", t.ServerPort, err)
			} else {
				log.Printf("quota exceeded: suspended port %d (%s)", t.ServerPort, t.Name)
			}
			if err := h.db.SuspendToken(t.Token); err != nil {
				log.Printf("quota enforce: suspend token %s: %v", t.Token, err)
			}
		}
	}

	// remove expired tokens from ssmanager
	expired, err := h.db.ExpiredActiveTokens()
	if err != nil {
		log.Printf("expired tokens query: %v", err)
		return
	}
	for _, t := range expired {
		if err := h.mgr.RemoveServer(t.ServerPort); err != nil {
			log.Printf("expiry enforce: remove port %d: %v", t.ServerPort, err)
		} else {
			log.Printf("token expired: removed port %d (%s)", t.ServerPort, t.Name)
		}
		if err := h.db.SuspendToken(t.Token); err != nil {
			log.Printf("expiry enforce: suspend token %s: %v", t.Token, err)
		}
	}
}

// --- auth ---

type claims struct {
	Username string `json:"username"`
	IsOwner  bool   `json:"is_owner"`
	jwt.RegisteredClaims
}

func (h *handler) signJWT(username string, isOwner bool) (string, error) {
	c := claims{
		Username: username,
		IsOwner:  isOwner,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(h.cfg.JWTSecret)
}

func (h *handler) parseJWT(r *http.Request) (*claims, error) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, errors.New("missing token")
	}
	tok, err := jwt.ParseWithClaims(strings.TrimPrefix(auth, "Bearer "), &claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return h.cfg.JWTSecret, nil
		})
	if err != nil || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	c, ok := tok.Claims.(*claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return c, nil
}

// jwtAuth middleware. ownerOnly=true requires is_owner claim.
func (h *handler) jwtAuth(ownerOnly bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := h.parseJWT(r)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if ownerOnly && !c.IsOwner {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (h *handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	admin, err := h.db.VerifyAdmin(req.Username, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	tok, err := h.signJWT(admin.Username, admin.IsOwner)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":    tok,
		"username": admin.Username,
		"is_owner": admin.IsOwner,
	})
}

// --- admin management (owner only) ---

func (h *handler) handleAdmins(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		admins, err := h.db.ListAdmins()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, admins)

	case http.MethodPost:
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := h.db.CreateAdmin(req.Username, req.Password, false); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *handler) handleAdminByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 {
		http.NotFound(w, r)
		return
	}
	username := parts[2]
	if err := h.db.DeleteAdmin(username); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found or cannot delete owner", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- token management ---

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
			UpdatedAt  string   `json:"updated_at"`
			ExpiresAt  *string  `json:"expires_at"`
			Active     bool     `json:"active"`
			Suspended  bool     `json:"suspended"`
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
				UpdatedAt:  t.UpdatedAt,
				ExpiresAt:  t.ExpiresAt,
				Active:     t.Active,
				Suspended:  t.Suspended,
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
		// seed baseline for this port so the first poll doesn't count pre-existing bytes
		if base, err := h.mgr.Stats(); err == nil {
			h.statBaseMu.Lock()
			if _, exists := h.statBase[port]; !exists {
				h.statBase[port] = base[port]
			}
			h.statBaseMu.Unlock()
		}
		t, err := h.db.CreateToken(req.Name, token, password, port, req.ExpiryDays, req.QuotaGB)
		if err != nil {
			// roll back ssmanager registration to avoid port leak
			if rmErr := h.mgr.RemoveServer(port); rmErr != nil {
				log.Printf("create token: rollback remove port %d: %v", port, rmErr)
			}
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
			QuotaGB *float64 `json:"quota_gb"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if err := h.db.UpdateQuota(token, req.QuotaGB); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// re-register with ssmanager if suspension was lifted
		if t, err := h.db.GetActiveToken(token); err == nil {
			if err := h.mgr.AddServer(t.ServerPort, t.Password, h.cfg.Cipher); err != nil {
				log.Printf("quota update: re-register port %d: %v", t.ServerPort, err)
			}
		}
		w.WriteHeader(http.StatusNoContent)

	case http.MethodDelete:
		// try active token first for port cleanup, fall back to any token for hard delete
		t, err := h.db.GetToken(token)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if t.Active && !t.Suspended {
			if err := h.mgr.RemoveServer(t.ServerPort); err != nil {
				log.Printf("warn: remove port %d: %v", t.ServerPort, err)
			}
		}
		if err := h.db.DeleteToken(token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
