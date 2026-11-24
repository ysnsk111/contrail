package server

import (
	"context"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"contrail/internal/auth"
	"contrail/internal/captcha"
	"contrail/internal/config"
	"contrail/internal/database"
	"contrail/internal/mailer"
	"contrail/internal/models"
	"contrail/internal/poller"
)

type contextKey string

const (
	ctxUserKey   contextKey = "ctx_user"
	ctxAPIKeyKey contextKey = "ctx_api_key"
)

// Server coordinates HTTP endpoints, security enforcement, and web interfaces.
type Server struct {
	cfg         *config.Config
	db          *database.DB
	poller      *poller.Poller
	airportDB   *poller.AirportDB
	mailer      *mailer.Mailer
	captcha     *captcha.CaptchaManager
	capVerifier *auth.CapVerifier
	auth        *auth.AuthManager
	httpSrv     *http.Server
}

// NewServer initializes the Server.
func NewServer(cfg *config.Config, db *database.DB, p *poller.Poller) *Server {
	m := mailer.NewMailer(cfg)
	c := captcha.NewCaptchaManager(cfg.JWTSecret)
	cv := auth.NewCapVerifier(cfg.CapVerifyURL, cfg.CapSecretKey, cfg.CapSiteKey)
	a := auth.NewAuthManager(cfg, db, m)

	s := &Server{
		cfg:         cfg,
		db:          db,
		poller:      p,
		airportDB:   poller.GetAirportDB(),
		mailer:      m,
		captcha:     c,
		capVerifier: cv,
		auth:        a,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Chain: Logging -> CORS -> API Key Enforcement -> Mux
	handler := s.loggingMiddleware(s.corsMiddleware(s.apiKeyEnforcementMiddleware(mux)))

	addr := cfg.Port
	if !containsColon(addr) {
		addr = ":" + addr
	}

	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func containsColon(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return true
		}
	}
	return false
}

// Start launches the HTTP server and checks for initial live flight data.
func (s *Server) Start() error {
	log.Printf("[Server] Contrail Flight Telemetry & Radar API listening on %s (Public Base: %s)", s.httpSrv.Addr, s.cfg.APIBaseURL)
	
	// Warm up live_flights if empty
	go func() {
		time.Sleep(1 * time.Second)
		if restored, err := s.db.RestoreLiveFlightsFromLatestSnapshot(); err == nil && restored > 0 {
			log.Printf("[Server] Warmed up %d aircraft into live_flights from previous snapshot state", restored)
		}
	}()

	return s.httpSrv.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-API-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		if r.URL.Path != "/api/v1/health" && !strings.HasPrefix(r.URL.Path, "/static/") {
			log.Printf("[HTTP] %s %s - %v", r.Method, r.URL.RequestURI(), duration)
		}
	})
}

// apiKeyEnforcementMiddleware enforces that all data endpoints require a valid API key,
// while public pages (welcome page, docs, health, static, auth) remain accessible.
func (s *Server) apiKeyEnforcementMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Publicly accessible paths (no API key required)
		if path == "/" ||
			path == "/docs" ||
			path == "/terms" ||
			path == "/data-notice" ||
			strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/cap/") ||
			path == "/api/v1/health" ||
			strings.HasPrefix(path, "/api/v1/auth/") ||
			strings.HasPrefix(path, "/api/v1/portal/") ||
			strings.HasPrefix(path, "/api/v1/admin/") {
			next.ServeHTTP(w, r)
			return
		}

		// All other endpoints require authentication
		rawKey := extractAPIKey(r)
		authHeader := r.Header.Get("Authorization")

		var matchedKey *models.APIKey
		var matchedUser *models.User

		// Check if authenticated via API key
		if rawKey != "" {
			apiKey, user, err := s.db.AuthenticateKey(rawKey)
			if err != nil || apiKey == nil {
				s.respondError(w, http.StatusUnauthorized, "Invalid or unrecognized API key. Obtain a free key at "+s.cfg.APIBaseURL, time.Now())
				return
			}
			if apiKey.Status == "revoked" {
				s.respondError(w, http.StatusUnauthorized, "This API key has been revoked. Please generate a new key in your portal.", time.Now())
				return
			}
			if user.Status == "banned" {
				s.respondError(w, http.StatusForbidden, "User account has been suspended by an administrator.", time.Now())
				return
			}

			// Rate limit check and headers
			rl := s.auth.EvaluateRateLimit(apiKey, user)
			if rl.IsAdmin {
				w.Header().Set("X-RateLimit-Limit", "unlimited")
				w.Header().Set("X-RateLimit-Remaining", "unlimited")
				w.Header().Set("X-RateLimit-Reset", "0")
			} else {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.LimitRPM))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(rl.RemainingRPM))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(rl.ResetEpoch, 10))
				w.Header().Set("X-RateLimit-Limit-Daily", strconv.Itoa(rl.LimitDaily))
				w.Header().Set("X-RateLimit-Remaining-Daily", strconv.Itoa(rl.RemainingDaily))
				w.Header().Set("X-RateLimit-Daily-Reset", strconv.FormatInt(rl.DailyResetEpoch, 10))
			}

			if !rl.Allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(int64(math.Max(1, float64(rl.ResetEpoch-time.Now().Unix()))), 10))
				s.respondError(w, http.StatusTooManyRequests, rl.ErrorMessage, time.Now())
				return
			}

			// Record key usage
			s.db.RecordKeyUsage(apiKey.ID)

			matchedKey = apiKey
			matchedUser = user
		} else if strings.HasPrefix(authHeader, "Bearer ") {
			// Alternatively allow authenticated console session to view radar data
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := s.auth.VerifyJWT(tokenStr)
			if err == nil && claims != nil {
				if user, _ := s.db.GetUserByID(claims.UserID); user != nil && user.Status != "banned" {
					matchedUser = user
				}
			}
		}

		if matchedUser == nil {
			s.respondError(w, http.StatusUnauthorized, "API key required. Provide via X-API-Key header or ?api_key= query parameter. Sign up for free access at "+s.cfg.APIBaseURL, time.Now())
			return
		}

		// Inject user and key into context
		ctx := context.WithValue(r.Context(), ctxUserKey, matchedUser)
		if matchedKey != nil {
			ctx = context.WithValue(ctx, ctxAPIKeyKey, matchedKey)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractAPIKey(r *http.Request) string {
	// 1. Header: X-API-Key
	if k := r.Header.Get("X-API-Key"); strings.TrimSpace(k) != "" {
		return strings.TrimSpace(k)
	}

	// 2. Header: Authorization: Bearer ct_live_...
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ct_live_") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// 3. Query parameter: ?api_key=...
	if k := r.URL.Query().Get("api_key"); strings.TrimSpace(k) != "" {
		return strings.TrimSpace(k)
	}

	return ""
}
