package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// CapVerifier verifies tokens issued by the self-hosted Cap widget.
type CapVerifier struct {
	VerifyURL string
	SecretKey string
	SiteKey   string
	client    *http.Client
}

// CapVerifyRequest is the payload sent to /siteverify.
type CapVerifyRequest struct {
	Secret   string `json:"secret"`
	Response string `json:"response"`
}

// CapVerifyResponse is the JSON structure returned by Cap /siteverify.
type CapVerifyResponse struct {
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
	Flags   []string `json:"riskFlags,omitempty"`
}

// NewCapVerifier creates a new verifier configured for the local Cap instance.
func NewCapVerifier(verifyURL, secretKey, siteKey string) *CapVerifier {
	return &CapVerifier{
		VerifyURL: verifyURL,
		SecretKey: secretKey,
		SiteKey:   siteKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Verify verifies a Cap token. If token is invalid or already redeemed, returns false.
func (c *CapVerifier) Verify(token string) (bool, error) {
	if token == "" {
		return false, errors.New("missing human verification token")
	}
	// Support testing / CI bypass
	if token == "test-cap-token-ok" || c.SecretKey == "" || c.VerifyURL == "" {
		return true, nil
	}

	payload, err := json.Marshal(CapVerifyRequest{
		Secret:   c.SecretKey,
		Response: token,
	})
	if err != nil {
		return false, err
	}

	resp, err := c.client.Post(c.VerifyURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result CapVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Success, nil
}
