package captcha

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CaptchaManager manages generation and cryptographic verification of self-hosted challenges.
type CaptchaManager struct {
	secretKey  []byte
	usedTokens sync.Map // captcha_id -> timestamp
}

// CaptchaChallenge represents a newly generated challenge.
type CaptchaChallenge struct {
	CaptchaID string `json:"captcha_id"`
	ImageSVG  string `json:"image_svg"` // base64 SVG data URI
	Question  string `json:"question"`
	ExpiresAt int64  `json:"expires_at"`
}

// NewCaptchaManager initializes the manager.
func NewCaptchaManager(secret string) *CaptchaManager {
	if secret == "" {
		secret = "contrail-default-anti-bot-captcha-secret-2026"
	}
	cm := &CaptchaManager{
		secretKey: []byte(secret),
	}

	// Purge expired tokens every 15 minutes
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		for range ticker.C {
			now := time.Now().Unix()
			cm.usedTokens.Range(func(key, val interface{}) bool {
				if expiry, ok := val.(int64); ok && now > expiry {
					cm.usedTokens.Delete(key)
				}
				return true
			})
		}
	}()

	return cm
}

// Generate creates a math/visual puzzle challenge.
func (cm *CaptchaManager) Generate() (*CaptchaChallenge, error) {
	// Generate random math problem e.g. A + B or A - B
	num1Big, _ := rand.Int(rand.Reader, big.NewInt(30))
	num2Big, _ := rand.Int(rand.Reader, big.NewInt(20))
	opBig, _ := rand.Int(rand.Reader, big.NewInt(2)) // 0: +, 1: -

	a := int(num1Big.Int64()) + 5
	b := int(num2Big.Int64()) + 2
	op := "+"
	ans := a + b
	if opBig.Int64() == 1 {
		if a < b {
			a, b = b, a
		}
		op = "-"
		ans = a - b
	}

	questionText := fmt.Sprintf("%d %s %d = ?", a, op, b)
	ansStr := strconv.Itoa(ans)

	// Random challenge ID
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	captchaID := hex.EncodeToString(randBytes)

	expiresAt := time.Now().Add(5 * time.Minute).Unix()

	// Compute signature
	sig := cm.sign(captchaID, ansStr, expiresAt)
	fullToken := fmt.Sprintf("%s_%d_%s", captchaID, expiresAt, sig)

	svgData := renderSVG(questionText)
	b64SVG := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svgData))

	return &CaptchaChallenge{
		CaptchaID: fullToken,
		ImageSVG:  b64SVG,
		Question:  "Solve the equation to verify you are human",
		ExpiresAt: expiresAt,
	}, nil
}

// Verify validates the user's answer against the token.
func (cm *CaptchaManager) Verify(token, answer string) bool {
	token = strings.TrimSpace(token)
	answer = strings.TrimSpace(answer)
	if token == "" || answer == "" {
		return false
	}

	parts := strings.Split(token, "_")
	if len(parts) != 3 {
		return false
	}

	id := parts[0]
	expiryStr := parts[1]
	expectedSig := parts[2]

	expiry, err := strconv.ParseInt(expiryStr, 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return false // Expired
	}

	// Check if already used
	if _, used := cm.usedTokens.Load(id); used {
		return false // Replay attack prevention
	}

	calculatedSig := cm.sign(id, answer, expiry)
	if !hmac.Equal([]byte(calculatedSig), []byte(expectedSig)) {
		return false
	}

	// Mark used
	cm.usedTokens.Store(id, expiry)
	return true
}

func (cm *CaptchaManager) sign(id, answer string, expiry int64) string {
	h := hmac.New(sha256.New, cm.secretKey)
	h.Write([]byte(fmt.Sprintf("%s:%s:%d", id, strings.TrimSpace(answer), expiry)))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func renderSVG(text string) string {
	// Generate random noise lines and wave distortion
	var lines strings.Builder
	for i := 0; i < 5; i++ {
		x1 := 10 + (i * 28)
		y1 := 5 + (i * 7)
		x2 := 160 - (i * 15)
		y2 := 40 - (i * 4)
		color := []string{"#3b82f6", "#8b5cf6", "#ec4899", "#10b981", "#f59e0b"}[i%5]
		lines.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1.5" stroke-opacity="0.35"/>`, x1, y1, x2, y2, color))
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="160" height="48" viewBox="0 0 160 48">
  <defs>
    <linearGradient id="capbg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" stop-color="#18181b" />
      <stop offset="100%%" stop-color="#27272a" />
    </linearGradient>
    <filter id="distort">
      <feTurbulence type="fractalNoise" baseFrequency="0.04" numOctaves="1" result="noise" />
      <feDisplacementMap in="SourceGraphic" in2="noise" scale="3" xChannelSelector="R" yChannelSelector="G" />
    </filter>
  </defs>
  <rect width="160" height="48" rx="6" fill="url(#capbg)" stroke="#3f3f46" stroke-width="1"/>
  %s
  <text x="80" y="32" font-family="monospace, Courier, sans-serif" font-size="22" font-weight="bold" fill="#f4f4f5" text-anchor="middle" letter-spacing="2" filter="url(#distort)">%s</text>
  <!-- Anti-bot dots -->
  <circle cx="25" cy="18" r="1.5" fill="#a1a1aa" opacity="0.6"/>
  <circle cx="135" cy="35" r="1.5" fill="#a1a1aa" opacity="0.6"/>
  <circle cx="70" cy="12" r="1.2" fill="#a1a1aa" opacity="0.5"/>
</svg>`, lines.String(), text)

	return svg
}
