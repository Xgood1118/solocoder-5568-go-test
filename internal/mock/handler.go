package mock

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"apitester/internal/models"
)

type Handler struct {
	server *MockServer
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rules := h.server.getRules()
	rule := MatchRequest(r, rules)

	if rule == nil {
		http.NotFound(w, r)
		return
	}

	if rule.DelayMs > 0 {
		time.Sleep(time.Duration(rule.DelayMs) * time.Millisecond)
	}

	for key, value := range rule.Headers {
		w.Header().Set(key, value)
	}

	status := rule.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)

	body, err := h.getResponseBody(rule)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if body != nil {
		_, _ = w.Write(body)
	}
}

func (h *Handler) getResponseBody(rule *models.MockRule) ([]byte, error) {
	if rule.BodyFile != "" {
		return os.ReadFile(rule.BodyFile)
	}

	if rule.Body != nil {
		switch v := rule.Body.(type) {
		case string:
			return []byte(v), nil
		case []byte:
			return v, nil
		default:
			return json.Marshal(rule.Body)
		}
	}

	return nil, nil
}
