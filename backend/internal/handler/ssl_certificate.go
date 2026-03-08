package handler

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

const (
	configKeySSLCert = "ssl.certificate"
	configKeySSLKey  = "ssl.private_key"
)

type SSLCertificateHandler struct {
	svc service.SystemConfigService
}

func NewSSLCertificateHandler(svc service.SystemConfigService) *SSLCertificateHandler {
	return &SSLCertificateHandler{svc: svc}
}

type certMeta struct {
	Subject   string    `json:"subject"`
	Issuer    string    `json:"issuer"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	SANs      []string  `json:"sans"`
}

func parseCertMeta(pemData string) *certMeta {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil
	}
	sans := cert.DNSNames
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}
	return &certMeta{
		Subject:   cert.Subject.String(),
		Issuer:    cert.Issuer.String(),
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		SANs:      sans,
	}
}

// Get returns the stored certificate (PEM) and its parsed metadata.
// The private key is never returned; only whether one is stored.
func (h *SSLCertificateHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()

	certPEM := ""
	if raw, err := h.svc.Get(ctx, configKeySSLCert); err == nil {
		_ = json.Unmarshal(raw, &certPEM)
	}

	hasKey := false
	if raw, err := h.svc.Get(ctx, configKeySSLKey); err == nil {
		var keyStr string
		if json.Unmarshal(raw, &keyStr) == nil && strings.TrimSpace(keyStr) != "" {
			hasKey = true
		}
	}

	result := gin.H{
		"cert":    certPEM,
		"has_key": hasKey,
		"meta":    parseCertMeta(certPEM),
	}
	response.Success(c, result)
}

// Set saves a new certificate and private key (both PEM-encoded).
func (h *SSLCertificateHandler) Set(c *gin.Context) {
	var body struct {
		Cert string `json:"cert"`
		Key  string `json:"key"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	cert := strings.TrimSpace(body.Cert)
	key := strings.TrimSpace(body.Key)

	// Validate certificate PEM
	if cert != "" {
		block, _ := pem.Decode([]byte(cert))
		if block == nil {
			response.Fail(c, http.StatusBadRequest, "证书不是有效的 PEM 格式")
			return
		}
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			response.Fail(c, http.StatusBadRequest, "无法解析证书: "+err.Error())
			return
		}
	}

	// Validate private key PEM (basic check: decodable)
	if key != "" {
		block, _ := pem.Decode([]byte(key))
		if block == nil {
			response.Fail(c, http.StatusBadRequest, "私钥不是有效的 PEM 格式")
			return
		}
	}

	certJSON, _ := json.Marshal(cert)
	if err := h.svc.Set(c.Request.Context(), configKeySSLCert, certJSON); err != nil {
		response.FailWithError(c, err)
		return
	}

	keyJSON, _ := json.Marshal(key)
	if err := h.svc.Set(c.Request.Context(), configKeySSLKey, keyJSON); err != nil {
		response.FailWithError(c, err)
		return
	}

	response.Success(c, gin.H{"saved": true, "meta": parseCertMeta(cert)})
}

// Delete clears both the certificate and private key.
func (h *SSLCertificateHandler) Delete(c *gin.Context) {
	empty, _ := json.Marshal("")
	ctx := c.Request.Context()
	_ = h.svc.Set(ctx, configKeySSLCert, empty)
	_ = h.svc.Set(ctx, configKeySSLKey, empty)
	response.Success(c, gin.H{"deleted": true})
}
