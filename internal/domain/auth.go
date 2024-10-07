package domain

// UserLoginRequest is the payload for login.
type UserLoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// TokenPair is the response after login or refresh.
type TokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int64  `json:"expires_in"` // access token TTL in seconds
	TokenType        string `json:"token_type"` // "Bearer"
	ProfileCompleted bool   `json:"profile_completed"`
}
