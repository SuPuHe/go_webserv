package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"strings"
	"strconv"
)

type WebHandler struct {
	Config ServerConfig
}

func (h *WebHandler) findLocation(path string) (LocationConfig, bool) {
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
	return result, found
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

func (h *WebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	location, found := h.findLocation(r.URL.Path)
	if !found {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if !h.isMethodAllowed(r.Method, location) {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.ContentLength > h.parseMaxBody(h.Config.ClientMaxBodySize) {
		http.Error(w, "Payload Too Large", http.StatusRequestEntityTooLarge)
		return
	}

	if location.CGIExtension != "" && strings.HasSuffix(r.URL.Path, location.CGIExtension) {
		h.handleCGI(w, r, location)
		return
	}

	http.ServeFile(w, r, location.Root + r.URL.Path)
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
