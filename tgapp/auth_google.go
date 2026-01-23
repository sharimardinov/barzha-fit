package tgapp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const googleStateTTL = 10 * time.Minute

type googleState struct {
	Mode   string `json:"mode"`
	ChatID int64  `json:"chat_id,omitempty"`
	Exp    int64  `json:"exp"`
	Nonce  string `json:"nonce"`
}

type googleTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	IDToken          string `json:"id_token"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type googleTokenInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Aud           string `json:"aud"`
	Iss           string `json:"iss"`
	Exp           string `json:"exp"`
}

func (s *Server) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.googleEnabled() {
		writeGoogleHTML(w, "Google sign in", "Google sign in is not configured.", "")
		return
	}

	state, err := s.buildGoogleState("login", 0)
	if err != nil {
		writeGoogleHTML(w, "Google sign in", "Failed to start Google auth.", "")
		return
	}
	authURL, err := s.buildGoogleAuthURL(r, state)
	if err != nil {
		writeGoogleHTML(w, "Google sign in", "Failed to build Google auth link.", "")
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.googleEnabled() {
		writeGoogleHTML(w, "Google sign in", "Google sign in is not configured.", "")
		return
	}

	if errMsg := strings.TrimSpace(r.URL.Query().Get("error")); errMsg != "" {
		writeGoogleHTML(w, "Google sign in", "Google sign in was cancelled.", "")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	stateRaw := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || stateRaw == "" {
		writeGoogleHTML(w, "Google sign in", "Invalid Google response.", "")
		return
	}

	if _, err := s.parseGoogleState(stateRaw); err != nil {
		writeGoogleHTML(w, "Google sign in", "Invalid auth state.", "")
		return
	}

	tokenResp, err := s.exchangeGoogleCode(r.Context(), code, s.googleRedirectURI(r))
	if err != nil || tokenResp.IDToken == "" {
		writeGoogleHTML(w, "Google sign in", "Failed to exchange Google code.", "")
		return
	}

	info, err := s.verifyGoogleIDToken(r.Context(), tokenResp.IDToken)
	if err != nil {
		writeGoogleHTML(w, "Google sign in", "Invalid Google token.", "")
		return
	}

	if s.googleAuth == nil {
		writeGoogleHTML(w, "Google sign in", "Google sign in is not available.", "")
		return
	}
	chatID, _, err := s.googleAuth.EnsureChatIDBySub(r.Context(), info.Sub, info.Email, info.Name)
	if err != nil || chatID <= 0 {
		if err != nil {
			log.Printf("google auth create failed: sub=%s err=%v", info.Sub, err)
		}
		writeGoogleHTML(w, "Google sign in", "Failed to create account.", "")
		return
	}
	displayName := strings.TrimSpace(info.Email)
	if displayName == "" {
		displayName = strings.TrimSpace(info.Name)
	}
	session := s.sessions.Create(chatID, displayName)
	writeGoogleLoginSuccess(w, session)
}

func (s *Server) googleEnabled() bool {
	return s.googleClientID != "" && s.googleClientSecret != "" && s.googleAuth != nil
}

func (s *Server) googleRedirectURI(r *http.Request) string {
	return baseURLFromRequest(r) + "/auth/google/callback"
}

func baseURLFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host := r.Host
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		host = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return scheme + "://" + host
}

func (s *Server) buildGoogleAuthURL(r *http.Request, state string) (string, error) {
	if !s.googleEnabled() {
		return "", errors.New("google_auth_not_configured")
	}
	u, err := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("client_id", s.googleClientID)
	q.Set("redirect_uri", s.googleRedirectURI(r))
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("prompt", "select_account")
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *Server) buildGoogleState(mode string, chatID int64) (string, error) {
	state := googleState{
		Mode:   mode,
		ChatID: chatID,
		Exp:    time.Now().Add(googleStateTTL).Unix(),
		Nonce:  randomToken(12),
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	sig := signGoogleState(raw, s.authBotToken)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + sig, nil
}

func (s *Server) parseGoogleState(raw string) (googleState, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return googleState{}, errors.New("invalid_state")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return googleState{}, err
	}
	expected := signGoogleState(payload, s.authBotToken)
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return googleState{}, errors.New("invalid_state")
	}
	var state googleState
	if err := json.Unmarshal(payload, &state); err != nil {
		return googleState{}, err
	}
	if state.Exp <= 0 || time.Now().Unix() > state.Exp {
		return googleState{}, errors.New("state_expired")
	}
	if state.Mode == "" {
		return googleState{}, errors.New("invalid_state")
	}
	return state, nil
}

func signGoogleState(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func randomToken(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buf)
}

func (s *Server) exchangeGoogleCode(ctx context.Context, code, redirectURI string) (googleTokenResponse, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", s.googleClientID)
	form.Set("client_secret", s.googleClientSecret)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return googleTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return googleTokenResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return googleTokenResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return googleTokenResponse{}, fmt.Errorf("token_exchange_failed")
	}
	var token googleTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return googleTokenResponse{}, err
	}
	if token.Error != "" {
		return googleTokenResponse{}, fmt.Errorf("token_exchange_failed")
	}
	return token, nil
}

func (s *Server) verifyGoogleIDToken(ctx context.Context, idToken string) (googleTokenInfo, error) {
	if idToken == "" {
		return googleTokenInfo{}, errors.New("empty_token")
	}
	infoURL := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return googleTokenInfo{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return googleTokenInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return googleTokenInfo{}, errors.New("invalid_token")
	}
	var info googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return googleTokenInfo{}, err
	}
	if info.Sub == "" {
		return googleTokenInfo{}, errors.New("invalid_token")
	}
	if info.Aud != s.googleClientID {
		return googleTokenInfo{}, errors.New("invalid_token")
	}
	if info.Iss != "" && info.Iss != "https://accounts.google.com" && info.Iss != "accounts.google.com" {
		return googleTokenInfo{}, errors.New("invalid_token")
	}
	if info.EmailVerified != "" && info.EmailVerified != "true" {
		return googleTokenInfo{}, errors.New("email_not_verified")
	}
	if info.Exp != "" {
		exp, err := strconv.ParseInt(info.Exp, 10, 64)
		if err != nil || time.Now().Unix() > exp {
			return googleTokenInfo{}, errors.New("token_expired")
		}
	}
	return info, nil
}

func writeGoogleHTML(w http.ResponseWriter, title, message, redirectURL string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	escapedTitle := htmlEscape(title)
	escapedMessage := htmlEscape(message)
	action := ""
	script := ""
	if redirectURL != "" {
		escapedURL := htmlEscape(redirectURL)
		action = fmt.Sprintf(`<a class="btn" href="%s">Back to app</a>`, escapedURL)
		script = fmt.Sprintf(`setTimeout(function(){ window.location.href = "%s"; }, 1200);`, escapedURL)
	}
	fmt.Fprintf(w, `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>%s</title>
    <style>
      body { margin: 0; padding: 24px; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #f5f6f8; color: #111; }
      .card { max-width: 460px; margin: 0 auto; background: #fff; border-radius: 16px; padding: 24px; box-shadow: 0 12px 30px rgba(0, 0, 0, 0.08); }
      h1 { margin: 0 0 10px; font-size: 20px; }
      p { margin: 0 0 16px; color: #444; }
      .btn { display: inline-block; padding: 12px 18px; border-radius: 12px; background: #111; color: #fff; text-decoration: none; font-weight: 600; }
    </style>
  </head>
  <body>
    <main class="card">
      <h1>%s</h1>
      <p>%s</p>
      %s
    </main>
    <script>%s</script>
  </body>
</html>`, escapedTitle, escapedTitle, escapedMessage, action, script)
}

func writeGoogleLoginSuccess(w http.ResponseWriter, session authSession) {
	payload := map[string]interface{}{
		"token":     session.Token,
		"userId":    session.UserID,
		"username":  session.Username,
		"expiresAt": session.ExpiresAt.Unix(),
	}
	data, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Google sign in</title>
    <style>
      body { margin: 0; padding: 24px; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #f5f6f8; color: #111; }
      .card { max-width: 460px; margin: 0 auto; background: #fff; border-radius: 16px; padding: 24px; box-shadow: 0 12px 30px rgba(0, 0, 0, 0.08); }
      h1 { margin: 0 0 10px; font-size: 20px; }
      p { margin: 0 0 16px; color: #444; }
    </style>
  </head>
  <body>
    <main class="card">
      <h1>Signed in</h1>
      <p>You can return to the app now.</p>
    </main>
    <script>
      const payload = %s;
      const hasBridge = window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.authComplete;
      if (hasBridge) {
        window.webkit.messageHandlers.authComplete.postMessage(payload);
      } else {
        try { localStorage.setItem("auth_token", payload.token); } catch (_) {}
        setTimeout(function () { window.location.href = "/miniapp/"; }, 400);
      }
    </script>
  </body>
</html>`, data)
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(value)
}
