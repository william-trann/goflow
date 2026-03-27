package handlers

import (
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"io"
	"net/http"

	"github.com/william-trann/goflow/internal/errors"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	if err == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "unknown error"})
		return
	}
	writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
}

func readJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if stdErrors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("%w: invalid request body", errors.ErrValidation)
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != nil && !stdErrors.Is(err, io.EOF) {
		return fmt.Errorf("%w: invalid request body", errors.ErrValidation)
	} else if err == nil {
		return fmt.Errorf("%w: invalid request body", errors.ErrValidation)
	}

	return nil
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", joinMethods(methods))
	writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
}

func joinMethods(methods []string) string {
	switch len(methods) {
	case 0:
		return ""
	case 1:
		return methods[0]
	}

	result := methods[0]
	for i := 1; i < len(methods); i++ {
		result += ", " + methods[i]
	}
	return result
}

func statusForError(err error) int {
	switch {
	case stdErrors.Is(err, errors.ErrValidation):
		return http.StatusBadRequest
	case stdErrors.Is(err, errors.ErrNotFound):
		return http.StatusNotFound
	case stdErrors.Is(err, errors.ErrConflict), stdErrors.Is(err, errors.ErrInvalidState):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
