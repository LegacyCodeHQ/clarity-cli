package watch

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// broker manages SSE client connections and broadcasts DOT strings.
type broker struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
	latest  string
}

func newBroker() *broker {
	return &broker{
		clients: make(map[chan string]struct{}),
	}
}

func (b *broker) subscribe() chan string {
	ch := make(chan string, 1)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	if b.latest != "" {
		ch <- b.latest
	}
	b.mu.Unlock()
	return ch
}

func (b *broker) unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *broker) publish(dot string) {
	b.mu.Lock()
	b.latest = dot
	for ch := range b.clients {
		select {
		case ch <- dot:
		default:
		}
	}
	b.mu.Unlock()
}

func newServer(b *broker, port int) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/events", handleSSE(b))

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}

func handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

func handleSSE(b *broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := b.subscribe()
		defer b.unsubscribe(ch)

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case dot, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "event: graph\n")
				for _, line := range strings.Split(dot, "\n") {
					fmt.Fprintf(w, "data: %s\n", line)
				}
				fmt.Fprintf(w, "\n")
				flusher.Flush()
			}
		}
	}
}
