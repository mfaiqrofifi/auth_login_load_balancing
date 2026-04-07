package model

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Message      string `json:"message"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"-"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type RegisterResponse struct {
	Message         string `json:"message"`
	UserID          string `json:"user_id"`
	Email           string `json:"email"`
	IsEmailVerified bool   `json:"is_email_verified"`
}

type AuthIdentity struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
	Message      string `json:"message"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"-"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type MeResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type SessionDeleteResponse struct {
	Message string `json:"message"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error     APIError `json:"error"`
	RequestID string   `json:"request_id,omitempty"`
}
