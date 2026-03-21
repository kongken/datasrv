package gen

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	oai "github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/openai/openai-go/option"
)

const (
	ProviderGoogleAI = "googleai"
	ProviderOpenAI   = "openai"

	defaultGoogleModel = "googleai/gemini-2.5-flash"
	defaultOpenAIModel = "openai/gpt-4.1-mini"
)

const defaultSystemPrompt = `You are an engineering assistant specialized in summarizing GitHub issues.

Your task is to read the issue body together with all replies and produce a concise, accurate summary that is useful for engineering collaboration.

Follow these rules:
1. Start by summarizing the core problem, request, or goal of the issue.
2. Extract important conclusions, investigation progress, decisions, and suggestions from the replies.
3. Clearly call out disagreements, blockers, or open questions when they exist.
4. Do not invent facts that are not present in the issue or replies.
5. Keep the output brief, ideally 3 to 6 sentences.`

// Config controls how the Genkit summarizer connects to the model provider.
type Config struct {
	Provider      string
	Model         string
	SystemPrompt  string
	OpenAIAPIKey  string
	OpenAIBaseURL string
	GoogleAPIKey  string
}

// Issue contains the issue body used as summary input.
type Issue struct {
	Repo      string
	Number    int32
	Title     string
	State     string
	Author    string
	Body      string
	Labels    []string
	HTMLURL   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IssueReply contains a single issue reply or comment.
type IssueReply struct {
	ID        int64
	Author    string
	Body      string
	HTMLURL   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Summarizer wraps a Genkit registry plus the configured model.
type Summarizer struct {
	g            *genkit.Genkit
	modelName    string
	systemPrompt string
}

// NewSummarizer creates a reusable Genkit-based issue summarizer.
func NewSummarizer(ctx context.Context, cfg Config) (_ *Summarizer, err error) {
	defer func() {
		if r := recover(); r != nil {
			if panicErr, ok := r.(error); ok {
				err = panicErr
				return
			}
			err = fmt.Errorf("init genkit summarizer: %v", r)
		}
	}()

	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = ProviderOpenAI
	}

	modelName := normalizeModelName(provider, cfg.Model)
	systemPrompt := strings.TrimSpace(cfg.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}

	var g *genkit.Genkit
	switch provider {
	case ProviderOpenAI:
		plugin := &oai.OpenAI{
			APIKey: strings.TrimSpace(cfg.OpenAIAPIKey),
		}
		if baseURL := strings.TrimSpace(cfg.OpenAIBaseURL); baseURL != "" {
			plugin.Opts = append(plugin.Opts, option.WithBaseURL(baseURL))
		}
		g = genkit.Init(ctx, genkit.WithPlugins(plugin))
	case ProviderGoogleAI:
		g = genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{
			APIKey: strings.TrimSpace(cfg.GoogleAPIKey),
		}))
	default:
		return nil, fmt.Errorf("unsupported genkit provider %q", provider)
	}

	return &Summarizer{
		g:            g,
		modelName:    modelName,
		systemPrompt: systemPrompt,
	}, nil
}

// SummarizeIssue builds a summary from the issue body plus all replies.
func (s *Summarizer) SummarizeIssue(ctx context.Context, issue Issue, replies []IssueReply) (string, error) {
	if s == nil || s.g == nil {
		return "", errors.New("genkit summarizer is not initialized")
	}
	if strings.TrimSpace(issue.Title) == "" && strings.TrimSpace(issue.Body) == "" {
		return "", errors.New("issue title or body is required")
	}

	summary, err := genkit.GenerateText(ctx, s.g,
		ai.WithModelName(s.modelName),
		ai.WithSystem(s.systemPrompt),
		ai.WithPrompt(buildSummaryPrompt(issue, replies)),
	)
	if err != nil {
		return "", fmt.Errorf("generate issue summary: %w", err)
	}

	return strings.TrimSpace(summary), nil
}

func normalizeModelName(provider, model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		switch provider {
		case ProviderGoogleAI:
			return defaultGoogleModel
		default:
			return defaultOpenAIModel
		}
	}
	if strings.Contains(model, "/") {
		return model
	}
	return provider + "/" + model
}

func buildSummaryPrompt(issue Issue, replies []IssueReply) string {
	var b strings.Builder

	b.WriteString("Please generate a summary based on the following GitHub issue and replies.\n\n")
	b.WriteString("## Issue\n")
	b.WriteString("Repo: ")
	b.WriteString(fallback(issue.Repo, "-"))
	b.WriteString("\n")
	b.WriteString("Number: ")
	b.WriteString(fmt.Sprintf("%d", issue.Number))
	b.WriteString("\n")
	b.WriteString("Title: ")
	b.WriteString(fallback(issue.Title, "-"))
	b.WriteString("\n")
	b.WriteString("State: ")
	b.WriteString(fallback(issue.State, "-"))
	b.WriteString("\n")
	b.WriteString("Author: ")
	b.WriteString(fallback(issue.Author, "-"))
	b.WriteString("\n")
	b.WriteString("Labels: ")
	if len(issue.Labels) == 0 {
		b.WriteString("-")
	} else {
		b.WriteString(strings.Join(issue.Labels, ", "))
	}
	b.WriteString("\n")
	b.WriteString("CreatedAt: ")
	b.WriteString(formatTime(issue.CreatedAt))
	b.WriteString("\n")
	b.WriteString("UpdatedAt: ")
	b.WriteString(formatTime(issue.UpdatedAt))
	b.WriteString("\n")
	b.WriteString("URL: ")
	b.WriteString(fallback(issue.HTMLURL, "-"))
	b.WriteString("\n")
	b.WriteString("Body:\n")
	b.WriteString(fallback(strings.TrimSpace(issue.Body), "(empty)"))
	b.WriteString("\n\n")

	b.WriteString("## Replies\n")
	if len(replies) == 0 {
		b.WriteString("(no replies)\n")
	} else {
		for i, reply := range replies {
			b.WriteString(fmt.Sprintf("### Reply %d\n", i+1))
			b.WriteString("ID: ")
			b.WriteString(fmt.Sprintf("%d", reply.ID))
			b.WriteString("\n")
			b.WriteString("Author: ")
			b.WriteString(fallback(reply.Author, "-"))
			b.WriteString("\n")
			b.WriteString("CreatedAt: ")
			b.WriteString(formatTime(reply.CreatedAt))
			b.WriteString("\n")
			b.WriteString("UpdatedAt: ")
			b.WriteString(formatTime(reply.UpdatedAt))
			b.WriteString("\n")
			b.WriteString("URL: ")
			b.WriteString(fallback(reply.HTMLURL, "-"))
			b.WriteString("\n")
			b.WriteString("Body:\n")
			b.WriteString(fallback(strings.TrimSpace(reply.Body), "(empty)"))
			b.WriteString("\n\n")
		}
	}

	b.WriteString("Return the summary in English. Do not use markdown bullet lists.")
	return b.String()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

func fallback(v, defaultValue string) string {
	if strings.TrimSpace(v) == "" {
		return defaultValue
	}
	return v
}
