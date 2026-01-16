package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type AIClient struct {
	apiKey  string
	model   string
	http    *http.Client
	logPath string
}

func NewAIClient() (*AIClient, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY is required")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	timeout := 60 * time.Second
	if raw := os.Getenv("AI_TIMEOUT_SEC"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			timeout = time.Duration(v) * time.Second
		}
	}
	logPath := os.Getenv("AI_LOG_PATH")
	if logPath == "" {
		logPath = "logs/ai_training.log"
	}
	return &AIClient{
		apiKey:  key,
		model:   model,
		logPath: logPath,
		http: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

type respReq struct {
	Model           string       `json:"model"`
	Input           []respMsg    `json:"input"`
	Instructions    string       `json:"instructions,omitempty"`
	Temperature     float64      `json:"temperature,omitempty"`
	TopP            float64      `json:"top_p,omitempty"`
	MaxOutputTokens int          `json:"max_output_tokens,omitempty"`
	Text            *respTextCfg `json:"text,omitempty"`
}

type respMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type respTextCfg struct {
	Format any `json:"format,omitempty"`
}

type respError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    any    `json:"code,omitempty"`
}

type respOut struct {
	ID     string     `json:"id"`
	Object string     `json:"object"`
	Status string     `json:"status"`
	Error  *respError `json:"error,omitempty"`
	Output []struct {
		Type    string `json:"type"`
		Role    string `json:"role,omitempty"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content,omitempty"`
	} `json:"output"`
}

func (c *AIClient) postResponses(ctx context.Context, req respReq) (respOut, []byte, int, error) {
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return respOut{}, nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return respOut{}, nil, resp.StatusCode, err
	}
	return respOut{}, respBody, resp.StatusCode, nil
}

func (c *AIClient) appendAILog(kind, payload, note string) {
	if c.logPath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(c.logPath), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(c.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format(time.RFC3339)
	if note != "" {
		_, _ = fmt.Fprintf(f, "[%s] %s: %s\n", ts, kind, note)
	} else {
		_, _ = fmt.Fprintf(f, "[%s] %s:\n", ts, kind)
	}
	if payload != "" {
		_, _ = fmt.Fprintln(f, payload)
	}
	_, _ = fmt.Fprintln(f, "-----")
}

func extractOutputText(out respOut) string {
	rawText := ""
	for _, item := range out.Output {
		for _, c := range item.Content {
			if c.Type == "output_text" && c.Text != "" {
				rawText += c.Text
			}
		}
	}
	return strings.TrimSpace(rawText)
}

func cleanJSON(raw string) string {
	raw = sanitizeJSON(raw)
	return strings.TrimSpace(raw)
}
