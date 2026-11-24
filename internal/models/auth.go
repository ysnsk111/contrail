package models

import "time"

// User represents an authorized portal user or administrator.
type User struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	Name          string    `json:"name"`
	Purpose       string    `json:"purpose"`
	Role          string    `json:"role"`   // "user", "admin"
	Status        string    `json:"status"` // "active", "banned"
	TermsAgreedAt int64     `json:"terms_agreed_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// APIKey represents a provisioned token for calling Contrail APIs.
type APIKey struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	KeyName       string     `json:"key_name"`
	KeyPrefix     string     `json:"key_prefix"`
	KeyHash       string     `json:"-"`
	Status        string     `json:"status"` // "active", "revoked"
	TermsAgreed   bool       `json:"terms_agreed"`
	CreatedAt     time.Time  `json:"created_at"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	TotalRequests int64      `json:"total_requests"`
	TodayRequests int64      `json:"today_requests"`
}

// RegisterRequest represents the payload submitted during user registration.
type RegisterRequest struct {
	Name             string `json:"name"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	Purpose          string `json:"purpose"`
	VerificationCode string `json:"verification_code"`
	AgreeTerms       bool   `json:"agree_terms"`
	CapToken         string `json:"cap_token"`
}

// LoginRequest represents credential authentication.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	CapToken string `json:"cap_token"`
}

// SendCodeRequest represents an email verification dispatch request.
type SendCodeRequest struct {
	Email         string `json:"email"`
	CapToken      string `json:"cap_token"`
	CaptchaID     string `json:"captcha_id,omitempty"`
	CaptchaAnswer string `json:"captcha_answer,omitempty"`
}

// CreateAPIKeyRequest represents a key generation request.
type CreateAPIKeyRequest struct {
	KeyName    string `json:"key_name"`
	AgreeTerms bool   `json:"agree_terms"`
}

// UpdateUserStatusRequest represents an admin action to ban or activate a user.
type UpdateUserStatusRequest struct {
	Status string `json:"status"` // "active", "banned"
}

// UpdatePasswordRequest represents user password modification.
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// FieldRestrictionsConfig defines which top-level flight fields are permitted for normal users.
type FieldRestrictionsConfig struct {
	AllowedFields []string `json:"allowed_fields"`
}
