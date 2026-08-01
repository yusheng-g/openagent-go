package huaweicloud

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Sign produces SDK-HMAC-SHA256 authorization headers for a HuaweiCloud API
// request. ak/sk/securityToken come from the process environment — never from
// the LLM. The returned headers should be merged into the HTTP request.
//
// This implements the same algorithm as the official HuaweiCloud SDK:
//
//	Authorization: SDK-HMAC-SHA256 Access={ak}, SignedHeaders={...}, Signature={...}
//	x-sdk-date: {timestamp}
//	[x-security-token: {token}]   (when temporary credentials are used)
func Sign(method, endpoint, path string, query map[string]string, body []byte, ak, sk, securityToken string) map[string]string {
	ts := timestamp()
	host := hostFromEndpoint(endpoint)

	// Canonical query string (sorted keys).
	cqs := canonicalQueryString(query)

	// Canonical URI.
	curi := canonicalURI(path)

	// Signed headers — include the security token when present so the
	// signature covers it and the server can validate the temporary
	// credential.
	signedHeaders := "host;x-sdk-date"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-sdk-date:%s\n", host, ts)
	if securityToken != "" {
		signedHeaders = "host;x-sdk-date;x-security-token"
		canonicalHeaders = fmt.Sprintf("host:%s\nx-sdk-date:%s\nx-security-token:%s\n", host, ts, securityToken)
	}

	// Payload hash.
	payloadHash := sha256Hex(body)

	// Canonical request.
	canonicalRequest := strings.Join([]string{
		method,
		curi,
		cqs,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// String to sign.
	stringToSign := fmt.Sprintf("SDK-HMAC-SHA256\n%s\n%s", ts, sha256Hex([]byte(canonicalRequest)))

	// Signature.
	signature := hmacSHA256(sk, stringToSign)

	headers := map[string]string{
		"host":       host,
		"x-sdk-date": ts,
		"Authorization": fmt.Sprintf(
			"SDK-HMAC-SHA256 Access=%s, SignedHeaders=%s, Signature=%s",
			ak, signedHeaders, signature,
		),
	}
	if securityToken != "" {
		headers["x-security-token"] = securityToken
	}
	return headers
}

// timestamp returns the current time in the format expected by HuaweiCloud:
// 20060102T150405Z (basic ISO 8601, UTC).
func timestamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

// hostFromEndpoint extracts the host (with port if present) from an endpoint URL.
func hostFromEndpoint(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		// Fallback: strip scheme and path.
		s := strings.TrimPrefix(endpoint, "https://")
		s = strings.TrimPrefix(s, "http://")
		if i := strings.IndexByte(s, '/'); i >= 0 {
			s = s[:i]
		}
		return s
	}
	return u.Host
}

// canonicalURI encodes each path segment and joins with /, with a trailing /.
// Matches the TypeScript reference:
//
//	"/" + segments.map(urlEncode).join("/") + "/"
//
// The root path ("/" or "") maps to "/" (not "//").
func canonicalURI(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "/"
	}
	segments := strings.Split(trimmed, "/")
	for i, s := range segments {
		segments[i] = urlEncode(s)
	}
	return "/" + strings.Join(segments, "/") + "/"
}

// canonicalQueryString sorts query keys and URL-encodes both keys and values.
func canonicalQueryString(query map[string]string) string {
	if len(query) == 0 {
		return ""
	}
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, urlEncode(k)+"="+urlEncode(query[k]))
	}
	return strings.Join(parts, "&")
}

// sha256Hex returns the hex-encoded SHA-256 digest of data.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// hmacSHA256 returns the hex-encoded HMAC-SHA256 of data keyed by key.
func hmacSHA256(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// urlEncode percent-encodes a string per RFC 3986. Unreserved characters
// (A-Za-z0-9-_.~) are not encoded; everything else is %XX with uppercase hex.
// This matches the TypeScript reference's urlEncode and differs from Go's
// url.QueryEscape (which encodes space as + and encodes /).
func urlEncode(s string) string {
	const hexUpper = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hexUpper[c>>4])
			b.WriteByte(hexUpper[c&0xF])
		}
	}
	return b.String()
}

// isUnreserved reports whether c is an RFC 3986 unreserved character.
func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '~'
}
