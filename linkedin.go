package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
)

const defaultLinkedInVersion = "202609"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func linkedinPersonURN(token string) (string, error) {
	if urn := os.Getenv("LINKEDIN_PERSON_URN"); urn != "" {
		return urn, nil
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo %d: %s (o token precisa do escopo 'openid profile'; ou defina LINKEDIN_PERSON_URN=urn:li:person:SEU_ID no .env)", resp.StatusCode, body)
	}

	var info struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(body, &info); err != nil || info.Sub == "" {
		return "", fmt.Errorf("userinfo sem 'sub': %s", body)
	}
	return "urn:li:person:" + info.Sub, nil
}

func postToLinkedIn(token, text string) (string, error) {
	author, err := linkedinPersonURN(token)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(map[string]any{
		"author":     author,
		"commentary": escapeCommentary(text),
		"visibility": "PUBLIC",
		"distribution": map[string]any{
			"feedDistribution":               "MAIN_FEED",
			"targetEntities":                 []any{},
			"thirdPartyDistributionChannels": []any{},
		},
		"lifecycleState":            "PUBLISHED",
		"isReshareDisabledByAuthor": false,
	})
	if err != nil {
		return "", err
	}

	version := os.Getenv("LINKEDIN_VERSION")
	if version == "" {
		version = defaultLinkedInVersion
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.linkedin.com/rest/posts", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LinkedIn-Version", version)
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("posts %d: %s", resp.StatusCode, body)
	}
	return "https://www.linkedin.com/feed/update/" + resp.Header.Get("x-restli-id") + "/", nil
}

// commentary uses LinkedIn's "little text" format, where these characters are reserved.
const reservedChars = `|{}@[]()<>#\*_~`

func escapeCommentary(text string) string {
	var b strings.Builder
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '#' {
			j := i + 1
			for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j])) {
				j++
			}
			if j > i+1 {
				fmt.Fprintf(&b, `{hashtag|\#|%s}`, string(runes[i+1:j]))
				i = j - 1
				continue
			}
		}
		if strings.ContainsRune(reservedChars, r) {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
