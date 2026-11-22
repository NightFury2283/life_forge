package core_http_request

import (
	"encoding/json"
	"fmt"
	core_errors "github.com/NightFury2283/life_forge/internal/core/errors"
	"github.com/go-playground/validator/v10"
	"io"
	"net/http"
)

var requestValidator = validator.New()

func DecodeAndValidateRequest(r *http.Request, dest any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return fmt.Errorf("decode json: %w: %w", core_errors.ErrBadRequest, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("request must contain one JSON object: %w", core_errors.ErrBadRequest)
	}

	if err := requestValidator.Struct(dest); err != nil {
		return fmt.Errorf("request validation: %w: %w", core_errors.ErrInvalidArgument, err)
	}

	return nil
}
