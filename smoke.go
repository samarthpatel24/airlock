package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// smoke runs after the model server has started: it waits for /health, asks
// one question with a known answer, and appends the result to the report.
// Works with any OpenAI-compatible server (llama.cpp, vLLM). It proves the
// server answers; it does not prove the server loaded the verified files.
func smoke(args []string) error {
	fs := flag.NewFlagSet("smoke", flag.ExitOnError)
	base := fs.String("url", "http://127.0.0.1:8080", "OpenAI-compatible model server")
	prompt := fs.String("prompt", "What is 2+2? Answer with just the number.", "test prompt")
	expect := fs.String("expect", "4", "answer must equal this (case-insensitive, ignoring spaces and a trailing period); empty accepts any answer")
	wait := fs.Duration("wait", 2*time.Minute, "how long to wait for the server to become healthy")
	out := fs.String("report", "airlock-report.md", "report to append the result to")
	fs.Parse(args)
	if fs.NArg() != 0 {
		return errors.New("usage: airlock smoke [--url URL] [--prompt P] [--expect E] [--report FILE]")
	}
	u, err := url.Parse(*base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("--url %q: need http(s)://host:port", *base)
	}
	root := strings.TrimRight(u.String(), "/")
	shown := *u
	shown.User = nil // never write credentials into the report

	start := time.Now()
	answer, err := ask(root, *prompt, *expect, *wait)
	took := time.Since(start).Round(time.Millisecond)

	result := "PASS"
	if err != nil {
		result = "FAIL"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n## Smoke test\n\n- **Result:** %s\n- **Server:** %s\n- **Prompt:** %s\n", result, code(shown.String()), code(*prompt))
	fmt.Fprintf(&b, "- **Expected:** %s\n- **Answer:** %s\n- **Took:** %s\n- **Tested at:** %s\n",
		code(*expect), code(answer), took, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		fmt.Fprintf(&b, "- ✘ %s\n", code(err.Error()))
	}
	if werr := appendReport(*out, b.String()); werr != nil {
		return werr
	}
	if err != nil {
		return fmt.Errorf("FAIL — smoke test: %w (report: %s)", err, *out)
	}
	fmt.Printf("%s PASS — model answered %q in %s (report: %s)\n", markOK, answer, took, *out)
	return nil
}

// code renders untrusted text as a quoted inline code span, so nothing in it
// (links, HTML, line breaks, escapes) is interpreted by a Markdown viewer.
func code(s string) string {
	return "`" + strings.ReplaceAll(fmt.Sprintf("%q", s), "`", "'") + "`"
}

func ask(root, prompt, expect string, wait time.Duration) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("redirect refused")
	}}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, root+"/health", nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("server not healthy after %s", wait)
		case <-time.After(time.Second):
		}
	}
	body, _ := json.Marshal(map[string]any{
		"messages":    []map[string]string{{"role": "user", "content": prompt}},
		"max_tokens":  32,
		"temperature": 0,
	})
	resp, err := client.Post(root+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("chat completion: HTTP %d", resp.StatusCode) // never the server's status text
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&r); err != nil {
		return "", errors.New("chat completion: invalid JSON")
	}
	if len(r.Choices) == 0 || strings.TrimSpace(r.Choices[0].Message.Content) == "" {
		return "", errors.New("empty answer")
	}
	answer := truncate(strings.TrimSpace(r.Choices[0].Message.Content), 200)
	if expect != "" && !strings.EqualFold(strings.TrimSuffix(answer, "."), expect) {
		return answer, fmt.Errorf("answer is not %q", expect)
	}
	return answer, nil
}

// truncate cuts s to at most n bytes without splitting a UTF-8 character.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

func appendReport(path, text string) error {
	if st, err := os.Lstat(path); err == nil && !st.Mode().IsRegular() {
		return fmt.Errorf("report path %q is not a regular file; refusing to write through it", path)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
