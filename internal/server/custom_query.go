package server

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"contrail/internal/models"
)

// handleCustomQuery processes the custom JSON payload for querying flights.
// POST /api/v1/query
// Payload format:
//
//	{
//	  "filters": {
//	    "callsign_prefix": "CES",
//	    "is_emergency": false,
//	    "is_diverting": true,
//	    "min_altitude_ft": 5000,
//	    "max_altitude_ft": 35000,
//	    "origin_icao": "ZSSS",
//	    "destination_icao": "ZBAA",
//	    "bounding_box": {
//	      "min_lat": 20.0,
//	      "min_lon": 110.0,
//	      "max_lat": 40.0,
//	      "max_lon": 125.0
//	    }
//	  },
//	  "sort_by": "last_seen",
//	  "sort_order": "desc",
//	  "limit": 50,
//	  "page": 1
//	}
func (s *Server) handleCustomQuery(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		s.respondError(w, http.StatusMethodNotAllowed, "Method not allowed. Use POST for /api/v1/query", start)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Failed to read request body: "+err.Error(), start)
		return
	}
	defer r.Body.Close()

	var req models.QueryRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error(), start)
			return
		}
	}

	user, _ := r.Context().Value(ctxUserKey).(*models.User)
	if user == nil || user.Role != "admin" {
		allowedMap := s.auth.GetAllowedFieldsMap()
		if !allowedMap["emergency"] && req.Filters.IsEmergency != nil {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by emergency requires 'emergency' permission", start)
			return
		}
		if !allowedMap["diversion_analysis"] && req.Filters.IsDiverting != nil {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by diversion requires 'diversion_analysis' permission", start)
			return
		}
		if !allowedMap["route"] && (req.Filters.OriginICAO != "" || req.Filters.DestinationICAO != "") {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by origin or destination requires 'route' permission", start)
			return
		}
		if !allowedMap["flight"] && req.Filters.CallsignPrefix != "" {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by callsign requires 'flight' permission", start)
			return
		}
		if !allowedMap["aircraft"] && req.Filters.Hex != "" {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by hex requires 'aircraft' permission", start)
			return
		}
		if !allowedMap["telemetry"] && (req.Filters.MinAltitudeFt != nil || req.Filters.MaxAltitudeFt != nil || req.Filters.BoundingBox != nil) {
			s.respondError(w, http.StatusForbidden, "Field restriction: filtering by altitude or bounding box requires 'telemetry' permission", start)
			return
		}
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 3000 {
		limit = 3000
	}

	offset := (page - 1) * limit
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "last_seen"
	}
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	flights, totalMatching, err := s.db.GetLiveFlights(req.Filters, sortBy, sortOrder, limit, offset)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Query execution error: "+err.Error(), start)
		return
	}

	globalStats, _ := s.db.GetGlobalStats()
	totalRecords := 0
	if globalStats != nil {
		if t, ok := globalStats["total_tracked"].(int); ok {
			totalRecords = t
		}
	}

	responseData := s.auth.ApplyFieldFiltering(flights, user)

	s.respondJSON(w, r, http.StatusOK, responseData, totalRecords, totalMatching, page, limit, start)
}
