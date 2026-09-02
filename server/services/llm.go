package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LLMClient struct {
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
	httpClient *http.Client
}

type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewLLMClient(baseURL, apiKey, model string) *LLMClient {
	return &LLMClient{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		Timeout:  30 * time.Second,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *LLMClient) Chat(messages []LLMMessage) (string, error) {
	if c.BaseURL == "" || c.APIKey == "" || c.Model == "" {
		return "", fmt.Errorf("LLM 未配置：请检查 base_url、api_key、model")
	}

	payload := map[string]interface{}{
		"model": c.Model,
		"messages": messages,
		"temperature": 0.7,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API 返回错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(respBody, &llmResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if llmResp.Error != nil {
		return "", fmt.Errorf("LLM 错误: %s", llmResp.Error.Message)
	}

	if len(llmResp.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回空响应")
	}

	return llmResp.Choices[0].Message.Content, nil
}

func GetLLMConfig() (baseURL, apiKey, model string) {
	baseURL = GetSetting("llm.base_url")
	apiKey = GetSetting("llm.api_key")
	model = GetSetting("llm.model")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return
}
