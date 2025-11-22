package activitypub

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPSignature represents an HTTP signature for ActivityPub requests
type HTTPSignature struct {
	KeyID     string
	Algorithm string
	Headers   []string
	Signature string
}

// SignRequest signs an HTTP request with the given private key
func SignRequest(r *http.Request, privateKeyPEM string, keyID string) error {
	// Get current date for Date header
	now := time.Now().UTC().Format(http.TimeFormat)
	r.Header.Set("Date", now)

	// Calculate digest for POST/PUT requests
	if r.Body != nil && (r.Method == "POST" || r.Method == "PUT") {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}

		// Calculate SHA-256 digest
		h := sha256.New()
		h.Write(bodyBytes)
		digest := base64.StdEncoding.EncodeToString(h.Sum(nil))
		r.Header.Set("Digest", "SHA-256="+digest)

		// Reset body for actual request
		r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		r.ContentLength = int64(len(bodyBytes))
	}

	// Build signing string
	var signingParts []string
	headers := []string{"(request-target)", "host", "date"}

	// Add digest header if present
	if r.Header.Get("Digest") != "" {
		headers = append(headers, "digest")
	}

	for _, header := range headers {
		var value string
		if header == "(request-target)" {
			value = fmt.Sprintf("%s %s", strings.ToLower(r.Method), r.URL.Path)
		} else if header == "host" {
			value = r.Host
			if value == "" {
				value = r.URL.Host
			}
		} else {
			value = r.Header.Get(header)
		}
		signingParts = append(signingParts, fmt.Sprintf("%s: %s", header, value))
	}

	signingString := strings.Join(signingParts, "\n")

	// Sign the string using RSA-SHA256
	signature, err := signString(signingString, privateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to sign string: %w", err)
	}

	// Build Signature header
	signatureHeader := fmt.Sprintf(
		`keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`,
		keyID,
		strings.Join(headers, " "),
		signature,
	)

	r.Header.Set("Signature", signatureHeader)
	return nil
}

// VerifyRequest verifies an HTTP signature on an incoming request
func VerifyRequest(r *http.Request, publicKeyPEM string) error {
	// Parse Signature header
	sigHeader := r.Header.Get("Signature")
	if sigHeader == "" {
		return fmt.Errorf("missing Signature header")
	}

	sig, err := parseSignatureHeader(sigHeader)
	if err != nil {
		return fmt.Errorf("failed to parse signature: %w", err)
	}

	// Build signing string from headers
	var signingParts []string
	for _, header := range sig.Headers {
		var value string
		if header == "(request-target)" {
			value = fmt.Sprintf("%s %s", strings.ToLower(r.Method), r.URL.Path)
		} else {
			value = r.Header.Get(http.CanonicalHeaderKey(header))
			if value == "" {
				return fmt.Errorf("missing header: %s", header)
			}
		}
		signingParts = append(signingParts, fmt.Sprintf("%s: %s", header, value))
	}

	signingString := strings.Join(signingParts, "\n")

	// Verify the signature
	return verifyString(signingString, sig.Signature, publicKeyPEM)
}

// parseSignatureHeader parses the Signature header into components
func parseSignatureHeader(header string) (*HTTPSignature, error) {
	sig := &HTTPSignature{}
	parts := strings.Split(header, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := kv[0]
		value := strings.Trim(kv[1], `"`)

		switch key {
		case "keyId":
			sig.KeyID = value
		case "algorithm":
			sig.Algorithm = value
		case "headers":
			sig.Headers = strings.Split(value, " ")
		case "signature":
			sig.Signature = value
		}
	}

	if sig.KeyID == "" || sig.Signature == "" {
		return nil, fmt.Errorf("invalid signature header: missing keyId or signature")
	}

	return sig, nil
}

// signString signs a string using RSA-SHA256
func signString(data string, privateKeyPEM string) (string, error) {
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)

	signature, err := rsaSign(privateKey, hashed)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifyString verifies a signature using RSA-SHA256
func verifyString(data string, signature string, publicKeyPEM string) error {
	publicKey, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return err
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)

	return rsaVerify(publicKey, hashed, sigBytes)
}

// FetchActor fetches an ActivityPub actor from a remote server
func FetchActor(actorURL string, privateKeyPEM string, keyID string) (map[string]any, error) {
	req, err := http.NewRequest("GET", actorURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/activity+json, application/ld+json")
	req.Header.Set("User-Agent", "terminalpub/1.0")

	// Sign the request
	if err := SignRequest(req, privateKeyPEM, keyID); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch actor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var actor map[string]any
	if err := parseJSON(resp.Body, &actor); err != nil {
		return nil, fmt.Errorf("failed to parse actor: %w", err)
	}

	return actor, nil
}

// ResolveWebFinger resolves a WebFinger query for an actor
func ResolveWebFinger(username, domain string) (string, error) {
	webfingerURL := fmt.Sprintf("https://%s/.well-known/webfinger?resource=acct:%s@%s",
		domain, username, domain)

	req, err := http.NewRequest("GET", webfingerURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create webfinger request: %w", err)
	}

	req.Header.Set("Accept", "application/jrd+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch webfinger: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("webfinger returned status: %d", resp.StatusCode)
	}

	var result map[string]any
	if err := parseJSON(resp.Body, &result); err != nil {
		return "", fmt.Errorf("failed to parse webfinger response: %w", err)
	}

	// Extract actor URL from links
	links, ok := result["links"].([]any)
	if !ok {
		return "", fmt.Errorf("no links in webfinger response")
	}

	for _, link := range links {
		linkMap, ok := link.(map[string]any)
		if !ok {
			continue
		}

		rel, _ := linkMap["rel"].(string)
		linkType, _ := linkMap["type"].(string)
		href, _ := linkMap["href"].(string)

		if rel == "self" && linkType == "application/activity+json" && href != "" {
			return href, nil
		}
	}

	return "", fmt.Errorf("no ActivityPub actor link found in webfinger")
}

// NormalizeActorID normalizes an actor identifier (username@domain or URL) to an ActivityPub actor URL
func NormalizeActorID(identifier string) (string, error) {
	// If it's already a URL, return it
	if strings.HasPrefix(identifier, "http://") || strings.HasPrefix(identifier, "https://") {
		return identifier, nil
	}

	// Parse as username@domain
	parts := strings.Split(identifier, "@")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid actor identifier: %s", identifier)
	}

	username := parts[0]
	domain := parts[1]

	// Resolve via WebFinger
	return ResolveWebFinger(username, domain)
}

// GetActorInbox extracts the inbox URL from an actor object
func GetActorInbox(actor map[string]any) (string, error) {
	// Try shared inbox first
	if endpoints, ok := actor["endpoints"].(map[string]any); ok {
		if sharedInbox, ok := endpoints["sharedInbox"].(string); ok && sharedInbox != "" {
			return sharedInbox, nil
		}
	}

	// Fall back to regular inbox
	inbox, ok := actor["inbox"].(string)
	if !ok || inbox == "" {
		return "", fmt.Errorf("no inbox found in actor")
	}

	return inbox, nil
}

// IsPublicAddress checks if an address is the ActivityPub public collection
func IsPublicAddress(addr string) bool {
	return addr == "https://www.w3.org/ns/activitystreams#Public" ||
		addr == "as:Public" ||
		addr == "Public"
}

// NormalizeURL ensures a URL has a scheme
func NormalizeURL(rawURL string) string {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return "https://" + rawURL
	}
	return rawURL
}

// ExtractDomain extracts the domain from a URL
func ExtractDomain(rawURL string) (string, error) {
	u, err := url.Parse(NormalizeURL(rawURL))
	if err != nil {
		return "", err
	}
	return u.Host, nil
}
