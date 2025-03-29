package qrcode

import (
	"github.com/skip2/go-qrcode"
)

// Generator handles QR code generation
type Generator struct {
	baseURL string
}

// NewGenerator creates a new QR code generator
func NewGenerator(baseURL string) *Generator {
	return &Generator{
		baseURL: baseURL,
	}
}

// GenerateQRCode generates a QR code for a short URL
func (g *Generator) GenerateQRCode(shortCode string, size int) ([]byte, error) {
	// Combine base URL with short code
	targetURL := g.baseURL + "/" + shortCode
	
	// Generate QR code as PNG
	var png []byte
	png, err := qrcode.Encode(targetURL, qrcode.Medium, size)
	if err != nil {
		return nil, err
	}
	
	return png, nil
} 