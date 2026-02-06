package tgapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	starsCurrency        = "XTR"
	defaultDonationStars = 10
	maxDonationStars     = 10000
)

type starsInvoicePayload struct {
	Amount int `json:"amount"`
}

type tgLabeledPrice struct {
	Label  string `json:"label"`
	Amount int    `json:"amount"`
}

type tgInvoiceLinkRequest struct {
	Title         string           `json:"title"`
	Description   string           `json:"description"`
	Payload       string           `json:"payload"`
	ProviderToken string           `json:"provider_token"`
	Currency      string           `json:"currency"`
	Prices        []tgLabeledPrice `json:"prices"`
}

type tgInvoiceLinkResponse struct {
	OK          bool   `json:"ok"`
	Result      string `json:"result"`
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
}

func (s *Server) handleStarsInvoice(w http.ResponseWriter, r *http.Request, auth authContext) {
	if s.botToken == "" {
		writeJSON(w, http.StatusServiceUnavailable, apiResponse{OK: false, Error: "stars_unavailable"})
		return
	}

	var payload starsInvoicePayload
	if err := decodeJSON(r, &payload); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, apiResponse{OK: false, Error: "bad_request"})
		return
	}

	amount := payload.Amount
	if amount <= 0 {
		amount = defaultDonationStars
	}
	if amount > maxDonationStars {
		amount = maxDonationStars
	}

	url, err := s.createStarsInvoiceLink(r.Context(), auth.User.ID, amount)
	if err != nil {
		log.Printf("stars invoice create failed: user_id=%d err=%v", auth.User.ID, err)
		writeJSON(w, http.StatusInternalServerError, apiResponse{OK: false, Error: "stars_invoice_failed"})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Data: map[string]string{"url": url}})
}

func (s *Server) createStarsInvoiceLink(ctx context.Context, userID int64, amount int) (string, error) {
	if s.botToken == "" {
		return "", errors.New("bot_token_missing")
	}

	reqPayload := tgInvoiceLinkRequest{
		Title:         "Поддержка проекта",
		Description:   "Спасибо за поддержку!",
		Payload:       fmt.Sprintf("donate:%d:%d", userID, time.Now().Unix()),
		ProviderToken: "",
		Currency:      starsCurrency,
		Prices: []tgLabeledPrice{
			{Label: "Поддержка", Amount: amount},
		},
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/createInvoiceLink", s.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("telegram http %d: %s", resp.StatusCode, string(respBody))
	}

	var out tgInvoiceLinkResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", err
	}
	if !out.OK || out.Result == "" {
		if out.Description != "" {
			return "", errors.New(out.Description)
		}
		return "", errors.New("telegram invoice failed")
	}
	return out.Result, nil
}
