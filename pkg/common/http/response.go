package common

// DefaultResponse default payload response
type DefaultResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message"`
	// Internal error  `json:"-"`
}

// NewValidationErrorResponse default validation error response
func NewValidationErrorResponse(message string) DefaultResponse {
	return DefaultResponse{
		400,
		ValidationErrStatus,
		message,
	}
}

// NewUnauthorizedResponse default unauthorized response
func NewUnauthorizedResponse(msg string) DefaultResponse {
	return DefaultResponse{
		401,
		UnauthorizedStatus,
		msg,
	}
}

// NewUnauthorizedResponse default unauthorized response
func NewForbiddenResponse(msg string) DefaultResponse {
	return DefaultResponse{
		403,
		ForbiddenStatus,
		msg,
	}
}

// NewDefaultSuccessResponse default validation error response
func NewDefaultSuccessResponse() DefaultResponse {
	return DefaultResponse{
		200,
		SuccessStatus,
		"Success",
	}
}

// NewDefaultSuccessResponse default validation error response
func NewDefaultCreatedResponse() DefaultResponse {
	return DefaultResponse{
		201,
		SuccessStatus,
		"Success",
	}
}

// ErrorResponse error response
type ErrorResponse struct {
	Code     int    `json:"code"`
	Status   string `json:"status,omitempty"`
	Message  string `json:"message"`
	Internal error  `json:"-"`
}
