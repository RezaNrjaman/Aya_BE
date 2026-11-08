package logging

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

var sensitiveKeys = []string{
	"password",
	"kata_sandi",
	"token",
	"jwt",
	"secret",
	"service_key",
	"authorization",
	"credit_card",
	"cvv",
}

var bearerRegex = regexp.MustCompile(`(?i)Bearer\s+([A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+\.?[A-Za-z0-9\-_.+/=]*)`)

// RedactValue masks sensitive string data.
func RedactValue(key, val string) string {
	lowerKey := strings.ToLower(key)
	for _, sk := range sensitiveKeys {
		if strings.Contains(lowerKey, sk) {
			return "[REDACTED]"
		}
	}
	// Mask Bearer tokens inside values
	if bearerRegex.MatchString(val) {
		return bearerRegex.ReplaceAllString(val, "Bearer [REDACTED]")
	}
	return val
}

// RedactMap recurses through a map to redact sensitive keys.
func RedactMap(data map[string]any) map[string]any {
	cleaned := make(map[string]any, len(data))
	for k, v := range data {
		lowerK := strings.ToLower(k)
		isSensitive := false
		for _, sk := range sensitiveKeys {
			if strings.Contains(lowerK, sk) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			cleaned[k] = "[REDACTED]"
			continue
		}

		switch val := v.(type) {
		case string:
			cleaned[k] = RedactValue(k, val)
		case map[string]any:
			cleaned[k] = RedactMap(val)
		default:
			cleaned[k] = v
		}
	}
	return cleaned
}

type LogEntry struct {
	Level     string         `json:"level"`
	Timestamp string         `json:"timestamp"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id,omitempty"`
	Method    string         `json:"method,omitempty"`
	Path      string         `json:"path,omitempty"`
	Status    int            `json:"status,omitempty"`
	LatencyMs int64          `json:"latency_ms,omitempty"`
	ClientIP  string         `json:"client_ip,omitempty"`
	Error     string         `json:"error,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Logger struct {
	out io.Writer
}

var defaultLogger = &Logger{out: os.Stdout}

func SetOutput(w io.Writer) {
	defaultLogger.out = w
}

func logJSON(entry LogEntry) {
	entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	if entry.Metadata != nil {
		entry.Metadata = RedactMap(entry.Metadata)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("failed to serialize log: %v", err)
		return
	}
	defaultLogger.out.Write(append(data, '\n'))
}

func Info(msg string, meta map[string]any) {
	logJSON(LogEntry{
		Level:    "INFO",
		Message:  msg,
		Metadata: meta,
	})
}

func Error(msg string, err error, meta map[string]any) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	logJSON(LogEntry{
		Level:    "ERROR",
		Message:  msg,
		Error:    errStr,
		Metadata: meta,
	})
}

func LogHTTP(entry LogEntry) {
	logJSON(entry)
}
