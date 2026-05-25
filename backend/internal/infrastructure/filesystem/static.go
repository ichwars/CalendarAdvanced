package filesystem

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type SPAHandler struct {
	Dir string
}

func (h SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	path := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if path == "." || path == "" {
		path = "index.html"
	}
	full := filepath.Join(h.Dir, path)
	if !strings.HasPrefix(full, filepath.Clean(h.Dir)) {
		http.NotFound(w, r)
		return
	}
	if info, err := os.Stat(full); err == nil && !info.IsDir() {
		http.ServeFile(w, r, full)
		return
	}
	http.ServeFile(w, r, filepath.Join(h.Dir, "index.html"))
}
