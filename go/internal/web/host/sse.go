package host

import (
	"fmt"
	"net/http"
	"time"
)

// HandleSSE serves Server-Sent Events from the runtime event bus.
func (h *Host) HandleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	lastID := r.Header.Get("Last-Event-ID")
	if lastID == "" {
		lastID = r.URL.Query().Get("last_event_id")
	}

	if h.EventBus == nil {
		fmt.Fprintf(w, "event: error\ndata: {\"message\": \"event bus unavailable\"}\n\n")
		flusher.Flush()
		return
	}

	sub, replay, _ := h.EventBus.Subscribe(lastID, 128)
	defer sub.Close()

	// Replay any missed events
	for _, ev := range replay {
		fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", ev.ID, ev.Type, string(ev.Data))
	}
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-sub.C:
			if !ok {
				return
			}
			fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", ev.ID, ev.Type, string(ev.Data))
			flusher.Flush()
		case <-ticker.C:
			// SSE comment for keep-alive
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
