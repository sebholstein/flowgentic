package relay

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

)

// allowedPrefixes is the set of service path prefixes that the relay will
// forward to workers. Requests for any other path are rejected with 403.
var allowedPrefixes = []string{
	"/worker.v1.SystemService/",
	"/worker.v1.WorkerService/",
}

// StartDeps are the dependencies required to start the relay.
type StartDeps struct {
	Mux *http.ServeMux
	Log *slog.Logger
}

// Registry holds the set of known workers and their connection details.
type Registry struct {
	mu      sync.RWMutex
	urls    map[string]*url.URL
	secrets map[string]string
	// defaultID is the worker used when no X-Worker-Id header is provided.
	defaultID string
}

// Lookup returns the URL and secret for the given worker ID.
func (r *Registry) Lookup(workerID string) (workerURL string, secret string, ok bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.urls[workerID]
	if !ok {
		return "", "", false
	}
	return u.String(), r.secrets[workerID], true
}

// AddWorker adds or replaces a worker in the registry.
func (r *Registry) AddWorker(id string, rawURL string, secret string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.urls[id] = u
	r.secrets[id] = secret
	if r.defaultID == "" {
		r.defaultID = id
	}
	return nil
}

// RemoveWorker removes a worker from the registry.
func (r *Registry) RemoveWorker(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.urls, id)
	delete(r.secrets, id)
	if r.defaultID == id {
		r.defaultID = ""
		for k := range r.urls {
			r.defaultID = k
			break
		}
	}
}

// Start registers the relay handler on the mux for each allowed service prefix
// and returns the Registry so other features can add workers dynamically.
func Start(deps StartDeps) *Registry {
	reg := &Registry{
		urls:    make(map[string]*url.URL),
		secrets: make(map[string]string),
	}

	for _, prefix := range allowedPrefixes {
		deps.Mux.Handle(prefix, &relayHandler{
			log: deps.Log,
			reg: reg,
		})
	}

	deps.Log.Info("relay: registered handlers")
	return reg
}

type relayHandler struct {
	log *slog.Logger
	reg *Registry
}

func (h *relayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Determine target worker.
	workerID := r.Header.Get("X-Worker-Id")
	if workerID == "" {
		h.reg.mu.RLock()
		workerID = h.reg.defaultID
		h.reg.mu.RUnlock()
	}

	workerURL, secret, ok := h.reg.Lookup(workerID)
	if !ok {
		h.log.Warn("relay: unknown worker", "worker_id", workerID)
		http.Error(w, "unknown worker", http.StatusForbidden)
		return
	}

	// Verify the request path is in the allowlist.
	if !isAllowed(r.URL.Path) {
		http.Error(w, "service not allowed", http.StatusForbidden)
		return
	}

	targetURL, _ := url.Parse(workerURL)
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(targetURL)
			pr.Out.Header.Set("Authorization", "Bearer "+secret)
			pr.Out.Header.Del("X-Worker-Id")
		},
	}
	proxy.ServeHTTP(w, r)
}

func isAllowed(path string) bool {
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
