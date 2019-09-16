package auto

type AutomaticAccessToken struct {
	UserID       string `json:"user_id" validate:"required"`
	AccessToken  string `json:"access_token" validate:"required"`
	ExpiresIn    int    `json:"expires_in" validate:"required"`
	Scope        string `json:"scope" validate:"required"`
	RefreshToken string `json:"refresh_token" validate:"required"`
	TokenType    string `json:"token_type" validate:"required"`
}
