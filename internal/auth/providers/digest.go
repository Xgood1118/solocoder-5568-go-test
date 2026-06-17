package providers

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"apitester/internal/models"
)

type DigestProvider struct {
	config *models.DigestAuth
	mu     sync.Mutex
	nc     int
	cnonce string
}

type digestChallenge struct {
	realm     string
	nonce     string
	qop       string
	algorithm string
	opaque    string
	stale     string
}

func NewDigestProvider(config *models.DigestAuth) *DigestProvider {
	return &DigestProvider{
		config: config,
		nc:     0,
		cnonce: generateCNonce(),
	}
}

func generateCNonce() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func parseWWWAuthenticate(header string) (*digestChallenge, error) {
	if !strings.HasPrefix(header, "Digest ") {
		return nil, fmt.Errorf("invalid digest header")
	}

	header = strings.TrimPrefix(header, "Digest ")
	parts := strings.Split(header, ",")
	challenge := &digestChallenge{}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.Trim(strings.TrimSpace(kv[1]), "\"")

		switch key {
		case "realm":
			challenge.realm = value
		case "nonce":
			challenge.nonce = value
		case "qop":
			challenge.qop = value
		case "algorithm":
			challenge.algorithm = value
		case "opaque":
			challenge.opaque = value
		case "stale":
			challenge.stale = value
		}
	}

	if challenge.realm == "" || challenge.nonce == "" {
		return nil, fmt.Errorf("missing required digest parameters")
	}

	return challenge, nil
}

func md5Hash(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func computeHA1(username, realm, password, algorithm, nonce, cnonce string) string {
	ha1 := md5Hash(username + ":" + realm + ":" + password)

	if strings.EqualFold(algorithm, "MD5-sess") {
		ha1 = md5Hash(ha1 + ":" + nonce + ":" + cnonce)
	}

	return ha1
}

func computeHA2(method, uri, qop, entityBody string) string {
	if qop == "auth-int" {
		bodyHash := md5Hash(entityBody)
		return md5Hash(method + ":" + uri + ":" + bodyHash)
	}
	return md5Hash(method + ":" + uri)
}

func computeResponse(ha1, nonce, nc, cnonce, qop, ha2 string) string {
	if qop == "auth" || qop == "auth-int" {
		return md5Hash(ha1 + ":" + nonce + ":" + nc + ":" + cnonce + ":" + qop + ":" + ha2)
	}
	return md5Hash(ha1 + ":" + nonce + ":" + ha2)
}

func (p *DigestProvider) Apply(req *http.Request, tokenCache TokenCache, refresher TokenRefresher, cacheKey string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	challenge, err := p.getChallenge(req)
	if err != nil {
		return fmt.Errorf("failed to get digest challenge: %w", err)
	}

	p.nc++
	ncStr := fmt.Sprintf("%08x", p.nc)

	uri := req.URL.RequestURI()
	if uri == "" {
		uri = req.URL.Path
		if req.URL.RawQuery != "" {
			uri += "?" + req.URL.RawQuery
		}
	}

	method := req.Method
	var entityBody string
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		entityBody = string(bodyBytes)
		req.Body = io.NopCloser(strings.NewReader(entityBody))
	}

	ha1 := computeHA1(p.config.Username, challenge.realm, p.config.Password, challenge.algorithm, challenge.nonce, p.cnonce)
	ha2 := computeHA2(method, uri, challenge.qop, entityBody)
	response := computeResponse(ha1, challenge.nonce, ncStr, p.cnonce, challenge.qop, ha2)

	var authParts []string
	authParts = append(authParts, fmt.Sprintf("username=\"%s\"", p.config.Username))
	authParts = append(authParts, fmt.Sprintf("realm=\"%s\"", challenge.realm))
	authParts = append(authParts, fmt.Sprintf("nonce=\"%s\"", challenge.nonce))
	authParts = append(authParts, fmt.Sprintf("uri=\"%s\"", uri))
	authParts = append(authParts, fmt.Sprintf("response=\"%s\"", response))

	if challenge.algorithm != "" {
		authParts = append(authParts, fmt.Sprintf("algorithm=%s", challenge.algorithm))
	}

	if challenge.qop != "" {
		authParts = append(authParts, fmt.Sprintf("qop=%s", challenge.qop))
		authParts = append(authParts, fmt.Sprintf("nc=%s", ncStr))
		authParts = append(authParts, fmt.Sprintf("cnonce=\"%s\"", p.cnonce))
	}

	if challenge.opaque != "" {
		authParts = append(authParts, fmt.Sprintf("opaque=\"%s\"", challenge.opaque))
	}

	authHeader := "Digest " + strings.Join(authParts, ", ")
	req.Header.Set("Authorization", authHeader)

	return nil
}

func (p *DigestProvider) getChallenge(req *http.Request) (*digestChallenge, error) {
	headReq, err := http.NewRequest("HEAD", req.URL.String(), nil)
	if err != nil {
		return nil, err
	}

	for k, v := range req.Header {
		if strings.ToLower(k) != "authorization" {
			headReq.Header[k] = v
		}
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * 1000000000,
	}

	resp, err := client.Do(headReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if wwwAuth != "" {
			return parseWWWAuthenticate(wwwAuth)
		}
	}

	challengeURL := &url.URL{
		Scheme: req.URL.Scheme,
		Host:   req.URL.Host,
		Path:   "/",
	}

	headReq2, err := http.NewRequest("HEAD", challengeURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp2, err := client.Do(headReq2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == http.StatusUnauthorized {
		wwwAuth := resp2.Header.Get("WWW-Authenticate")
		if wwwAuth != "" {
			return parseWWWAuthenticate(wwwAuth)
		}
	}

	return nil, fmt.Errorf("could not obtain digest challenge from server")
}
