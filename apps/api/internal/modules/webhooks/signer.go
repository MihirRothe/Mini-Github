package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SignPayload computes HMAC-SHA256 signature in the format: sha256=<hex>
func SignPayload(secret string, payload []byte) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	return "sha256=" + hex.EncodeToString(expectedMAC)
}

// VerifySignature checks if the provided signature matches the HMAC-SHA256 signature of the payload
func VerifySignature(secret string, payload []byte, signature string) bool {
	if secret == "" {
		return signature == ""
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sigHex := strings.TrimPrefix(signature, "sha256=")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)

	return hmac.Equal(sigBytes, expectedMAC)
}
