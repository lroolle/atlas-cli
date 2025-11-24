package cmdutil

import (
	"fmt"
	"os"

	"github.com/lroolle/atlas-cli/api"
)

func ExitIfError(err error) {
	if err == nil {
		return
	}

	var msg string
	if e, ok := err.(*api.ErrUnexpectedResponse); ok {
		msg = formatAPIError(e)
	} else {
		msg = err.Error()
	}

	Fail(msg)
}

func formatAPIError(e *api.ErrUnexpectedResponse) string {
	if e.Body.String() != "" {
		return e.Body.String()
	}
	return fmt.Sprintf("request failed with status %d: %s", e.StatusCode, e.Status)
}

func Fail(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(1)
}

func FailWithHint(msg, hint string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	fmt.Fprintf(os.Stderr, "Hint: %s\n", hint)
	os.Exit(1)
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
