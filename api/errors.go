package api

import (
	"fmt"
	"strings"
)

type ErrUnexpectedResponse struct {
	Body       ErrorResponse
	Status     string
	StatusCode int
}

func (e *ErrUnexpectedResponse) Error() string {
	return fmt.Sprintf("unexpected response (%d): %s", e.StatusCode, e.Body.String())
}

type ErrorResponse struct {
	Errors          map[string]string `json:"errors"`
	ErrorMessages   []string          `json:"errorMessages"`
	WarningMessages []string          `json:"warningMessages"`
}

func (e ErrorResponse) String() string {
	var out strings.Builder

	if len(e.ErrorMessages) > 0 || len(e.Errors) > 0 {
		for _, v := range e.ErrorMessages {
			out.WriteString(fmt.Sprintf("%s\n", v))
		}
		for k, v := range e.Errors {
			out.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		}
	}

	if len(e.WarningMessages) > 0 {
		out.WriteString("\nWarning:\n")
		for _, v := range e.WarningMessages {
			out.WriteString(fmt.Sprintf("  - %s\n", v))
		}
	}

	return strings.TrimSpace(out.String())
}
