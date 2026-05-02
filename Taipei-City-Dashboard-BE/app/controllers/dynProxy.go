// Package controllers stores all the controllers for the Gin router.
package controllers

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

/*
dynProxy.go implements a generic, runtime-registered reverse proxy.

Control endpoints (statically registered with Gin so they always resolve):
  POST /api/v1/proxy/register  { "name":"chat", "addr":"host:port",
                                 "prefix":"/api/v1/chat",
                                 "upstream":"/api/dev/chat",
                                 "ping":"/api/dev/ping" }
  POST /api/v1/proxy/detach    { "name":"chat" }
  GET  /api/v1/proxy/list

prefix / upstream / ping are optional. Defaults derived from name:
  prefix   = "/api/v1/<name>"
  upstream = "/api/dev/<name>"
  ping     = "/api/dev/ping"

Dispatching is done from Gin's NoRoute fallback (DynProxyNoRoute), so any
path that is statically registered in Go (e.g. /api/v1/component) takes
priority over a dynamically-registered proxy with the same prefix. A
registration that collides is silently shadowed by the static route.
*/

type proxyEntry struct {
	Name     string
	Prefix   string // request path prefix on this BE, e.g. "/api/v1/chat"
	Upstream string // remote path prefix on target,    e.g. "/api/dev/chat"
	Target   *url.URL
}

var (
	proxyMu      sync.RWMutex
	proxyEntries = map[string]*proxyEntry{}
)

type proxyRegisterRequest struct {
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	Prefix   string `json:"prefix"`
	Upstream string `json:"upstream"`
	Ping     string `json:"ping"`
}

type proxyDetachRequest struct {
	Name string `json:"name"`
}

func normalizeBase(addr string) (*url.URL, string, error) {
	base := strings.TrimSpace(addr)
	if base == "" {
		return nil, "", errBadAddr("addr is required")
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	base = strings.TrimRight(base, "/")
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return nil, "", errBadAddr("invalid addr")
	}
	return &url.URL{Scheme: u.Scheme, Host: u.Host}, base, nil
}

type errBadAddr string

func (e errBadAddr) Error() string { return string(e) }

func normalizePath(p, fallback string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		p = fallback
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimRight(p, "/")
}

// DynProxyRegister verifies a downstream and stores it as a proxy entry.
func DynProxyRegister(c *gin.Context) {
	var req proxyRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "name is required"})
		return
	}
	target, base, err := normalizeBase(req.Addr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}
	prefix := normalizePath(req.Prefix, "/api/v1/"+name)
	upstream := normalizePath(req.Upstream, "/api/dev/"+name)
	ping := normalizePath(req.Ping, "/api/dev/ping")

	pingURL := base + ping
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(pingURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"status": "error", "message": "ping failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"status": "error", "message": "ping read failed: " + err.Error()})
		return
	}
	if strings.TrimSpace(string(body)) != "pong" {
		c.JSON(http.StatusBadGateway, gin.H{"status": "error", "message": "ping body != pong"})
		return
	}

	entry := &proxyEntry{
		Name:     name,
		Prefix:   prefix,
		Upstream: upstream,
		Target:   target,
	}
	proxyMu.Lock()
	proxyEntries[name] = entry
	proxyMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"name":     name,
		"target":   target.String(),
		"prefix":   prefix,
		"upstream": upstream,
	})
}

// DynProxyDetach removes a proxy entry by name.
func DynProxyDetach(c *gin.Context) {
	var req proxyDetachRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "name is required"})
		return
	}
	proxyMu.Lock()
	prev, ok := proxyEntries[name]
	delete(proxyEntries, name)
	proxyMu.Unlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "no such entry"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"detached": prev.Name,
		"target":   prev.Target.String(),
	})
}

// DynProxyList returns all registered proxy entries.
func DynProxyList(c *gin.Context) {
	proxyMu.RLock()
	defer proxyMu.RUnlock()
	out := make([]gin.H, 0, len(proxyEntries))
	for _, e := range proxyEntries {
		out = append(out, gin.H{
			"name":     e.Name,
			"target":   e.Target.String(),
			"prefix":   e.Prefix,
			"upstream": e.Upstream,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": out})
}

// matchEntry returns the entry whose Prefix matches the request path
// (longest prefix wins). A match requires the path to either equal the
// prefix or have it followed by '/'.
func matchEntry(reqPath string) (*proxyEntry, string) {
	proxyMu.RLock()
	defer proxyMu.RUnlock()
	var best *proxyEntry
	for _, e := range proxyEntries {
		if reqPath == e.Prefix || strings.HasPrefix(reqPath, e.Prefix+"/") {
			if best == nil || len(e.Prefix) > len(best.Prefix) {
				best = e
			}
		}
	}
	if best == nil {
		return nil, ""
	}
	sub := strings.TrimPrefix(reqPath, best.Prefix)
	return best, sub
}

// DynProxyNoRoute is mounted as Gin's NoRoute fallback. If the request
// path matches a registered proxy prefix, it is forwarded; otherwise
// the standard 404 is returned.
func DynProxyNoRoute(c *gin.Context) {
	entry, sub := matchEntry(c.Request.URL.Path)
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "not found"})
		return
	}

	target := entry.Target
	upstream := entry.Upstream
	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = target.Host
		path := upstream + sub
		req.URL.Path = path
		req.URL.RawPath = ""
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"status":"error","message":"upstream error: ` + err.Error() + `"}`))
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}
