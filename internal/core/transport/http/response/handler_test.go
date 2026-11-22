package core_http_response_test

import (
	"encoding/json"
	"errors"
	"fmt"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	core_logger "github.com/NightFury2283/life_forge/internal/core/logger"
	response "github.com/NightFury2283/life_forge/internal/core/transport/http/response"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorResponse(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{"invalid_argument", core_errors.ErrInvalidArgument, 400}, {"bad_request", core_errors.ErrBadRequest, 400}, {"not_found", core_errors.ErrNotFound, 404}, {"conflict", core_errors.ErrConflict, 409}, {"unauthorized", core_errors.ErrUnauthorized, 401}, {"unknown", errors.New("failure"), 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			log := &core_logger.Logger{Logger: zap.NewNop()}
			err := fmt.Errorf("service: %w", tt.err)
			response.NewHTTPResponseHanlder(log, rec).ErrorResponse(err, "operation failed")
			result := rec.Result()
			defer result.Body.Close()
			if result.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", result.StatusCode, tt.status)
			}
			if got := result.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q", got)
			}
			var body map[string]string
			if err := json.NewDecoder(result.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["message"] != "operation failed" || body["error"] != err.Error() {
				t.Errorf("body = %v", body)
			}
		})
	}
}

func TestResponseWriterStatus(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name  string
		write func(http.ResponseWriter)
		want  int
	}{
		{"empty_handler", func(http.ResponseWriter) {}, 200},
		{"implicit_ok", func(w http.ResponseWriter) { _, _ = w.Write([]byte("ok")) }, 200},
		{"explicit_status", func(w http.ResponseWriter) { w.WriteHeader(201); _, _ = w.Write([]byte("created")) }, 201},
		{"first_status_wins", func(w http.ResponseWriter) { w.WriteHeader(201); w.WriteHeader(500) }, 201},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			writer := response.NewResponseWriter(rec)
			tt.write(writer)
			if got := writer.GetStatusCodeOrPanic(); got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
			if rec.Result().StatusCode != tt.want {
				t.Errorf("recorded status = %d", rec.Result().StatusCode)
			}
		})
	}
}
