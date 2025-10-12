package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"

	"icscommemulator/pkg/logger"
)

func sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("Failed to encode JSON response: %v", err)
	}
}

func sendError(w http.ResponseWriter, message string, status int) {
	logger.Warning("Sending error response: %d - %s", status, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	response := APIResponse{
		Status: status,
		Error:  message,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode error response: %v", err)
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		sendError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return err
	}
	return nil
}

func extractPathSegment(path, prefix string) string {
	path = strings.TrimPrefix(path, prefix)
	path = strings.Trim(path, "/")
	return path
}

func incrementIP(ip net.IP) net.IP {
	result := make(net.IP, len(ip))
	copy(result, ip)
	for i := len(result) - 1; i >= 0; i-- {
		result[i]++
		if result[i] > 0 {
			break
		}
	}
	return result
}

func toJSONFunc(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		logger.Error("Failed to marshal data to JSON in template: %v", err)
		return template.JS("{}")
	}
	return template.JS(b)
}
