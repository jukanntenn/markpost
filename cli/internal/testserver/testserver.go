// Package testserver is a hand-rolled stand-in for the markpost backend,
// mounted on an httptest.Server: it issues one fixed user's session, tracks
// the tokens it considers valid (so the 401→refresh→retry dance is
// observable), filters posts by the search parameter, and records every
// request. Real HTTP keeps the client under test honest (URL building,
// headers, body encoding) the way gh's cache tests use httptest.
package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const (
	Username = "alice"
	Password = "wonderland"
	PostKey  = "mpk-test"
)

// Server is the fake markpost. All mutating access is mutex-guarded because
// tests assert state while the handler goroutine may still be finishing.
type Server struct {
	mu           sync.Mutex
	access       string
	refresh      string
	refreshCalls int
	requests     []string
	posts        map[string]post
	nextQID      int
}

type post struct {
	Title     string
	Body      string
	CreatedAt string
}

func New() *Server {
	return &Server{
		access:  "at-1",
		refresh: "rt-1",
		posts: map[string]post{
			"p-existing": {Title: "Existing", Body: "Body", CreatedAt: "2026-09-01T10:00:00Z"},
		},
		nextQID: 2,
	}
}

// Start mounts the handler and registers cleanup; the returned URL is the
// server's base URL.
func (s *Server) Start(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(s.Handler())
	t.Cleanup(server.Close)
	return server.URL
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(v)
	}
	writeErr := func(w http.ResponseWriter, status int, code, message string) {
		writeJSON(w, status, map[string]any{"code": code, "message": message})
	}

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		s.record("POST /api/v1/auth/login")
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Username != Username || req.Password != Password {
			writeErr(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user": map[string]any{
				"id": 1, "username": Username, "name": "Alice", "role": "user", "email": "a@x.io",
			},
			"token":         s.currentAccess(),
			"refresh_token": s.currentRefresh(),
			"expires_in":    3600,
		})
	})
	mux.HandleFunc("/api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if !s.rotate(req.RefreshToken) {
			writeErr(w, http.StatusUnauthorized, "invalid_token", "refresh token is invalid")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"token":         s.currentAccess(),
			"refresh_token": s.currentRefresh(),
			"expires_in":    3600,
		})
	})
	mux.HandleFunc("/api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		s.record("POST /api/v1/auth/logout")
		if !s.authed(r) {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
	})
	mux.HandleFunc("/api/v1/post-key", func(w http.ResponseWriter, r *http.Request) {
		s.record("GET /api/v1/post-key")
		if !s.authed(r) {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"post_key": PostKey, "created_at": "2026-09-03T00:00:00Z"})
	})
	mux.HandleFunc("/api/v1/post-key/rotate", func(w http.ResponseWriter, r *http.Request) {
		if !s.authed(r) {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"post_key": "mpk-rotated"})
	})
	mux.HandleFunc("/api/v1/me/retention", func(w http.ResponseWriter, r *http.Request) {
		s.record("GET /api/v1/me/retention")
		if !s.authed(r) {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"posts_days": 7, "history_days": 30})
	})
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/ready", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": "v0.1.0"})
	})
	mux.HandleFunc("/api/v1/posts", func(w http.ResponseWriter, r *http.Request) {
		if !s.authed(r) {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "missing or invalid token")
			return
		}
		s.record("GET /api/v1/posts?" + r.URL.RawQuery)
		items := make([]map[string]any, 0)
		search := r.URL.Query().Get("search")
		for qid, p := range s.snapshotPosts() {
			if search != "" && !strings.Contains(strings.ToLower(p.Title), strings.ToLower(search)) {
				continue
			}
			items = append(items, map[string]any{
				"id": 1, "qid": qid, "title": p.Title, "created_at": p.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items": items, "total": len(items), "page": 1, "limit": 20, "total_pages": 1,
		})
	})
	mux.HandleFunc("/"+PostKey, func(w http.ResponseWriter, r *http.Request) {
		s.record(r.Method + " /" + PostKey)
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusNotFound, "not_found", "post not found")
			return
		}
		var req struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		qid := s.createPost(req.Title, req.Body)
		writeJSON(w, http.StatusCreated, map[string]string{"id": qid})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.record(r.Method + " " + r.URL.Path)
		qid := strings.TrimPrefix(r.URL.Path, "/")
		qid = strings.TrimPrefix(qid, "api/v1/posts/")
		if r.Method == http.MethodDelete {
			if !s.deletePost(qid) {
				writeErr(w, http.StatusNotFound, "not_found", "post not found")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		p, ok := s.getPost(qid)
		if !ok {
			writeErr(w, http.StatusNotFound, "not_found", "post not found")
			return
		}
		if r.URL.Query().Get("format") == "raw" {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			_, _ = w.Write([]byte("# " + p.Title + "\n\n" + p.Body))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><h1>" + p.Title + "</h1></body></html>"))
	})
	return mux
}

// --- state helpers ---

func (s *Server) record(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, line)
}

func (s *Server) authed(r *http.Request) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return r.Header.Get("Authorization") == "Bearer "+s.access
}

func (s *Server) currentAccess() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.access
}

func (s *Server) currentRefresh() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refresh
}

// rotate simulates a refresh: it succeeds exactly when the presented refresh
// token is current, then issues the next pair. The request log is appended
// under the already-held lock — calling record() here would self-deadlock.
func (s *Server) rotate(presented string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, "POST /api/v1/auth/refresh")
	s.refreshCalls++
	if presented != s.refresh {
		return false
	}
	s.access = fmt.Sprintf("at-%d", s.refreshCalls+1)
	s.refresh = fmt.Sprintf("rt-%d", s.refreshCalls+1)
	return true
}

func (s *Server) createPost(title, body string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	qid := fmt.Sprintf("p-%d", s.nextQID)
	s.nextQID++
	s.posts[qid] = post{Title: title, Body: body, CreatedAt: "2026-09-03T00:00:00Z"}
	return qid
}

func (s *Server) getPost(qid string) (post, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.posts[qid]
	return p, ok
}

func (s *Server) deletePost(qid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.posts[qid]; !ok {
		return false
	}
	delete(s.posts, qid)
	return true
}

func (s *Server) snapshotPosts() map[string]post {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]post, len(s.posts))
	for k, v := range s.posts {
		out[k] = v
	}
	return out
}

// --- assertions for tests ---

func (s *Server) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func (s *Server) RefreshCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refreshCalls
}

func (s *Server) Tokens() (access, refresh string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.access, s.refresh
}

func (s *Server) HasPost(qid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.posts[qid]
	return ok
}

// ValidAccess is the currently accepted bearer token.
func (s *Server) ValidAccess() string { return s.currentAccess() }

// ValidRefresh is the currently accepted refresh token.
func (s *Server) ValidRefresh() string { return s.currentRefresh() }
