package core_http_request_test

import (
	"errors"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	request "github.com/NightFury2283/life_forge/internal/core/transport/http/request"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeAndValidateRequest(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body string
		wantErr    error
	}{
		{"valid", `{"title":"Сила","value":0}`, nil},
		{"empty_body", "", core_errors.ErrBadRequest},
		{"invalid_json", `{`, core_errors.ErrBadRequest},
		{"wrong_type", `{"title":1}`, core_errors.ErrBadRequest},
		{"unknown_field", `{"title":"Сила","user_id":2}`, core_errors.ErrBadRequest},
		{"two_objects", `{"title":"Сила"} {"title":"Other"}`, core_errors.ErrBadRequest},
		{"trailing_garbage", `{"title":"Сила"} garbage`, core_errors.ErrBadRequest},
		{"missing_required", `{}`, core_errors.ErrInvalidArgument},
		{"negative_value", `{"title":"Сила","value":-1}`, core_errors.ErrInvalidArgument},
		{"null", `null`, core_errors.ErrInvalidArgument},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var dest struct {
				Title string `json:"title" validate:"required,max=100"`
				Value int    `json:"value" validate:"min=0"`
			}
			r := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			err := request.DecodeAndValidateRequest(r, &dest)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if err == nil && dest.Title != "Сила" {
				t.Errorf("decoded title = %q", dest.Title)
			}
		})
	}
}
