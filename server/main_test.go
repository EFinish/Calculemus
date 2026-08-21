package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/EFinish/Calculemus/core"
)

func testMux(t *testing.T) *http.ServeMux {
	t.Helper()
	s := &server{dataDir: t.TempDir()}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/universes", s.publish)
	mux.HandleFunc("GET /api/universes/{id}", s.fetch)
	return mux
}

func ballJSON(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../examples/ball.json")
	if err != nil {
		t.Fatalf("read example: %v", err)
	}
	return data
}

func TestPublishAndFetchRoundTrip(t *testing.T) {
	mux := testMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/universes", bytes.NewReader(ballJSON(t))))
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish: status %d, body %s", rec.Code, rec.Body)
	}
	var resp struct{ ID, Path string }
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !idPattern.MatchString(resp.ID) || resp.Path != "/u/"+resp.ID {
		t.Fatalf("bad publish response: %+v", resp)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/universes/"+resp.ID, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch: status %d", rec.Code)
	}
	var u core.Universe
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatal(err)
	}
	if u.Title != "Is it time to play?" || len(u.Statements) != 3 {
		t.Fatalf("round-trip mangled the universe: %+v", u)
	}
	// The stored snapshot must still evaluate — the whole point of sharing.
	if _, err := core.Evaluate(&u); err != nil {
		t.Fatalf("shared universe does not evaluate: %v", err)
	}
}

func TestPublishRejectsGarbage(t *testing.T) {
	mux := testMux(t)
	cases := map[string]struct {
		body string
		want int
	}{
		"not json":        {"{nope", http.StatusBadRequest},
		"missing version": {`{"title":"x","statements":[]}`, http.StatusUnprocessableEntity},
		"dangling ref": {
			`{"version":1,"title":"x","statements":[{"id":"s1","text":"a"}],
			  "formulas":[{"id":"f1","op":"NOT","args":["ghost"]}]}`,
			http.StatusUnprocessableEntity,
		},
	}
	for name, tc := range cases {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/universes", strings.NewReader(tc.body)))
		if rec.Code != tc.want {
			t.Errorf("%s: status %d, want %d (body %s)", name, rec.Code, tc.want, rec.Body)
		}
	}
}

func TestPublishBodyLimit(t *testing.T) {
	mux := testMux(t)
	big := append([]byte(`{"version":1,"title":"`), bytes.Repeat([]byte("x"), maxUniverseBytes)...)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/universes", bytes.NewReader(big)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized body: status %d, want 400", rec.Code)
	}
}

func TestFetchRejectsBadIDs(t *testing.T) {
	mux := testMux(t)
	for id, want := range map[string]int{
		"nope":          http.StatusBadRequest, // wrong shape
		"..%2fetc":      http.StatusBadRequest, // traversal attempt
		"aaaaaaaaaa":    http.StatusNotFound,   // well-formed, absent
		"AAAAAAAAAA":    http.StatusBadRequest, // uppercase not in alphabet
		"aaaaaaaaaaa":   http.StatusBadRequest, // too long
		"a a a a a a a": http.StatusBadRequest,
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/universes/"+strings.ReplaceAll(id, " ", "%20"), nil))
		if rec.Code != want {
			t.Errorf("id %q: status %d, want %d", id, rec.Code, want)
		}
	}
}
