package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GeminiClient はGemini APIとの通信を担当するクライアント
type GeminiClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewGeminiClient は新しいGeminiClientインスタンスを作成
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GeminiRequest はGemini APIへのリクエスト構造体
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

// Content はリクエストの内容
type Content struct {
	Parts []Part `json:"parts"`
}

// Part はテキスト部分
type Part struct {
	Text string `json:"text"`
}

// GeminiResponse はGemini APIからのレスポンス構造体
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate は生成された候補
type Candidate struct {
	Content Content `json:"content"`
}

// StoryContent は物語のタイトルと本文を含む構造体
type StoryContent struct {
	Title string
	Story string
}

// GenerateContent はGemini APIを使ってコンテンツを生成する
func (c *GeminiClient) GenerateContent(ctx context.Context, prompt string) (string, error) {
	req := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("リクエストのシリアライズに失敗: %w", err)
	}

	url := fmt.Sprintf("%s/models/gemini-pro:generateContent?key=%s", c.baseURL, c.apiKey)
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("HTTPリクエストの作成に失敗: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("APIリクエストに失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API呼び出しエラー (status: %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("レスポンスの読み取りに失敗: %w", err)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("レスポンスのパースに失敗: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("有効なレスポンスが生成されませんでした")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// GenerateStoryContent はGemini APIを使ってタイトルと物語を同時生成する
func (c *GeminiClient) GenerateStoryContent(ctx context.Context, prompt string) (*StoryContent, error) {
	content, err := c.GenerateContent(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// レスポンスを解析してタイトルと物語を抽出
	return c.parseStoryContent(content), nil
}

// parseStoryContent は生成されたコンテンツからタイトルと物語を抽出
func (c *GeminiClient) parseStoryContent(content string) *StoryContent {
	lines := strings.Split(content, "\n")
	
	var title, story string
	var storyStarted bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// タイトルの検出パターン
		if strings.HasPrefix(line, "タイトル:") || strings.HasPrefix(line, "【タイトル】") {
			title = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "タイトル:"), "【タイトル】"))
			continue
		}
		
		// 物語の開始検出パターン
		if strings.HasPrefix(line, "物語:") || strings.HasPrefix(line, "【物語】") || strings.HasPrefix(line, "本文:") {
			storyStarted = true
			story = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "物語:"), "【物語】"), "本文:"))
			if story != "" {
				continue
			}
		}
		
		// タイトルがまだ設定されていない場合、最初の行をタイトルとする
		if title == "" && !storyStarted {
			title = line
			continue
		}
		
		// 物語部分の収集
		if storyStarted || title != "" {
			if story != "" {
				story += " " + line
			} else {
				story = line
			}
		}
	}
	
	// フォールバック処理
	if title == "" && story != "" {
		// 物語の最初の30文字をタイトルにする
		if len(story) > 30 {
			title = story[:30] + "..."
		} else {
			title = story
		}
	}
	
	return &StoryContent{
		Title: title,
		Story: story,
	}
}
