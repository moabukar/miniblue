package storageauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"unicode/utf8"
)

// VerifyBlobSharedKey checks Authorization against key1 or key2 (full SharedKey, then SharedKeyLite).
func VerifyBlobSharedKey(r *http.Request, accountName string, key1, key2 []byte) bool {
	authz := r.Header.Get("Authorization")
	if authz == "" {
		return false
	}
	if strings.HasPrefix(authz, "SharedKey ") {
		return verifyScheme(r, accountName, authz[len("SharedKey "):], key1, key2, false)
	}
	if strings.HasPrefix(authz, "SharedKeyLite ") {
		return verifyScheme(r, accountName, authz[len("SharedKeyLite "):], key1, key2, true)
	}
	return false
}

func verifyScheme(r *http.Request, accountName string, cred string, key1, key2 []byte, lite bool) bool {
	// cred is "account:signatureBase64"
	i := strings.LastIndex(cred, ":")
	if i <= 0 || i == len(cred)-1 {
		return false
	}
	acct := cred[:i]
	sigB64 := cred[i+1:]
	if acct != accountName {
		return false
	}
	wantSig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return false
	}
	s1, err := buildStringToSign(r, accountName, lite)
	if err != nil {
		return false
	}
	if matchHMAC(wantSig, []byte(s1), key1) || matchHMAC(wantSig, []byte(s1), key2) {
		return true
	}
	return false
}

func matchHMAC(wantSig, msg, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(msg)
	got := mac.Sum(nil)
	return subtle.ConstantTimeCompare(got, wantSig) == 1
}

// SignBlobSharedKey sets Authorization using SharedKey (for tests and tooling).
func SignBlobSharedKey(r *http.Request, accountName string, key []byte, lite bool) error {
	s, err := buildStringToSign(r, accountName, lite)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(s))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	scheme := "SharedKey"
	if lite {
		scheme = "SharedKeyLite"
	}
	r.Header.Set("Authorization", fmt.Sprintf("%s %s:%s", scheme, accountName, sig))
	return nil
}

func buildStringToSign(r *http.Request, accountName string, lite bool) (string, error) {
	if lite {
		return buildStringToSignLite(r, accountName)
	}
	return buildStringToSignFull(r, accountName)
}

func buildStringToSignFull(r *http.Request, accountName string) (string, error) {
	canonicalRes, err := buildCanonicalizedResource(accountName, r.URL)
	if err != nil {
		return "", err
	}
	canonicalHdrs := buildCanonicalizedHeaders(r)

	cl := r.Header.Get("Content-Length")
	if cl == "0" {
		cl = ""
	}

	dateField := r.Header.Get("Date")
	if r.Header.Get("X-Ms-Date") != "" {
		dateField = ""
	}

	var b strings.Builder
	b.WriteString(strings.ToUpper(r.Method))
	b.WriteByte('\n')
	writeField(&b, r.Header.Get("Content-Encoding"))
	writeField(&b, r.Header.Get("Content-Language"))
	writeField(&b, cl)
	writeField(&b, r.Header.Get("Content-Md5"))
	writeField(&b, r.Header.Get("Content-Type"))
	writeField(&b, dateField)
	writeField(&b, r.Header.Get("If-Modified-Since"))
	writeField(&b, r.Header.Get("If-Match"))
	writeField(&b, r.Header.Get("If-None-Match"))
	writeField(&b, r.Header.Get("If-Unmodified-Since"))
	writeField(&b, r.Header.Get("Range"))
	b.WriteString(canonicalHdrs)
	b.WriteString(canonicalRes)
	return b.String(), nil
}

func buildStringToSignLite(r *http.Request, accountName string) (string, error) {
	canonicalRes, err := buildCanonicalizedResource(accountName, r.URL)
	if err != nil {
		return "", err
	}
	canonicalHdrs := buildCanonicalizedHeaders(r)

	dateField := r.Header.Get("Date")
	if r.Header.Get("X-Ms-Date") != "" {
		dateField = ""
	}

	var b strings.Builder
	b.WriteString(strings.ToUpper(r.Method))
	b.WriteByte('\n')
	writeField(&b, r.Header.Get("Content-Md5"))
	writeField(&b, r.Header.Get("Content-Type"))
	writeField(&b, dateField)
	b.WriteString(canonicalHdrs)
	b.WriteString(canonicalRes)
	return b.String(), nil
}

func writeField(b *strings.Builder, v string) {
	b.WriteString(v)
	b.WriteByte('\n')
}

func buildCanonicalizedHeaders(r *http.Request) string {
	type hdr struct{ k, v string }
	var xs []hdr
	for name, vals := range r.Header {
		ln := strings.ToLower(name)
		if !strings.HasPrefix(ln, "x-ms-") {
			continue
		}
		val := strings.Join(vals, ",")
		val = collapseSpace(val)
		xs = append(xs, hdr{k: ln, v: val})
	}
	sort.Slice(xs, func(i, j int) bool {
		if xs[i].k != xs[j].k {
			return xs[i].k < xs[j].k
		}
		return xs[i].v < xs[j].v
	})
	var b strings.Builder
	for _, h := range xs {
		b.WriteString(h.k)
		b.WriteByte(':')
		b.WriteString(h.v)
		b.WriteByte('\n')
	}
	return b.String()
}

func collapseSpace(s string) string {
	var b strings.Builder
	var space bool
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !space && b.Len() > 0 {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		space = false
		if r < utf8.RuneSelf {
			b.WriteByte(byte(r))
		} else {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func buildCanonicalizedResource(accountName string, u *url.URL) (string, error) {
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	var b strings.Builder
	b.WriteByte('/')
	b.WriteString(accountName)
	b.WriteString(path)

	raw := u.RawQuery
	if raw == "" {
		return b.String(), nil
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return "", err
	}
	// group decoded names -> sorted values per name
	params := make(map[string][]string)
	for k, vs := range q {
		dk, err := url.QueryUnescape(k)
		if err != nil {
			dk = k
		}
		dk = strings.ToLower(dk)
		for _, v := range vs {
			dv, err := url.QueryUnescape(v)
			if err != nil {
				dv = v
			}
			params[dk] = append(params[dk], dv)
		}
	}
	names := make([]string, 0, len(params))
	for k := range params {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		vals := append([]string(nil), params[k]...)
		sort.Strings(vals)
		b.WriteByte('\n')
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(strings.Join(vals, ","))
	}
	return b.String(), nil
}
