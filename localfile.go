package main

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// localFileHandler 通过 /localfile?path=... 向前端提供任意本地文件，
// 使用 http.ServeContent 支持 Range 请求（视频 seek 必须）。
type localFileHandler struct{}

func (h *localFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/localfile") {
		http.NotFound(w, r)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	f, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if mt := mime.TypeByExtension(ext); mt != "" {
		w.Header().Set("Content-Type", mt)
	}

	http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
}
