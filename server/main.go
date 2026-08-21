// Command server is the M5 sharing server: a dumb document store plus static
// hosting for the built app. It imports core/ only to validate what it
// stores — all reasoning stays in the client's WASM engine (DESIGN §7).
//
// Shared universes are immutable snapshots: publishing again mints a new id,
// and there are no update or delete endpoints. Storage is one JSON file per
// id under -data; no database (the leibniz grave is well marked).
package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/EFinish/Calculemus/core"
)

const maxUniverseBytes = 1 << 20 // 1 MiB — a hand-built universe is kilobytes

var idPattern = regexp.MustCompile(`^[a-z0-9]{10}$`)

func main() {
	addr := flag.String("addr", ":8737", "listen address")
	dataDir := flag.String("data", "data", "directory for stored universe documents")
	distDir := flag.String("dist", "app/dist", "built app to serve (empty to disable)")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}
	s := &server{dataDir: *dataDir}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/universes", s.publish)
	mux.HandleFunc("GET /api/universes/{id}", s.fetch)
	if *distDir != "" {
		mux.Handle("GET /", spaHandler(*distDir))
	}

	log.Printf("calculemus server on %s (data: %s)", *addr, *dataDir)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

type server struct {
	dataDir string
}

func (s *server) path(id string) string {
	return filepath.Join(s.dataDir, id+".json")
}

func (s *server) publish(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, maxUniverseBytes)
	var u core.Universe
	if err := json.NewDecoder(body).Decode(&u); err != nil {
		httpError(w, http.StatusBadRequest, "not valid JSON: "+err.Error())
		return
	}
	if u.Version < 1 {
		httpError(w, http.StatusUnprocessableEntity, "missing or invalid version")
		return
	}
	if err := core.Validate(&u); err != nil {
		httpError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	id, err := newID()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "id generation failed")
		return
	}
	// Store the re-marshaled canonical form: only what parses as a universe
	// ever reaches disk or another viewer.
	data, err := json.Marshal(u)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "marshal failed")
		return
	}
	if err := os.WriteFile(s.path(id), data, 0o644); err != nil {
		httpError(w, http.StatusInternalServerError, "store failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "path": "/u/" + id})
}

func (s *server) fetch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !idPattern.MatchString(id) {
		httpError(w, http.StatusBadRequest, "malformed universe id")
		return
	}
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		httpError(w, http.StatusNotFound, "no universe with that id")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // snapshots never change
	_, _ = w.Write(data)
}

// spaHandler serves the built app, falling back to index.html for the /u/…
// share routes (the app itself resolves the id client-side).
func spaHandler(dist string) http.Handler {
	fs := http.FileServer(http.Dir(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(r.URL.Path)
		if strings.HasPrefix(clean, "/u/") || clean == "/u" {
			http.ServeFile(w, r, filepath.Join(dist, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func newID() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b), nil
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
