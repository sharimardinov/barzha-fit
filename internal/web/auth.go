package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	errUnauthorized   = errors.New("unauthorized")
	errMissingInit    = errors.New("missing_init_data")
	errBadHash        = errors.New("bad_hash")
	errStaleAuthDate  = errors.New("stale_auth_date")
	errBadInitData    = errors.New("bad_init_data")
	errBadUserPayload = errors.New("bad_user_payload")
)

type webUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type authContext struct {
	User webUser
}

func (s *Server) authenticate(r *http.Request) (authContext, error) {
	initData := strings.TrimSpace(r.Header.Get("X-Tg-Init-Data"))
	if initData == "" {
		log.Printf("miniapp auth: missing init data")
		return authContext{}, errMissingInit
	}

	values, err := validateInitData(initData, s.botToken)
	if err != nil {
		log.Printf("miniapp auth: validate failed: %v", err)
		return authContext{}, err
	}

	userRaw := values.Get("user")
	if userRaw == "" {
		return authContext{}, errBadUserPayload
	}

	var user webUser
	if err := json.Unmarshal([]byte(userRaw), &user); err != nil {
		return authContext{}, errUnauthorized
	}
	if user.ID == 0 {
		return authContext{}, errBadUserPayload
	}

	return authContext{User: user}, nil
}

func validateInitData(initData, botToken string) (url.Values, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, errBadInitData
	}

	hash := values.Get("hash")
	if hash == "" {
		return nil, errBadInitData
	}
	values.Del("hash")

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(values.Get(k))
	}

	secretMac := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretMac.Write([]byte(botToken))
	secretKey := secretMac.Sum(nil)

	mac := hmac.New(sha256.New, secretKey)
	_, _ = mac.Write([]byte(b.String()))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(hash)) {
		return nil, errBadHash
	}

	if authDate := values.Get("auth_date"); authDate != "" {
		ts, err := strconv.ParseInt(authDate, 10, 64)
		if err != nil {
			return nil, errBadInitData
		}
		if time.Since(time.Unix(ts, 0)) > 24*time.Hour {
			return nil, errStaleAuthDate
		}
	}

	return values, nil
}
