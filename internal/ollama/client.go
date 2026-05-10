package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// freelancerProfile is the structured summary of Angela's CV used as context for the LLM.
const freelancerProfile = `
Angela Scherer is an experienced Freelance Software & DevOps Engineer based in Switzerland.
She operates through her own GmbH (Aiza GmbH) as a sole contractor — she can take on projects
alone or hire 1-2 additional people for larger mandates.

Core skills:
- Backend: C#/.NET, Go, Python, Redis, GraphQL, gRPC, RabbitMQ, Kafka
- Frontend: Angular, Svelte, Vue.js, Razor Pages, TypeScript, JavaScript
- Cloud / DevOps: Kubernetes (CKA + CKAD certified), Docker, GitLab CI/CD, Azure, Rancher
- Databases: MS SQL, PostgreSQL, MySQL, Elasticsearch
- OS: Linux (Fedora, Ubuntu, Debian), Windows

Professional background:
- Freelance: OfficeWorld e-commerce (C#/.NET, Razor Pages, Redis, Azure),
  agricultural information system for public sector (C#/.NET, Svelte, Kubernetes, MS SQL)
- NTS Workspace AG: system modernisation (.NET → Go/Angular), Kubernetes CRDs/Operators,
  messaging integrations (RabbitMQ, Kafka, gRPC, GraphQL)
- Further .NET roles at adesso Schweiz, BFH, Swisscom
- Kubernetes trainer at letsboot.ch (Fundamentals, Operators, CRDs, Go)

Suitable tender characteristics:
- IT services, software development, DevOps, cloud migration, Kubernetes operations,
  .NET or Go development, full-stack web projects, system modernisation
- Manageable team size: solo or up to 3 people — NOT large-scale projects requiring 10+ staff
- Location: Switzerland-wide, preferably remote or German-speaking Switzerland

NOT suitable: construction, cleaning, transport, catering, classic public administration
without an IT component.
`

// AnalysisResult is the structured output from the LLM.
type AnalysisResult struct {
	IsMatch   bool
	Score     int    // 0–100
	Reasoning string // short explanation
}

type ollamaRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Options struct {
		Temperature float64 `json:"temperature"`
	} `json:"options"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

// Client wraps the local Ollama API.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// New creates a new Ollama client.
func New(baseURL, model string) *Client {
	return &Client{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LLM inference can take time
		},
	}
}

// Analyze sends a tender to the local LLM and returns a structured match result.
func (c *Client) Analyze(title, description, subType, procOffice, location, pubDate string) (*AnalysisResult, error) {
	prompt := fmt.Sprintf(`You are an assistant that evaluates Swiss public tenders for a freelancer.

FREELANCER PROFILE:
%s

TENDER:
Title: %s
Contracting authority: %s
Location: %s
Publication date: %s
Project sub-type: %s
Description: %s

TASK:
Decide whether this tender is a good match for the freelancer profile above.
Reply ONLY with the following JSON — no extra text, no markdown:

{
  "is_match": true or false,
  "score": integer 0 to 100 (100 = perfect match),
  "reasoning": "Two or three sentences in English explaining your decision."
}

Criteria for a match (score >= 50):
- IT services, software development, DevOps, cloud, Kubernetes, .NET, Go, Angular, Svelte
- Realistic for a team of 1–3 people
- No match for: construction, cleaning, transport, catering, pure administration without IT
`,
		freelancerProfile,
		title, procOffice, location, pubDate, subType, description,
	)

	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}
	reqBody.Options.Temperature = 0.1 // low temperature for consistent structured output

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/generate",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(raw))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	return parseAnalysis(ollamaResp.Response), nil
}

// parseAnalysis extracts the JSON block from the LLM response.
func parseAnalysis(raw string) *AnalysisResult {
	start, end := -1, -1
	for i, ch := range raw {
		if ch == '{' && start == -1 {
			start = i
		}
		if ch == '}' {
			end = i + 1
		}
	}

	result := &AnalysisResult{}
	if start == -1 || end <= start {
		result.Reasoning = "Could not parse LLM response: " + raw
		return result
	}

	var parsed struct {
		IsMatch   bool   `json:"is_match"`
		Score     int    `json:"score"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(raw[start:end]), &parsed); err != nil {
		result.Reasoning = "JSON parse error: " + err.Error()
		return result
	}

	result.IsMatch = parsed.IsMatch
	result.Score = parsed.Score
	result.Reasoning = parsed.Reasoning
	return result
}
