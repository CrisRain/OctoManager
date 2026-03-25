package httpx

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/cloudwego/hertz/pkg/app"
)

// StreamWriter writes newline-delimited JSON (NDJSON) events to an underlying writer.
// Each event is a JSON object {"event":"<name>","data":<payload>} followed by a newline.
type StreamWriter struct {
	w *bufio.Writer
}

// WriteEvent marshals payload and sends a named NDJSON event line.
func (s *StreamWriter) WriteEvent(event string, payload any) error {
	msg := map[string]any{
		"event": event,
		"data":  payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "%s\n", data); err != nil {
		return err
	}
	return s.w.Flush()
}

// PrepareStream sets chunked-streaming headers on the Hertz response and calls fn in a goroutine.
// The response body is NDJSON (application/x-ndjson), one event per line.
func PrepareStream(c *app.RequestContext, fn func(w *StreamWriter)) {
	c.Response.Header.Set("Content-Type", "application/x-ndjson")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")

	pr, pw := io.Pipe()

	go func() {
		bw := bufio.NewWriter(pw)
		fn(&StreamWriter{w: bw})
		_ = bw.Flush()
		pw.Close()
	}()

	c.SetBodyStream(pr, -1)
}

// SSEWriter is kept as an alias so callers can be migrated incrementally.
// Deprecated: use StreamWriter.
type SSEWriter = StreamWriter

// PrepareSSE is kept for backward compatibility.
// Deprecated: use PrepareStream.
func PrepareSSE(c *app.RequestContext, fn func(w *SSEWriter)) {
	PrepareStream(c, fn)
}
