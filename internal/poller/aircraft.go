package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"contrail/internal/database"
	"golang.org/x/time/rate"
)

const (
	nCharset     = "ABCDEFGHJKLMNPQRSTUVWXYZ" // alphabet without I and O
	nDigitset    = "0123456789"
	nAllchars    = "ABCDEFGHJKLMNPQRSTUVWXYZ0123456789"
	nSuffixSize  = 601
	nBucket4Size = 35
	nBucket3Size = 10*35 + 601    // 951
	nBucket2Size = 10*951 + 601   // 10111
	nBucket1Size = 10*10111 + 601 // 101711
)

func getNSuffix(offset int) string {
	if offset == 0 {
		return ""
	}
	cLen := len(nCharset)
	char0 := nCharset[(offset-1)/(cLen+1)]
	rem := (offset - 1) % (cLen + 1)
	if rem == 0 {
		return string(char0)
	}
	return string(char0) + string(nCharset[rem-1])
}

// HexToNNumber converts a US Mode S ICAO hex address to an FAA tail number (N-Number).
// US registrations are in hex range 0xA00001 to 0xADF7C7.
func HexToNNumber(hexStr string) string {
	hexClean := strings.TrimSpace(strings.ToUpper(hexStr))
	if len(hexClean) != 6 || hexClean[0] != 'A' {
		return ""
	}
	val, err := strconv.ParseInt(hexClean[1:], 16, 64)
	if err != nil {
		return ""
	}
	i := int(val) - 1
	if i < 0 {
		return "N"
	}
	if i > 915398 {
		return ""
	}

	dig1 := (i / nBucket1Size) + 1
	rem1 := i % nBucket1Size
	output := fmt.Sprintf("N%d", dig1)

	if rem1 < nSuffixSize {
		return output + getNSuffix(rem1)
	}

	rem1 -= nSuffixSize
	dig2 := rem1 / nBucket2Size
	rem2 := rem1 % nBucket2Size
	output += fmt.Sprintf("%d", dig2)

	if rem2 < nSuffixSize {
		return output + getNSuffix(rem2)
	}

	rem2 -= nSuffixSize
	dig3 := rem2 / nBucket3Size
	rem3 := rem2 % nBucket3Size
	output += fmt.Sprintf("%d", dig3)

	if rem3 < nSuffixSize {
		return output + getNSuffix(rem3)
	}

	rem3 -= nSuffixSize
	dig4 := rem3 / nBucket4Size
	rem4 := rem3 % nBucket4Size
	output += fmt.Sprintf("%d", dig4)

	if rem4 == 0 {
		return output
	}

	if rem4-1 < len(nAllchars) {
		return output + string(nAllchars[rem4-1])
	}
	return output
}

// HexToCountry maps an ICAO 24-bit Mode S hex address to its country of registration.
func HexToCountry(hexStr string) string {
	hexClean := strings.TrimSpace(strings.ToUpper(hexStr))
	if len(hexClean) < 2 {
		return ""
	}
	val, err := strconv.ParseUint(hexClean, 16, 32)
	if err != nil {
		return ""
	}

	v := uint32(val)
	switch {
	case v >= 0xA00000 && v <= 0xADFFFF:
		return "United States"
	case v >= 0xC00000 && v <= 0xC3FFFF:
		return "Canada"
	case v >= 0x780000 && v <= 0x7BFFFF:
		return "China"
	case v >= 0x840000 && v <= 0x87FFFF:
		return "Japan"
	case v >= 0x380000 && v <= 0x39FFFF:
		return "France"
	case v >= 0x3C0000 && v <= 0x3FFFFF:
		return "Germany"
	case v >= 0x400000 && v <= 0x43FFFF:
		return "United Kingdom"
	case v >= 0x480000 && v <= 0x487FFF:
		return "Netherlands"
	case v >= 0x710000 && v <= 0x717FFF:
		return "Republic of Korea"
	case v >= 0x7C0000 && v <= 0x7FFFFF:
		return "Australia"
	case v >= 0x4B0000 && v <= 0x4B7FFF:
		return "Switzerland"
	case v >= 0x440000 && v <= 0x447FFF:
		return "Austria"
	case v >= 0x300000 && v <= 0x33FFFF:
		return "Italy"
	case v >= 0x340000 && v <= 0x37FFFF:
		return "Spain"
	case v >= 0x140000 && v <= 0x15FFFF:
		return "Russia"
	case v >= 0xE40000 && v <= 0xE47FFF:
		return "Brazil"
	case v >= 0x0D0000 && v <= 0x0D7FFF:
		return "Mexico"
	case v >= 0x738000 && v <= 0x73FFFF:
		return "Singapore"
	case v >= 0x894000 && v <= 0x897FFF:
		return "United Arab Emirates"
	case v >= 0x898000 && v <= 0x899FFF:
		return "Qatar"
	case v >= 0x800000 && v <= 0x803FFF:
		return "India"
	case v >= 0x4A0000 && v <= 0x4A7FFF:
		return "Romania"
	case v >= 0x500000 && v <= 0x507FFF:
		return "Ireland"
	case v >= 0x470000 && v <= 0x477FFF:
		return "Norway"
	case v >= 0x4A8000 && v <= 0x4AFFFF:
		return "Sweden"
	case v >= 0x458000 && v <= 0x45FFFF:
		return "Denmark"
	case v >= 0x460000 && v <= 0x467FFF:
		return "Finland"
	case v >= 0x490000 && v <= 0x497FFF:
		return "Portugal"
	case v >= 0x4B8000 && v <= 0x4BFFFF:
		return "Turkey"
	default:
		return ""
	}
}

// AircraftEnricher manages aircraft registration and type enrichment.
type AircraftEnricher struct {
	db         *database.DB
	httpClient *http.Client
	limiter    *rate.Limiter
	memCache   sync.Map // hex -> *database.CachedAircraft
	queue      chan string
	queueSet   map[string]bool
	queueMu    sync.Mutex
	cacheTTL   time.Duration
}

// NewAircraftEnricher creates an AircraftEnricher.
func NewAircraftEnricher(db *database.DB, limiter *rate.Limiter) *AircraftEnricher {
	return &AircraftEnricher{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		limiter:  limiter,
		queue:    make(chan string, 10000),
		queueSet: make(map[string]bool),
		cacheTTL: 30 * 24 * time.Hour, // Aircraft metadata valid for 30 days
	}
}

// StartWorker starts background aircraft enrichment workers.
func (ae *AircraftEnricher) StartWorker(ctx context.Context, workers int) {
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case hex, ok := <-ae.queue:
					if !ok {
						return
					}
					ae.queueMu.Lock()
					delete(ae.queueSet, hex)
					ae.queueMu.Unlock()

					if ae.isCached(hex) {
						continue
					}

					if err := ae.limiter.Wait(ctx); err != nil {
						return
					}

					ae.fetchAndCache(ctx, hex)
				}
			}
		}()
	}
}

func (ae *AircraftEnricher) isCached(hex string) bool {
	if val, ok := ae.memCache.Load(hex); ok {
		cached := val.(*database.CachedAircraft)
		if cached.ExpiresAt.After(time.Now()) {
			return true
		}
	}
	cached, found, err := ae.db.GetCachedAircraft(hex)
	if err == nil && found {
		ae.memCache.Store(hex, cached)
		return true
	}
	return false
}

// EnrichAircraft returns registration, aircraft type, and country for a given hex.
func (ae *AircraftEnricher) EnrichAircraft(hex, upstreamReg, upstreamType string) (registration, typeCode, country string) {
	hexLower := strings.ToLower(strings.TrimSpace(hex))
	country = HexToCountry(hexLower)

	// Upstream-provided metadata (e.g. from adsb.fi)
	if upstreamReg != "" {
		registration = upstreamReg
	}
	if upstreamType != "" {
		typeCode = upstreamType
	}

	// For US aircraft, compute exact FAA N-number if registration is still missing
	if registration == "" {
		if nNum := HexToNNumber(hexLower); nNum != "" {
			registration = nNum
		}
	}

	// Fast in-memory cache check
	if val, ok := ae.memCache.Load(hexLower); ok {
		cached := val.(*database.CachedAircraft)
		if cached.ExpiresAt.After(time.Now()) {
			if registration == "" && cached.Registration != "" {
				registration = cached.Registration
			}
			if typeCode == "" && cached.TypeCode != "" {
				typeCode = cached.TypeCode
			}
			if country == "" && cached.Country != "" {
				country = cached.Country
			}
			return registration, typeCode, country
		}
	}

	// Database cache check
	cached, found, err := ae.db.GetCachedAircraft(hexLower)
	if err == nil && found {
		ae.memCache.Store(hexLower, cached)
		if registration == "" && cached.Registration != "" {
			registration = cached.Registration
		}
		if typeCode == "" && cached.TypeCode != "" {
			typeCode = cached.TypeCode
		}
		if country == "" && cached.Country != "" {
			country = cached.Country
		}
		return registration, typeCode, country
	}

	// If typeCode is missing, queue for background enrichment from adsbdb
	if typeCode == "" && hexLower != "" {
		ae.queueHex(hexLower)
	}

	return registration, typeCode, country
}

func (ae *AircraftEnricher) queueHex(hex string) {
	ae.queueMu.Lock()
	defer ae.queueMu.Unlock()

	if ae.queueSet[hex] {
		return
	}

	select {
	case ae.queue <- hex:
		ae.queueSet[hex] = true
	default:
	}
}

type adsbDbAircraftResponse struct {
	Response struct {
		Aircraft struct {
			Type         string `json:"type"`
			ICAOType     string `json:"icao_type"`
			Manufacturer string `json:"manufacturer"`
			ModeS        string `json:"mode_s"`
			Registration string `json:"registration"`
		} `json:"aircraft"`
	} `json:"response"`
}

func (ae *AircraftEnricher) fetchAndCache(ctx context.Context, hex string) {
	url := fmt.Sprintf("https://api.adsbdb.com/v0/aircraft/%s", hex)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "Contrail/1.0 (+https://airline.api.veyino.org)")

	resp, err := ae.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	country := HexToCountry(hex)

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		var apiRes adsbDbAircraftResponse
		if err := json.Unmarshal(body, &apiRes); err == nil {
			ac := apiRes.Response.Aircraft
			reg := ac.Registration
			if reg == "" {
				reg = HexToNNumber(hex)
			}
			tCode := ac.ICAOType
			if tCode == "" {
				tCode = ac.Type
			}

			cached := &database.CachedAircraft{
				Hex:          hex,
				Registration: reg,
				TypeCode:     tCode,
				Model:        ac.Type,
				Country:      country,
				CachedAt:     time.Now(),
				ExpiresAt:    time.Now().Add(ae.cacheTTL),
			}
			ae.memCache.Store(hex, cached)
			ae.db.SetCachedAircraft(hex, reg, tCode, ac.Type, country, ae.cacheTTL)
			return
		}
	}

	// Cache fallback negative or basic entry for 12 hours
	reg := HexToNNumber(hex)
	cached := &database.CachedAircraft{
		Hex:          hex,
		Registration: reg,
		TypeCode:     "",
		Model:        "",
		Country:      country,
		CachedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(12 * time.Hour),
	}
	ae.memCache.Store(hex, cached)
	ae.db.SetCachedAircraft(hex, reg, "", "", country, 12*time.Hour)
}
