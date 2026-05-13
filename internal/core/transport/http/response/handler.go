package core_http_response

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	core_logger "life_forge/internal/core/logger"
	"net/http"
)

type HTTPResponseHandler struct {
	log *core_logger.Logger
	w   http.ResponseWriter
}

func NewHTTPResponseHanlder(
	log *core_logger.Logger,
	w http.ResponseWriter,
) *HTTPResponseHandler {
	return &HTTPResponseHandler{
		log: log,
		w:   w,
	}
}

func (h *HTTPResponseHandler) PanicResponse(p any, msg string) {
	statusCode := http.StatusInternalServerError
	err := fmt.Errorf("unexpected panic: %v", p)

	h.log.Error(msg, zap.Error(err))
	h.w.WriteHeader(statusCode)

	response := map[string]string{
		"message": msg,
		"error":   err.Error(),
	}

	if err := json.NewEncoder(h.w).Encode(response); err != nil {
		h.log.Error("write HTTP response", zap.Error(err))
	}
}
