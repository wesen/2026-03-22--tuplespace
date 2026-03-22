package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/manuel/wesen/tuplespace/internal/service"
	"github.com/manuel/wesen/tuplespace/internal/validation"
)

func mapError(err error) (int, ErrorPayload) {
	if err == nil {
		return http.StatusOK, ErrorPayload{}
	}

	var validationErr *validation.Error
	if errors.As(err, &validationErr) {
		status := http.StatusBadRequest
		if validationErr.Code == "unsupported_type" {
			status = http.StatusBadRequest
		}
		return status, ErrorPayload{
			Code:    validationErr.Code,
			Message: validationErr.Message,
		}
	}

	switch {
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound, ErrorPayload{Code: "not_found", Message: err.Error()}
	case errors.Is(err, service.ErrTimeout):
		return http.StatusRequestTimeout, ErrorPayload{Code: "timeout", Message: err.Error()}
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusRequestTimeout, ErrorPayload{Code: "timeout", Message: err.Error()}
	case errors.Is(err, context.Canceled):
		return http.StatusRequestTimeout, ErrorPayload{Code: "timeout", Message: err.Error()}
	default:
		return http.StatusInternalServerError, ErrorPayload{Code: "internal", Message: err.Error()}
	}
}
