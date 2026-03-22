package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
)

// ---------------------------------------------------------------------------
// Context keys
// ---------------------------------------------------------------------------

type ctxKey string

const (
	ctxKeyUserID ctxKey = "userID"
	ctxKeyEmail  ctxKey = "email"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

type Config struct {
	EngineURL     string
	AdminUser     string
	AdminPassword string
	JWTSecret     string
	Port          string
	RateLimit     int
	RateWindow    time.Duration
}

func loadConfig() Config {
	cfg := Config{
		EngineURL:     envOr("ENGINE_URL", "http://engine:8443"),
		AdminUser:     envOr("ADMIN_USER", "admin"),
		AdminPassword: envOr("ADMIN_PASSWORD", "changeme"),
		JWTSecret:     envOr("JWT_SECRET", "shared-secret"),
		Port:          envOr("PORT", "3000"),
		RateLimit:     envOrInt("RATE_LIMIT", 15),
		RateWindow:    envOrDuration("RATE_WINDOW", time.Hour),
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envOrDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

// ---------------------------------------------------------------------------
// JWT
// ---------------------------------------------------------------------------

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func jwtAuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}
			if claims.UserID == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token missing user_id"})
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, ctxKeyEmail, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ---------------------------------------------------------------------------
// Rate Limiter
// ---------------------------------------------------------------------------

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	limit   int
	window  time.Duration
}

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
	}
}

func (rl *RateLimiter) Allow(userID string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, ok := rl.entries[userID]
	if !ok || now.After(entry.resetAt) {
		rl.entries[userID] = &rateLimitEntry{count: 1, resetAt: now.Add(rl.window)}
		return true, 0
	}
	if entry.count >= rl.limit {
		return false, time.Until(entry.resetAt)
	}
	entry.count++
	return true, 0
}

func (rl *RateLimiter) Info(userID string) (remaining int, resetAt time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, ok := rl.entries[userID]
	if !ok {
		return rl.limit, time.Now().Add(rl.window)
	}
	if time.Now().After(entry.resetAt) {
		return rl.limit, time.Now().Add(rl.window)
	}
	rem := rl.limit - entry.count
	if rem < 0 {
		rem = 0
	}
	return rem, entry.resetAt
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

type Service struct {
	cfg         Config
	rateLimiter *RateLimiter
	engineToken string
	httpClient  *http.Client
}

func NewService(cfg Config) *Service {
	return &Service{
		cfg:         cfg,
		rateLimiter: NewRateLimiter(cfg.RateLimit, cfg.RateWindow),
		httpClient:  &http.Client{},
	}
}

// Bootstrap waits for Engine, logs in, and creates an API token.
func (s *Service) Bootstrap(ctx context.Context) error {
	slog.InfoContext(ctx, "waiting for engine", "url", s.cfg.EngineURL)
	if err := s.waitForEngine(ctx); err != nil {
		return fmt.Errorf("wait for engine: %w", err)
	}
	slog.InfoContext(ctx, "engine is healthy")

	sessionToken, err := s.loginToEngine(ctx)
	if err != nil {
		return fmt.Errorf("login to engine: %w", err)
	}
	slog.InfoContext(ctx, "logged in to engine")

	apiToken, err := s.createAPIToken(ctx, sessionToken)
	if err != nil {
		return fmt.Errorf("create api token: %w", err)
	}
	s.engineToken = apiToken
	slog.InfoContext(ctx, "api token created")

	return nil
}

func (s *Service) waitForEngine(ctx context.Context) error {
	deadline := time.Now().Add(30 * time.Second)
	healthURL := s.cfg.EngineURL + "/api/v1/health"

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		resp, err := s.httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("engine not ready after 30s")
}

func (s *Service) loginToEngine(ctx context.Context) (string, error) {
	body, err := json.Marshal(map[string]string{
		"username": s.cfg.AdminUser,
		"password": s.cfg.AdminPassword,
	})
	if err != nil {
		return "", fmt.Errorf("marshal login body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.EngineURL+"/api/v1/auth/login",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed: status %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return result.Token, nil
}

func (s *Service) createAPIToken(ctx context.Context, sessionToken string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"name":   "service-proxy",
		"scopes": []string{"chat"},
	})
	if err != nil {
		return "", fmt.Errorf("marshal token body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.cfg.EngineURL+"/api/v1/auth/tokens",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sessionToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		// Token already exists from previous run — use session token instead
		slog.Info("api token already exists, using session token")
		return sessionToken, nil
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create token failed: status %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return result.Token, nil
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (s *Service) proxyChat(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(ctxKeyUserID).(string)

	allowed, retryAfter := s.rateLimiter.Allow(userID)
	if !allowed {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error":               "rate limit exceeded",
			"retry_after_seconds": int(retryAfter.Seconds()),
		})
		return
	}

	agentName := chi.URLParam(r, "agent")
	engineURL := s.cfg.EngineURL + "/api/v1/agents/" + agentName + "/chat"

	proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, engineURL, r.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build engine request"})
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+s.engineToken)

	// No timeout for SSE streaming.
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		slog.ErrorContext(r.Context(), "engine request failed", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "engine unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(resp.StatusCode)
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		w.Write(buf[:n])
		return
	}

	// Stream SSE back to the browser.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}
	flusher.Flush()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 256*1024)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(w, "%s\n", line)
		flusher.Flush()

		if r.Context().Err() != nil {
			break
		}
	}
}

func (s *Service) proxyRespond(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "session")
	engineURL := s.cfg.EngineURL + "/api/v1/sessions/" + sessionID + "/respond"

	proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, engineURL, r.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build engine request"})
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+s.engineToken)

	resp, err := s.httpClient.Do(proxyReq)
	if err != nil {
		slog.ErrorContext(r.Context(), "engine request failed", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "engine unavailable"})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
}

func (s *Service) proxyAgents(w http.ResponseWriter, r *http.Request) {
	engineURL := s.cfg.EngineURL + "/api/v1/agents"

	proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, engineURL, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build engine request"})
		return
	}
	proxyReq.Header.Set("Authorization", "Bearer "+s.engineToken)

	resp, err := s.httpClient.Do(proxyReq)
	if err != nil {
		slog.ErrorContext(r.Context(), "engine request failed", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "engine unavailable"})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
}

func (s *Service) health(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxKeyUserID).(string)

	resp := map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}

	if userID != "" {
		remaining, resetAt := s.rateLimiter.Info(userID)
		resp["rate_limit"] = map[string]any{
			"remaining": remaining,
			"limit":     s.cfg.RateLimit,
			"reset_at":  resetAt.UTC().Format(time.RFC3339),
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

func (s *Service) buildRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	auth := jwtAuthMiddleware(s.cfg.JWTSecret)

	r.Get("/api/v1/health", s.health)

	r.Route("/api/v1", func(api chi.Router) {
		api.Use(auth)
		api.Post("/chat/{agent}", s.proxyChat)
		api.Post("/respond/{session}", s.proxyRespond)
		api.Get("/agents", s.proxyAgents)
	})

	// Static files placeholder (for web-client build output).
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body><p>ByteBrew HR Assistant</p></body></html>`))
	})

	return r
}

// ---------------------------------------------------------------------------
// CORS
// ---------------------------------------------------------------------------

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	cfg := loadConfig()
	svc := NewService(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := svc.Bootstrap(ctx); err != nil {
		slog.ErrorContext(ctx, "bootstrap failed", "error", err)
		os.Exit(1)
	}

	router := svc.buildRouter()
	addr := ":" + cfg.Port

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "shutting down")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	slog.InfoContext(ctx, "service starting", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.ErrorContext(ctx, "server error", "error", err)
		os.Exit(1)
	}
}
