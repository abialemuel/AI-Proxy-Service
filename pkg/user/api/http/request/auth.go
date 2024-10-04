package request

type AuthCallback struct {
	State string `query:"state" validate:"required"`
	Code  string `query:"code" validate:"required"`
}

type AuthLogin struct {
	Provider string `query:"provider" validate:"required"`
}

type AuthRefresh struct {
	Provider     string `query:"provider" validate:"required"`
	RefreshToken string `query:"refresh_token" validate:"required"`
}
