package tgapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
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
	mux.HandleFunc("/auth/telegram/start", s.handleTelegramStart)
	mux.HandleFunc("/auth/telegram/callback", s.handleTelegramCallback)
	mux.HandleFunc("/auth/verify", s.handleAuthVerify)
	mux.HandleFunc("/auth/google/start", s.handleGoogleStart)
	mux.HandleFunc("/auth/google/callback", s.handleGoogleCallback)
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

func (s *Server) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "missing_token"})
		return
	}

	session, ok := s.sessions.Get(token)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiResponse{OK: false, Error: "invalid_token"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]interface{}{
		"user_id":  session.UserID,
		"username": session.Username,
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

func (s *Server) handleTelegramStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	botID := s.telegramBotID()
	if botID == "" {
		writeGoogleHTML(w, "Telegram sign in", "Telegram sign in is not configured.", "")
		return
	}
	base := baseURLFromRequest(r)
	returnTo := strings.TrimSpace(r.URL.Query().Get("return_to"))
	if returnTo == "" {
		returnTo = "/miniapp/"
	}
	callback := base + "/auth/telegram/callback?return_to=" + url.QueryEscape(returnTo)
	oauth := fmt.Sprintf("https://oauth.telegram.org/auth?bot_id=%s&origin=%s&request_access=write&embed=1&return_to=%s",
		url.QueryEscape(botID),
		url.QueryEscape(base),
		url.QueryEscape(callback),
	)
	http.Redirect(w, r, oauth, http.StatusFound)
}

func (s *Server) handleTelegramCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	returnTo := strings.TrimSpace(r.URL.Query().Get("return_to"))
	if returnTo == "" {
		returnTo = "/miniapp/"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Telegram sign in</title>
    <style>
      body { margin: 0; padding: 24px; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #f5f6f8; color: #111; }
      .card { max-width: 460px; margin: 0 auto; background: #fff; border-radius: 16px; padding: 24px; box-shadow: 0 12px 30px rgba(0, 0, 0, 0.08); }
      h1 { margin: 0 0 10px; font-size: 20px; }
      p { margin: 0; color: #444; }
    </style>
  </head>
  <body>
    <main class="card">
      <h1>Signing in…</h1>
      <p id="status">Preparing Telegram login</p>
    </main>
    <script>
      const statusEl = document.getElementById("status");
      const returnTo = %q;
      function setStatus(msg) { statusEl.textContent = msg || ""; }
      function decodeBase64Url(input) {
        let str = String(input || "").replace(/-/g, "+").replace(/_/g, "/");
        while (str.length %% 4) str += "=";
        return atob(str);
      }
      function appendToken(url, token) {
        try {
          const u = new URL(url, window.location.origin);
          u.searchParams.set("token", token);
          return u.toString();
        } catch (_) {
          const sep = url.includes("?") ? "&" : "?";
          return url + sep + "token=" + encodeURIComponent(token);
        }
      }
      function finish(payload) {
        if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.authComplete) {
          window.webkit.messageHandlers.authComplete.postMessage(payload);
          setStatus("Success. Return to the app.");
          return;
        }
        try { localStorage.setItem("auth_token", payload.token); } catch (_) {}
        window.location.href = appendToken(returnTo, payload.token);
      }
      const hash = window.location.hash || "";
      if (!hash.startsWith("#tgAuthResult=")) {
        setStatus("Missing Telegram data");
      } else {
        try {
          const encoded = hash.slice("#tgAuthResult=".length);
          const jsonText = decodeBase64Url(decodeURIComponent(encoded));
          const user = JSON.parse(jsonText);
          setStatus("Authorizing…");
          fetch("/auth/telegram", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(user),
          })
            .then((res) => res.json())
            .then((resp) => {
              if (!resp.ok) throw new Error(resp.error || "auth_failed");
              const data = resp.data || {};
              finish({
                token: data.token,
                userId: data.user_id,
                username: data.username,
                expiresAt: data.expires_at,
              });
            })
            .catch((err) => {
              console.error(err);
              setStatus("Telegram auth failed");
            });
        } catch (err) {
          console.error(err);
          setStatus("Telegram payload error");
        }
      }
    </script>
  </body>
</html>`, returnTo)
}

func (s *Server) telegramBotID() string {
	token := strings.TrimSpace(s.authBotToken)
	if token == "" {
		token = strings.TrimSpace(s.botToken)
	}
	if token == "" {
		return ""
	}
	parts := strings.SplitN(token, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
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
