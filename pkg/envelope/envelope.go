package envelope

// Envelope is the standard response format for all REST API endpoints.
// Every response wraps data in this envelope to ensure consistent parsing.
type Envelope struct {
	OK    bool         `json:"ok"`
	Data  any          `json:"data"`
	Error *ErrorDetail `json:"error"`
}

// ErrorDetail contains error information within an envelope response.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Success creates a successful envelope with the given data.
func Success(data any) *Envelope {
	return &Envelope{
		OK:    true,
		Data:  data,
		Error: nil,
	}
}

// Fail creates a failure envelope with an error code and message.
func Fail(code, message string) *Envelope {
	return &Envelope{
		OK:   false,
		Data: nil,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

// FailWithDetails creates a failure envelope with additional error details.
func FailWithDetails(code, message string, details any) *Envelope {
	return &Envelope{
		OK:   false,
		Data: nil,
		Error: &ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}
