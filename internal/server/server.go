package server

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"policylens/internal/policy"
)

//go:embed web/dist/*
var web embed.FS

type Config struct {
	EnginePath  string
	OllamaURL   string
	OllamaModel string
	RateLimit   int
}
type Service struct {
	Catalog    *policy.Catalog
	Engine     *policy.Engine
	Provider   *policy.Provider
	config     Config
	limiter    *limiter
	modelSlots chan struct{}
	checks     atomic.Uint64
	questions  atomic.Uint64
	requests   atomic.Uint64
}
type bucket struct {
	start time.Time
	count int
}
type limiter struct {
	mu      sync.Mutex
	entries map[string]bucket
	limit   int
}

func (l *limiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, b := range l.entries {
		if now.Sub(b.start) >= time.Minute {
			delete(l.entries, k)
		}
	}
	b := l.entries[key]
	if b.start.IsZero() {
		b.start = now
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	l.entries[key] = b
	return true
}

func New(cfg Config) (*Service, error) {
	c, err := policy.LoadCatalog()
	if err != nil {
		return nil, err
	}
	if cfg.EnginePath == "" {
		cfg.EnginePath = "bin/kyverno"
	}
	if cfg.RateLimit <= 0 {
		cfg.RateLimit = 60
	}
	s := &Service{Catalog: c, Engine: policy.NewEngine(cfg.EnginePath, c), config: cfg, limiter: &limiter{entries: map[string]bucket{}, limit: cfg.RateLimit}, modelSlots: make(chan struct{}, 2)}
	if cfg.OllamaModel != "" {
		if cfg.OllamaURL == "" {
			cfg.OllamaURL = "http://127.0.0.1:11434"
		}
		u, err := url.Parse(cfg.OllamaURL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return nil, fmt.Errorf("invalid OLLAMA_URL")
		}
		s.Provider = &policy.Provider{BaseURL: cfg.OllamaURL, Model: cfg.OllamaModel}
	}
	return s, nil
}

type requestIDKey struct{}

func idFrom(r *http.Request) string {
	value, _ := r.Context().Value(requestIDKey{}).(string)
	return value
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message, "request_id": idFrom(r)})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	media := strings.Split(r.Header.Get("Content-Type"), ";")[0]
	if media != "application/json" {
		return errors.New("Content-Type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("submit a single JSON object")
	}
	return nil
}

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		mode := "excerpts"
		if s.Provider != nil {
			mode = "ollama"
		}
		writeJSON(w, 200, map[string]any{"engine_available": s.Engine.Path != "", "engine_version": policy.EngineVersion, "answer_mode": mode, "model": s.config.OllamaModel, "policy_count": len(s.Catalog.Policies), "provenance": s.Catalog.Provenance})
	})
	mux.HandleFunc("GET /api/policies", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Catalog.Policies) })
	mux.HandleFunc("GET /api/examples", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, policy.LoadExamples()) })
	mux.HandleFunc("POST /api/check", s.check)
	mux.HandleFunc("POST /api/ask", s.ask)
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# TYPE policylens_requests_total counter\npolicylens_requests_total %d\n# TYPE policylens_checks_total counter\npolicylens_checks_total %d\n# TYPE policylens_questions_total counter\npolicylens_questions_total %d\n", s.requests.Load(), s.checks.Load(), s.questions.Load())
	})
	files, _ := fs.Sub(web, "web/dist")
	fileServer := http.FileServer(http.FS(files))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			fail(w, r, 404, "API route not found")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			fail(w, r, 405, "method not allowed")
			return
		}
		if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/assets/") {
			fail(w, r, 404, "page not found")
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	return s.middleware(mux)
}
func (s *Service) check(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Manifest string `json:"manifest"`
	}
	if err := decode(w, r, &in); err != nil {
		var max *http.MaxBytesError
		if errors.As(err, &max) {
			fail(w, r, 413, "request body exceeds 128 KiB")
		} else {
			fail(w, r, 400, "invalid JSON request: "+err.Error())
		}
		return
	}
	if len(in.Manifest) > policy.MaxManifestBytes {
		fail(w, r, 413, "manifest exceeds 64 KiB")
		return
	}
	if _, _, err := policy.NormalizeManifest(in.Manifest); err != nil {
		fail(w, r, 400, err.Error())
		return
	}
	s.checks.Add(1)
	result, err := s.Engine.Check(r.Context(), in.Manifest)
	if err != nil {
		if errors.Is(err, policy.ErrUnavailable) {
			fail(w, r, 503, err.Error())
		} else if errors.Is(err, context.DeadlineExceeded) {
			fail(w, r, 504, "policy check timed out")
		} else {
			fail(w, r, 502, "policy engine could not complete the check")
		}
		return
	}
	writeJSON(w, 200, result)
}
func (s *Service) ask(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var in struct {
		Question string `json:"question"`
		PolicyID string `json:"policy_id"`
	}
	if err := decode(w, r, &in); err != nil {
		var max *http.MaxBytesError
		if errors.As(err, &max) {
			fail(w, r, 413, "request body exceeds 128 KiB")
		} else {
			fail(w, r, 400, "invalid JSON request")
		}
		return
	}
	in.Question = strings.TrimSpace(in.Question)
	if in.Question == "" || len(in.Question) > 1000 {
		fail(w, r, 400, "question must contain 1–1000 bytes")
		return
	}
	if in.PolicyID != "" {
		if _, ok := s.Catalog.Find(in.PolicyID); !ok {
			fail(w, r, 400, "unknown policy_id")
			return
		}
	}
	s.questions.Add(1)
	evidence := s.Catalog.Retrieve(in.Question, in.PolicyID, 3)
	answer := policy.Excerpts(evidence)
	if s.Provider != nil && len(evidence) > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		select {
		case s.modelSlots <- struct{}{}:
			defer func() { <-s.modelSlots }()
		case <-ctx.Done():
			fail(w, r, 504, "model queue timed out")
			return
		}
		var err error
		answer, err = s.Provider.Generate(ctx, in.Question, evidence)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				fail(w, r, 504, "model request timed out")
			} else {
				fail(w, r, 502, "model failed or returned invalid citations; retry or use source-excerpt mode")
			}
			return
		}
	}
	answer.LatencyMS = time.Since(start).Milliseconds()
	writeJSON(w, 200, answer)
}

type loggedResponse struct {
	http.ResponseWriter
	status int
}

func (w *loggedResponse) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
func (w *loggedResponse) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	return w.ResponseWriter.Write(p)
}
func (s *Service) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		idBytes := make([]byte, 12)
		if _, err := rand.Read(idBytes); err != nil {
			fail(w, r, 500, "cannot allocate request ID")
			return
		}
		id := hex.EncodeToString(idBytes)
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id))
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'")
		w.Header().Set("Cache-Control", "no-store")
		lr := &loggedResponse{ResponseWriter: w}
		s.requests.Add(1)
		defer func() {
			if v := recover(); v != nil {
				slog.Error("request panicked", "request_id", id)
				if lr.status == 0 {
					fail(lr, r, 500, "internal server error")
				}
			}
			if lr.status == 0 {
				lr.status = 200
			}
			slog.Info("request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", lr.status, "duration_ms", time.Since(start).Milliseconds())
		}()
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/") {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			if !s.limiter.allow(host, start) {
				w.Header().Set("Retry-After", "60")
				fail(lr, r, 429, "request limit reached; retry in one minute")
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host {
					fail(lr, r, 403, "cross-origin request rejected")
					return
				}
			}
		}
		next.ServeHTTP(lr, r)
	})
}
