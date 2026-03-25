package main

import (
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	cacheControlNoCache         = "no-cache"
	cacheControlImmutableAssets = "public, max-age=31536000, immutable"
)

type staticFile struct {
	diskPath        string
	contentType     string
	contentEncoding string
	cacheControl    string
}

func resolveStaticFile(distDir, method, requestPath, acceptHeader, acceptEncoding string) (staticFile, bool) {
	relativePath := normalizeStaticRequestPath(requestPath)
	if strings.HasPrefix(relativePath, "api/") {
		return staticFile{}, false
	}

	if file, ok := resolveExactStaticFile(distDir, relativePath, acceptEncoding); ok {
		return file, true
	}

	if !shouldServeSPAFallback(method, relativePath, acceptHeader) {
		return staticFile{}, false
	}

	return resolveExactStaticFile(distDir, "index.html", acceptEncoding)
}

func normalizeStaticRequestPath(requestPath string) string {
	cleanedPath := path.Clean("/" + strings.TrimSpace(requestPath))
	relativePath := strings.TrimPrefix(cleanedPath, "/")
	if relativePath == "" || relativePath == "." {
		return "index.html"
	}
	return relativePath
}

func resolveExactStaticFile(distDir, relativePath, acceptEncoding string) (staticFile, bool) {
	diskPath := filepath.Join(distDir, filepath.FromSlash(relativePath))
	if !hasRegularFile(diskPath) {
		return staticFile{}, false
	}

	selectedPath, contentEncoding := selectCompressedStaticFile(diskPath, acceptEncoding)
	return staticFile{
		diskPath:        selectedPath,
		contentType:     mime.TypeByExtension(path.Ext(relativePath)),
		contentEncoding: contentEncoding,
		cacheControl:    cacheControlForStaticPath(relativePath),
	}, true
}

func shouldServeSPAFallback(method, relativePath, acceptHeader string) bool {
	if method != http.MethodGet && method != http.MethodHead {
		return false
	}
	if strings.HasPrefix(relativePath, "api/") {
		return false
	}
	if path.Ext(relativePath) != "" {
		return false
	}

	acceptHeader = strings.ToLower(strings.TrimSpace(acceptHeader))
	return acceptHeader == "" || strings.Contains(acceptHeader, "text/html")
}

func selectCompressedStaticFile(diskPath, acceptEncoding string) (string, string) {
	if acceptsEncoding(acceptEncoding, "br") && hasRegularFile(diskPath+".br") {
		return diskPath + ".br", "br"
	}
	if acceptsEncoding(acceptEncoding, "gzip") && hasRegularFile(diskPath+".gz") {
		return diskPath + ".gz", "gzip"
	}
	return diskPath, ""
}

func acceptsEncoding(acceptEncoding, wanted string) bool {
	for _, entry := range strings.Split(acceptEncoding, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		name := entry
		params := ""
		if idx := strings.Index(entry, ";"); idx >= 0 {
			name = strings.TrimSpace(entry[:idx])
			params = entry[idx+1:]
		}
		if !strings.EqualFold(name, wanted) && name != "*" {
			continue
		}
		if !encodingEnabled(params) {
			continue
		}
		return true
	}

	return false
}

func encodingEnabled(params string) bool {
	for _, param := range strings.Split(params, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(param), "=")
		if !ok || !strings.EqualFold(key, "q") {
			continue
		}

		qValue, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return true
		}
		return qValue > 0
	}

	return true
}

func cacheControlForStaticPath(relativePath string) string {
	if relativePath == "index.html" {
		return cacheControlNoCache
	}
	if strings.HasPrefix(relativePath, "assets/") {
		return cacheControlImmutableAssets
	}
	return cacheControlNoCache
}

func hasRegularFile(filename string) bool {
	info, err := os.Stat(filename)
	return err == nil && !info.IsDir()
}

func applyStaticFileHeaders(c *app.RequestContext, file staticFile) {
	c.Header("Cache-Control", file.cacheControl)
	c.Header("Vary", "Accept-Encoding")
	c.Header("X-Content-Type-Options", "nosniff")
	if file.contentType != "" {
		c.Header("Content-Type", file.contentType)
	}
	if file.contentEncoding != "" {
		c.Header("Content-Encoding", file.contentEncoding)
	}
}
