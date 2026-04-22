package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"strings"
	"strconv"
	"path/filepath"
	"os"
	"io"
)

type WebHandler struct {
	Config ServerConfig
}

func (h *WebHandler) findLocation(path string) (LocationConfig, string, bool) {
	var bestMatch string
	var result LocationConfig
	found := false

	for prefix, loc := range h.Config.Locations {
		if strings.HasPrefix(path, prefix) {
			if len(prefix) > len(bestMatch) {
				bestMatch = prefix
				result = loc
				found = true
			}
		}
	}
	return result, bestMatch, found
}

func (h *WebHandler) isMethodAllowed(method string, loc LocationConfig) bool {
	if len(loc.Methods) == 0 {
		return true
	}
	for _, m := range loc.Methods {
		if m == method {
			return true
		}
	}
	return false
}

func (h *WebHandler) parseMaxBody(s string) int64 {
	if s == "" {
		return 10 * 1024 * 1024
	}

	unit := s[len(s)-1:]
	val, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		return 0
	}

	switch strings.ToUpper(unit) {
	case "K":
		return val * 1024
	case "M":
		return val * 1024 * 1024
	case "G":
		return val * 1024 * 1024 * 1024
	default:
		res, _ := strconv.ParseInt(s, 10, 64)
		return res
	}
}

func (h *WebHandler) handleCGI(w http.ResponseWriter, r *http.Request, loc LocationConfig) {
	fmt.Fprintf(w, "CGI handler for %s is not implemented yet\n", loc.CGIPath)
}

func (h *WebHandler) generateAutoindex(w http.ResponseWriter, r *http.Request, directory string) {
	files, err := os.ReadDir(directory)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<html><body><h1>Index of %s</h1><hr><ul>", r.URL.Path)

	fmt.Fprintf(w, "<li><a href=\"..\">..</a></li>")

	for _, file := range files {
		name := file.Name()
		if file.IsDir() {
			name += "/"
		}

		link := filepath.Join(r.URL.Path, name)
		fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>", link, name)
	}

	fmt.Fprintf(w, "</ul><hr></body></html>")
}

func (h *WebHandler) sendError(w http.ResponseWriter, code int) {
	if path, ok := h.Config.ErrorPages[strconv.Itoa(code)]; ok {
		content, err := os.ReadFile(path)
		if err == nil {
			w.WriteHeader(code)
			w.Write(content)
			return
		}
	}
	http.Error(w, http.StatusText(code), code)
}

func (h *WebHandler) handleUpload(w http.ResponseWriter, r *http.Request, loc LocationConfig) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		h.sendError(w, http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		h.sendError(w, http.StatusBadRequest)
		return
	}
	defer file.Close()

	dstPath := filepath.Join(loc.UploadDir, handler.Filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		log.Printf("Ошибка создания файла: %v", err)
		h.sendError(w, http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.sendError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "File uploaded successfully: %s", handler.Filename)
}

func (h *WebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	location, matchedPrefix, found := h.findLocation(r.URL.Path)
	if !found {
		h.sendError(w, http.StatusNotFound)
		return
	}

	if !h.isMethodAllowed(r.Method, location) {
		h.sendError(w, http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength > h.parseMaxBody(h.Config.ClientMaxBodySize) {
		h.sendError(w, http.StatusRequestEntityTooLarge)
		return
	}

	if r.Method == "POST" && location.UploadDir != "" {
		h.handleUpload(w, r, location)
		return
	}

	if location.CGIExtension != "" && strings.HasSuffix(r.URL.Path, location.CGIExtension) {
		h.handleCGI(w, r, location)
		return
	}

	relPath := strings.TrimPrefix(r.URL.Path, matchedPrefix)
	relPath = strings.TrimLeft(relPath, "/")
	fullPath := filepath.Join(location.Root, relPath)

	stat, err := os.Stat(fullPath)
	if err != nil {
		h.sendError(w, http.StatusNotFound)
		return
	}

	if stat.IsDir() {
		indexName := "index.html"
		indexPath := filepath.Join(fullPath, indexName)

		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		if location.Autoindex {
			h.generateAutoindex(w, r, fullPath)
			return
		}

		h.sendError(w, http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func main() {

	cfg, err := LoadConfig("config/default.toml")
	if err != nil {
		log.Fatalf("Error with loading config file: %v", err)
	}

	cfg.PrettyPrint()

	var wg sync.WaitGroup

	for _, srvCfg := range cfg.Servers {
		wg.Add(1)

		go func (sc ServerConfig) {
			defer wg.Done()

			handler := &WebHandler{Config: sc}

			server := &http.Server{
				Addr: fmt.Sprintf(":%d", sc.Listen),
				Handler: handler,
			}

			fmt.Printf("Starting server [%s] on port %d...\n", sc.ServerName, sc.Listen)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Server %d error: %v\n", sc.Listen, err)
			}
		}(srvCfg)
	}
	wg.Wait()
}
