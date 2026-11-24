package server

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"contrail/internal/models"
)

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Web Frontend & Static Assets
	mux.HandleFunc("/", s.handleRoot)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/docs", s.handleDocs)
	mux.HandleFunc("/terms", s.handleTerms)
	mux.HandleFunc("/console", s.handleRoot)
	mux.HandleFunc("/data-notice", s.handleDataNotice)

	// Cap Proxy & Config
	mux.HandleFunc("/api/v1/auth/cap-config", s.handleCapConfig)
	if capURL, err := url.Parse("http://127.0.0.1:13000"); err == nil {
		capProxy := httputil.NewSingleHostReverseProxy(capURL)
		mux.Handle("/cap/", http.StripPrefix("/cap", capProxy))
	}

	// Authentication & Captcha (Public)
	mux.HandleFunc("/api/v1/auth/captcha", s.handleCaptcha)
	mux.HandleFunc("/api/v1/auth/send-code", s.handleSendCode)
	mux.HandleFunc("/api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("/api/v1/auth/me", s.handleMe)
	mux.HandleFunc("/api/v1/auth/password", s.handlePasswordUpdate)

	// User Portal (API Keys & Telemetry)
	mux.HandleFunc("/api/v1/portal/keys", s.handlePortalKeys)
	mux.HandleFunc("/api/v1/portal/keys/", s.handleRevokeKey)
	mux.HandleFunc("/api/v1/portal/stats", s.handlePortalStats)

	// Administrator Console
	mux.HandleFunc("/api/v1/admin/users", s.handleAdminUsers)
	mux.HandleFunc("/api/v1/admin/users/", s.handleAdminUserRouting)
	mux.HandleFunc("/api/v1/admin/field-restrictions", s.handleAdminFieldRestrictions)
	mux.HandleFunc("/api/v1/admin/registration-status", s.handleAdminRegistrationStatus)
	mux.HandleFunc("/api/v1/admin/alerts", s.handleAdminAlerts)
	mux.HandleFunc("/api/v1/admin/upstream-logs", s.handleAdminUpstreamLogs)
	mux.HandleFunc("/api/v1/admin/upstream-status", s.handleAdminUpstreamStatus)
	mux.HandleFunc("/api/v1/admin/send-test-alert", s.handleAdminSendTestAlert)

	// Data Endpoints (Guarded by API Key Enforcement)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/flights", s.handleFlights)
	mux.HandleFunc("/api/v1/flights/", s.handleFlightDetail)
	mux.HandleFunc("/api/v1/history", s.handleHistorySearch)
	mux.HandleFunc("/api/v1/emergencies", s.handleEmergencies)
	mux.HandleFunc("/api/v1/diversions", s.handleDiversions)
	mux.HandleFunc("/api/v1/airports/", s.handleAirports)
	mux.HandleFunc("/api/v1/stats", s.handleStats)
	mux.HandleFunc("/api/v1/query", s.handleCustomQuery)
}

func (s *Server) respondJSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}, total, filtered, page, limit int, start time.Time) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	durationMs := math.Round(float64(time.Since(start).Microseconds())/10.0) / 100.0

	var pagination *models.Pagination
	if limit > 0 {
		totalPages := int(math.Ceil(float64(filtered) / float64(limit)))
		if totalPages == 0 && filtered > 0 {
			totalPages = 1
		}
		pagination = &models.Pagination{
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
			TotalItems: filtered,
		}
	}

	resp := models.APIResponse{
		APIVersion:    "v1.0",
		BaseURL:       s.cfg.APIBaseURL,
		GeneratedAt:   time.Now().UTC().Truncate(time.Second),
		ExecutionMs:   durationMs,
		Status:        "success",
		Attribution:   models.DefaultAttribution(s.cfg.APIBaseURL),
		TotalRecords:  total,
		FilteredCount: filtered,
		Pagination:    pagination,
		Data:          data,
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) respondError(w http.ResponseWriter, statusCode int, message string, start time.Time) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	durationMs := math.Round(float64(time.Since(start).Microseconds())/10.0) / 100.0

	resp := models.APIResponse{
		APIVersion:   "v1.0",
		BaseURL:      s.cfg.APIBaseURL,
		GeneratedAt:  time.Now().UTC().Truncate(time.Second),
		ExecutionMs:  durationMs,
		Status:       "error",
		ErrorMessage: message,
		Attribution:  models.DefaultAttribution(s.cfg.APIBaseURL),
	}

	json.NewEncoder(w).Encode(resp)
}

func readProjectFile(relPath string) ([]byte, error) {
	candidates := []string{
		relPath,
		filepath.Join("..", relPath),
		filepath.Join("..", "..", relPath),
	}
	for _, c := range candidates {
		if data, err := os.ReadFile(c); err == nil {
			return data, nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	// If browser requests HTML or standard navigation, serve modern web portal
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") || (!strings.Contains(accept, "application/json") && r.URL.Query().Get("format") != "json") {
		htmlBytes, err := readProjectFile("web/templates/index.html")
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(htmlBytes)
			return
		}
	}

	start := time.Now()
	endpoints := map[string]string{
		"health":         s.cfg.APIBaseURL + "/api/v1/health",
		"flights":        s.cfg.APIBaseURL + "/api/v1/flights",
		"flight_detail":  s.cfg.APIBaseURL + "/api/v1/flights/{callsign_or_hex}",
		"emergencies":    s.cfg.APIBaseURL + "/api/v1/emergencies",
		"diversions":     s.cfg.APIBaseURL + "/api/v1/diversions",
		"airports":       s.cfg.APIBaseURL + "/api/v1/airports/{code}",
		"global_stats":   s.cfg.APIBaseURL + "/api/v1/stats",
		"custom_query":   s.cfg.APIBaseURL + "/api/v1/query (POST JSON)",
		"documentation":  s.cfg.APIBaseURL + "/docs",
	}

	data := map[string]interface{}{
		"project":        "Contrail - Near-Real-Time Flight Data API",
		"description":    "Global ADS-B flight tracking, route enrichment, transponder emergency monitoring, and diversion prediction service.",
		"endpoints":      endpoints,
		"license_notice": "Code: GPL-3.0. Data is collected from third-party sources and redistributed as-is under each source's terms: non-commercial use only, attribution required. See DATA_NOTICE.md.",
		"attribution":    "The OpenSky Network (opensky-network.org); adsb.lol (adsb.lol); adsb.fi (adsb.fi); route data: adsbdb.com, David Taylor and Jim Mason; airports: OurAirports.",
	}

	s.respondJSON(w, r, http.StatusOK, data, 1, 1, 1, 1, start)
}

// handleDataNotice serves the dedicated modern HTML compliance template web/templates/data_notice.html
// or falls back to raw text / markdown if specifically requested.
func (s *Server) handleDataNotice(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	accept := r.Header.Get("Accept")

	// If client explicitly requests raw markdown or plain text
	if format == "raw" || format == "text" || format == "md" || (strings.Contains(accept, "text/plain") && !strings.Contains(accept, "text/html")) {
		data, err := readProjectFile("DATA_NOTICE.md")
		if err == nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write(data)
			return
		}
	}

	htmlBytes, err := readProjectFile("web/templates/data_notice.html")
	if err != nil {
		// Fallback to DATA_NOTICE.md if template cannot be read
		data, err2 := readProjectFile("DATA_NOTICE.md")
		if err2 != nil {
			http.Error(w, "DATA_NOTICE not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlBytes)
}

// handleDocs serves the standalone interactive API documentation template web/templates/docs.html
func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	htmlBytes, err := readProjectFile("web/templates/docs.html")
	if err != nil {
		s.handleRoot(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlBytes)
}

// handleTerms serves the standalone Terms of Service and Compliance page web/templates/terms.html
func (s *Server) handleTerms(w http.ResponseWriter, r *http.Request) {
	htmlBytes, err := readProjectFile("web/templates/terms.html")
	if err != nil {
		// Fallback to data_notice
		s.handleDataNotice(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlBytes)
}

func (s *Server) applyStatsFiltering(stats map[string]interface{}, user *models.User) map[string]interface{} {
	if user != nil && user.Role == "admin" {
		return stats
	}
	allowedMap := s.auth.GetAllowedFieldsMap()
	filtered := make(map[string]interface{})
	for k, v := range stats {
		filtered[k] = v
	}
	if !allowedMap["route"] {
		delete(filtered, "top_origins")
		delete(filtered, "top_destinations")
	}
	if !allowedMap["telemetry"] {
		delete(filtered, "airborne_count")
		delete(filtered, "on_ground_count")
	}
	if !allowedMap["emergency"] {
		delete(filtered, "emergency_count")
		delete(filtered, "emergencies")
	}
	if !allowedMap["diversion_analysis"] {
		delete(filtered, "predicted_diversion_count")
		delete(filtered, "diverting")
		delete(filtered, "diverting_count")
	}
	if !allowedMap["metadata"] {
		delete(filtered, "tracked_observed_count")
		delete(filtered, "tracked_estimated_count")
		delete(filtered, "tracked_stale_count")
	}
	if !allowedMap["aircraft"] {
		delete(filtered, "total_tracked")
	}
	return filtered
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	stats, _ := s.db.GetGlobalStats()
	pollerStatus := s.poller.GetStatus()
	pauseInfo := s.poller.GetUpstreamPauseInfo()

	meanKm, p95Km, samples, _ := s.db.GetExtrapolationAccuracyStats(24)

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	filteredStats := s.applyStatsFiltering(stats, user)

	data := map[string]interface{}{
		"service":      "Contrail API",
		"status":       "operational",
		"timestamp":    time.Now().UTC().Truncate(time.Second),
		"poller":       pollerStatus,
		"flight_stats": filteredStats,
		"accuracy_stats_24h": map[string]interface{}{
			"mean_displacement_error_km": meanKm,
			"p95_displacement_error_km":  p95Km,
			"total_calibrated_samples":   samples,
		},
		"upstream_pause": pauseInfo,
	}

	s.respondJSON(w, r, http.StatusOK, data, 1, 1, 1, 1, start)
}

func (s *Server) handleFlights(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	query := r.URL.Query()

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	if user == nil || user.Role != "admin" {
		allowedMap := s.auth.GetAllowedFieldsMap()
		if !allowedMap["emergency"] && query.Get("emergency") != "" {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by emergency requires 'emergency' permission", start)
			return
		}
		if !allowedMap["diversion_analysis"] && query.Get("diverting") != "" {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by diversion requires 'diversion_analysis' permission", start)
			return
		}
		if !allowedMap["route"] && (query.Get("origin") != "" || query.Get("dest") != "") {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by origin or destination requires 'route' permission", start)
			return
		}
		if !allowedMap["flight"] && query.Get("callsign") != "" {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by callsign requires 'flight' permission", start)
			return
		}
		if !allowedMap["aircraft"] && query.Get("hex") != "" {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by hex requires 'aircraft' permission", start)
			return
		}
		if !allowedMap["telemetry"] && (query.Get("min_alt") != "" || query.Get("max_alt") != "" || query.Get("bbox") != "") {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by altitude or bounding box requires 'telemetry' permission", start)
			return
		}
	}

	filter := models.FlightQueryFilter{
		CallsignPrefix:  query.Get("callsign"),
		Hex:             query.Get("hex"),
		Squawk:          query.Get("squawk"),
		OriginICAO:      query.Get("origin"),
		DestinationICAO: query.Get("dest"),
	}

	if val := query.Get("emergency"); val != "" {
		b := (val == "true" || val == "1")
		filter.IsEmergency = &b
	}

	if val := query.Get("diverting"); val != "" {
		b := (val == "true" || val == "1")
		filter.IsDiverting = &b
	}

	if minAltStr := query.Get("min_alt"); minAltStr != "" {
		if v, err := strconv.ParseFloat(minAltStr, 64); err == nil {
			filter.MinAltitudeFt = &v
		}
	}

	if maxAltStr := query.Get("max_alt"); maxAltStr != "" {
		if v, err := strconv.ParseFloat(maxAltStr, 64); err == nil {
			filter.MaxAltitudeFt = &v
		}
	}

	if bboxStr := query.Get("bbox"); bboxStr != "" {
		parts := strings.Split(bboxStr, ",")
		if len(parts) != 4 {
			s.respondError(w, http.StatusBadRequest, "Invalid bbox parameter format. Expected 4 comma-separated numbers: minLat,minLon,maxLat,maxLon", start)
			return
		}
		minLat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		minLon, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		maxLat, err3 := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		maxLon, err4 := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid bbox parameter: coordinates must be valid numbers", start)
			return
		}
		if minLat < -90 || maxLat > 90 || minLon < -180 || maxLon > 180 || minLat > maxLat {
			s.respondError(w, http.StatusBadRequest, "Invalid bbox parameter range: latitude [-90, 90], longitude [-180, 180], minLat <= maxLat", start)
			return
		}
		filter.BoundingBox = &models.BoundingBoxFilter{
			MinLat: minLat,
			MinLon: minLon,
			MaxLat: maxLat,
			MaxLon: maxLon,
		}
	}

	page := 1
	if pStr := query.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 50
	if lStr := query.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 3000 {
		limit = 3000
	}

	offset := (page - 1) * limit
	sortBy := query.Get("sort")
	sortOrder := query.Get("order")

	flights, totalMatching, err := s.db.GetLiveFlights(filter, sortBy, sortOrder, limit, offset)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query live flights: "+err.Error(), start)
		return
	}

	globalStats, _ := s.db.GetGlobalStats()
	totalRecords := 0
	if globalStats != nil {
		if t, ok := globalStats["total_tracked"].(int); ok {
			totalRecords = t
		}
	}

	// Dynamic Field Restriction: normal users receive allowed fields, admin gets 100% full fields
	responseData := s.auth.ApplyFieldFiltering(flights, user)

	s.respondJSON(w, r, http.StatusOK, responseData, totalRecords, totalMatching, page, limit, start)
}

func (s *Server) handleFlightDetail(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/flights/")
	identifier := strings.TrimSpace(path)
	if identifier == "" {
		s.handleFlights(w, r)
		return
	}

	// Route: /api/v1/flights/{identifier}/history
	if strings.HasSuffix(identifier, "/history") {
		ident := strings.TrimSuffix(identifier, "/history")
		s.serveFlightHistory(w, r, ident, start)
		return
	}

	flight, err := s.db.GetFlightByIdentifier(identifier)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Database error: "+err.Error(), start)
		return
	}
	if flight == nil {
		s.respondError(w, http.StatusNotFound, "Flight not found: "+identifier, start)
		return
	}

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	responseData := s.auth.ApplySingleFlightFiltering(flight, user)

	s.respondJSON(w, r, http.StatusOK, responseData, 1, 1, 1, 1, start)
}

func (s *Server) handleHistorySearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	q := r.URL.Query()

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	if user == nil || user.Role != "admin" {
		allowedMap := s.auth.GetAllowedFieldsMap()
		if !allowedMap["telemetry"] {
			s.respondError(w, http.StatusForbidden, "Field restriction: access to historical trajectory telemetry requires 'telemetry' permission", start)
			return
		}
		if !allowedMap["flight"] && (q.Get("callsign") != "" || q.Get("flight") != "") {
			s.respondError(w, http.StatusForbidden, "Field restriction: querying flight history by callsign requires 'flight' permission", start)
			return
		}
		if !allowedMap["aircraft"] && q.Get("hex") != "" {
			s.respondError(w, http.StatusForbidden, "Field restriction: querying flight history by hex requires 'aircraft' permission", start)
			return
		}
	}

	ident := q.Get("callsign")
	if ident == "" {
		ident = q.Get("hex")
	}
	if ident == "" {
		ident = q.Get("flight")
	}
	if ident == "" {
		s.respondError(w, http.StatusBadRequest, "Flight identifier required: provide ?callsign= or ?hex=", start)
		return
	}
	s.serveFlightHistory(w, r, ident, start)
}

func (s *Server) serveFlightHistory(w http.ResponseWriter, r *http.Request, identifier string, start time.Time) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		s.respondError(w, http.StatusBadRequest, "Flight callsign or hex identifier required", start)
		return
	}

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	if user == nil || user.Role != "admin" {
		allowedMap := s.auth.GetAllowedFieldsMap()
		if !allowedMap["telemetry"] {
			s.respondError(w, http.StatusForbidden, "Field restriction: access to historical trajectory telemetry requires 'telemetry' permission", start)
			return
		}
	}

	query := r.URL.Query()
	limit := 300
	if lStr := query.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 2000 {
		limit = 2000
	}

	var fromUnix, toUnix int64
	if fStr := query.Get("from"); fStr != "" {
		if v, err := strconv.ParseInt(fStr, 10, 64); err == nil {
			fromUnix = v
		}
	}
	if tStr := query.Get("to"); tStr != "" {
		if v, err := strconv.ParseInt(tStr, 10, 64); err == nil {
			toUnix = v
		}
	}

	hist, err := s.db.GetFlightHistory(trimmed, fromUnix, toUnix, limit)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query historical track: "+err.Error(), start)
		return
	}
	if hist == nil || hist.PointCount == 0 {
		s.respondError(w, http.StatusNotFound, "No historical telemetry points found for flight within retention window: "+trimmed, start)
		return
	}

	// Dynamic Field Restriction for history
	if user == nil || user.Role != "admin" {
		allowedMap := s.auth.GetAllowedFieldsMap()
		if !allowedMap["aircraft"] {
			hist.Hex = ""
		}
		if !allowedMap["flight"] {
			hist.CallsignICAO = ""
			hist.CallsignIATA = nil
			hist.Airline = nil
		}
		if !allowedMap["route"] {
			hist.Route = nil
		}
		for i := range hist.TrackPoints {
			if !allowedMap["emergency"] {
				hist.TrackPoints[i].Squawk = ""
				hist.TrackPoints[i].IsEmergency = false
			}
			if !allowedMap["diversion_analysis"] {
				hist.TrackPoints[i].IsDiverting = false
			}
		}
	}

	s.respondJSON(w, r, http.StatusOK, hist, hist.PointCount, hist.PointCount, 1, limit, start)
}

func (s *Server) handleEmergencies(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	query := r.URL.Query()

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	if user == nil || user.Role != "admin" {
		allowedMap := s.auth.GetAllowedFieldsMap()
		if !allowedMap["emergency"] {
			s.respondError(w, http.StatusForbidden, "Field restriction: access to emergency flight data requires 'emergency' permission", start)
			return
		}
	}

	page := 1
	if pStr := query.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	limit := 50
	offset := (page - 1) * limit

	isEmerg := true
	filter := models.FlightQueryFilter{
		IsEmergency: &isEmerg,
	}

	flights, totalMatching, err := s.db.GetLiveFlights(filter, "last_seen", "desc", limit, offset)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Database error: "+err.Error(), start)
		return
	}

	responseData := s.auth.ApplyFieldFiltering(flights, user)

	s.respondJSON(w, r, http.StatusOK, responseData, totalMatching, totalMatching, page, limit, start)
}

func (s *Server) handleDiversions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	query := r.URL.Query()

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	if user == nil || user.Role != "admin" {
		allowedMap := s.auth.GetAllowedFieldsMap()
		if !allowedMap["diversion_analysis"] {
			s.respondError(w, http.StatusForbidden, "Field restriction: access to diversion flight data requires 'diversion_analysis' permission", start)
			return
		}
	}

	page := 1
	if pStr := query.Get("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	limit := 50
	offset := (page - 1) * limit

	isDiv := true
	filter := models.FlightQueryFilter{
		IsDiverting: &isDiv,
	}

	flights, totalMatching, err := s.db.GetLiveFlights(filter, "last_seen", "desc", limit, offset)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Database error: "+err.Error(), start)
		return
	}

	responseData := s.auth.ApplyFieldFiltering(flights, user)

	s.respondJSON(w, r, http.StatusOK, responseData, totalMatching, totalMatching, page, limit, start)
}

func (s *Server) handleAirports(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	code := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v1/airports/")))
	if code == "" {
		s.respondError(w, http.StatusBadRequest, "Airport ICAO or IATA code required", start)
		return
	}

	apt := s.airportDB.Lookup(code)
	if apt == nil {
		s.respondError(w, http.StatusNotFound, "Airport not found in OurAirports database: "+code, start)
		return
	}

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	data := map[string]interface{}{
		"airport": apt,
	}

	// Active inbound and outbound flights expose origin and destination routes.
	// Only query and return them if the user is admin or 'route' field is allowed.
	if user != nil && user.Role == "admin" || s.auth.GetAllowedFieldsMap()["route"] {
		inboundFilter := models.FlightQueryFilter{
			DestinationICAO: apt.ICAO,
		}
		inboundFlights, inCount, _ := s.db.GetLiveFlights(inboundFilter, "last_seen", "desc", 20, 0)

		outboundFilter := models.FlightQueryFilter{
			OriginICAO: apt.ICAO,
		}
		outboundFlights, outCount, _ := s.db.GetLiveFlights(outboundFilter, "last_seen", "desc", 20, 0)

		data["active_inbound"] = s.auth.ApplyFieldFiltering(inboundFlights, user)
		data["inbound_count"] = inCount
		data["active_outbound"] = s.auth.ApplyFieldFiltering(outboundFlights, user)
		data["outbound_count"] = outCount
	}

	s.respondJSON(w, r, http.StatusOK, data, 1, 1, 1, 1, start)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	stats, err := s.db.GetGlobalStats()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Database error: "+err.Error(), start)
		return
	}

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	filteredStats := s.applyStatsFiltering(stats, user)

	pauseInfo := s.poller.GetUpstreamPauseInfo()
	filteredStats["upstream_pause"] = pauseInfo

	s.respondJSON(w, r, http.StatusOK, filteredStats, 1, 1, 1, 1, start)
}
