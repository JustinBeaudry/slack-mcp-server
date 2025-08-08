package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"go.uber.org/zap"
)

// PatchedSSEServer is a wrapper around an http.Handler that fixes the JSON output for SSE.
type PatchedSSEServer struct {
	handler http.Handler
	logger  *zap.Logger
}

// NewPatchedSSEServer creates a new PatchedSSEServer.
func NewPatchedSSEServer(handler http.Handler, logger *zap.Logger) *PatchedSSEServer {
	return &PatchedSSEServer{
		handler: handler,
		logger:  logger,
	}
}

// Start starts the patched SSE server.
func (s *PatchedSSEServer) Start(addr string) error {
	s.logger.Info("Starting patched SSE server", zap.String("address", addr))
	return http.ListenAndServe(addr, s)
}

// ServeHTTP implements the http.Handler interface.
func (s *PatchedSSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	recorder := httptest.NewRecorder()
	s.handler.ServeHTTP(recorder, r)

	for k, v := range recorder.Header() {
		w.Header()[k] = v
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(recorder.Code)

	decoder := json.NewDecoder(recorder.Body)
	for decoder.More() {
		var v interface{}
		if err := decoder.Decode(&v); err != nil {
			s.logger.Error("Error decoding JSON from buffer", zap.Error(err))
			return
		}

		jsonBytes, err := json.Marshal(v)
		if err != nil {
			s.logger.Error("Error marshalling JSON", zap.Error(err))
			return
		}

		fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
	}

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
