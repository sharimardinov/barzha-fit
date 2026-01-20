package tgapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type telegramLogin struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	PhotoURL  string `json:"photo_url,omitempty"`
	AuthDate  int64  `json:"auth_date"`
	Hash      string `json:"hash"`
}

func (s *Server) registerAuth(mux *http.ServeMux) {
	mux.HandleFunc("/auth/telegram", s.handleTelegramAuth)
}

func (s *Server) handleTelegramAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.authBotToken == "" {
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "auth_not_configured"})
		return
	}

	var payload telegramLogin
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}
	if payload.ID == 0 || payload.AuthDate == 0 || payload.Hash == "" {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}
	if !verifyTelegramLogin(payload, s.authBotToken) {
		writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "unauthorized"})
		return
	}
	if time.Since(time.Unix(payload.AuthDate, 0)) > 24*time.Hour {
		writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "stale_auth_date"})
		return
	}

	session := s.sessions.Create(payload.ID, payload.Username)
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"token":      session.Token,
		"user_id":    session.UserID,
		"username":   session.Username,
		"expires_at": session.ExpiresAt.Unix(),
	}})
}

func verifyTelegramLogin(data telegramLogin, botToken string) bool {
	checkString := createLoginCheckString(data)

	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	_, _ = mac.Write([]byte(checkString))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(data.Hash))
}

func createLoginCheckString(data telegramLogin) string {
	params := map[string]string{
		"id":         strconv.FormatInt(data.ID, 10),
		"first_name": data.FirstName,
		"auth_date":  strconv.FormatInt(data.AuthDate, 10),
	}
	if data.LastName != "" {
		params["last_name"] = data.LastName
	}
	if data.Username != "" {
		params["username"] = data.Username
	}
	if data.PhotoURL != "" {
		params["photo_url"] = data.PhotoURL
	}

	keys := make([]string, 0, len(params))
	for k := range params {
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
		b.WriteString(params[k])
	}
	return b.String()
}
