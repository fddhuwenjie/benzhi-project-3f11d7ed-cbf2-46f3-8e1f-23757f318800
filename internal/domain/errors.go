package domain

import "fmt"

type Error struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func Invalid(field, message string) error {
	return &Error{Code: "validation_error", Field: field, Message: message}
}
func Conflict(message string) error  { return &Error{Code: "state_conflict", Message: message} }
func Forbidden(message string) error { return &Error{Code: "forbidden", Message: message} }
func NotFound(message string) error  { return &Error{Code: "not_found", Message: message} }
