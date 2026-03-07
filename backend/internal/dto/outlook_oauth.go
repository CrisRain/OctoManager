package dto

type OutlookAuthorizeURLRequest struct {
	Tenant              string   `json:"tenant,omitempty" binding:"omitempty"`
	ClientID            string   `json:"client_id" binding:"required"`
	RedirectURI         string   `json:"redirect_uri" binding:"required"`
	Scope               []string `json:"scope,omitempty" binding:"omitempty"`
	State               string   `json:"state,omitempty" binding:"omitempty"`
	Prompt              string   `json:"prompt,omitempty" binding:"omitempty"`
	LoginHint           string   `json:"login_hint,omitempty" binding:"omitempty"`
	CodeChallenge       string   `json:"code_challenge,omitempty" binding:"omitempty"`
	CodeChallengeMethod string   `json:"code_challenge_method,omitempty" binding:"omitempty"`
}

type OutlookAuthorizeURLResponse struct {
	AuthorizeURL        string   `json:"authorize_url"`
	Tenant              string   `json:"tenant"`
	Scope               []string `json:"scope"`
	State               string   `json:"state,omitempty"`
	CodeChallenge       string   `json:"code_challenge,omitempty"`
	CodeChallengeMethod string   `json:"code_challenge_method,omitempty"`
}

type OutlookExchangeCodeRequest struct {
	Tenant       string   `json:"tenant,omitempty" binding:"omitempty"`
	ClientID     string   `json:"client_id" binding:"required"`
	ClientSecret string   `json:"client_secret,omitempty" binding:"omitempty"`
	RedirectURI  string   `json:"redirect_uri" binding:"required"`
	Code         string   `json:"code" binding:"required"`
	Scope        []string `json:"scope,omitempty" binding:"omitempty"`
	TokenURL     string   `json:"token_url,omitempty" binding:"omitempty"`
	CodeVerifier string   `json:"code_verifier,omitempty" binding:"omitempty"`
}

type OutlookRefreshTokenRequest struct {
	Tenant       string   `json:"tenant,omitempty" binding:"omitempty"`
	ClientID     string   `json:"client_id" binding:"required"`
	ClientSecret string   `json:"client_secret,omitempty" binding:"omitempty"`
	RefreshToken string   `json:"refresh_token" binding:"required"`
	Scope        []string `json:"scope,omitempty" binding:"omitempty"`
	TokenURL     string   `json:"token_url,omitempty" binding:"omitempty"`
}

type OutlookTokenResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    string `json:"expires_at"`
	TokenURL     string `json:"token_url,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}
