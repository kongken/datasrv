package gen

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

const defaultPRReviewSystemPrompt = `You are a senior code reviewer specializing in pull request analysis.

Your task is to read the pull request title, description, and diff, then produce a structured review.

Follow these rules:
1. Summarize the overall purpose and scope of the changes concisely.
2. Identify risk areas: code that might introduce bugs, security issues, performance regressions, or maintenance burden.
3. Provide actionable suggestions for improvement.
4. Do not invent issues that are not present in the diff.
5. Be concise and focus on what matters most.

Output your review in exactly this format:

SUMMARY:
<2-4 sentences summarizing what this PR does>

RISK AREAS:
<bullet list of risk areas, or "None identified" if the changes are straightforward>

SUGGESTIONS:
<bullet list of actionable suggestions, or "No suggestions" if the PR looks good>`

// PRReviewResult contains the structured output of an AI PR review.
type PRReviewResult struct {
	ReviewSummary string
	RiskAreas     string
	Suggestions   string
}

// PRReviewer wraps a Genkit registry plus the configured model for PR reviews.
type PRReviewer struct {
	g            *genkit.Genkit
	modelName    string
	systemPrompt string
}

// NewPRReviewer creates a reusable Genkit-based PR reviewer.
func NewPRReviewer(ctx context.Context, cfg Config) (_ *PRReviewer, err error) {
	defer func() {
		if r := recover(); r != nil {
			if panicErr, ok := r.(error); ok {
				err = panicErr
				return
			}
			err = fmt.Errorf("init genkit pr reviewer: %v", r)
		}
	}()

	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = ProviderOpenAI
	}

	modelName := normalizeModelName(provider, cfg.Model)
	systemPrompt := strings.TrimSpace(cfg.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = defaultPRReviewSystemPrompt
	}

	var g *genkit.Genkit
	switch provider {
	case ProviderOpenAI:
		g, err = initOpenAI(ctx, cfg)
	case ProviderGoogleAI:
		g, err = initGoogleAI(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported genkit provider %q", provider)
	}
	if err != nil {
		return nil, err
	}

	return &PRReviewer{
		g:            g,
		modelName:    modelName,
		systemPrompt: systemPrompt,
	}, nil
}

// ReviewPR generates an AI review from the PR metadata and diff.
func (r *PRReviewer) ReviewPR(ctx context.Context, title, body, diff string) (PRReviewResult, error) {
	if r == nil || r.g == nil {
		return PRReviewResult{}, errors.New("genkit pr reviewer is not initialized")
	}
	if strings.TrimSpace(diff) == "" {
		return PRReviewResult{}, errors.New("diff is required for PR review")
	}

	prompt := buildPRReviewPrompt(title, body, diff)
	text, err := genkit.GenerateText(ctx, r.g,
		ai.WithModelName(r.modelName),
		ai.WithSystem(r.systemPrompt),
		ai.WithPrompt(prompt),
	)
	if err != nil {
		return PRReviewResult{}, fmt.Errorf("generate pr review: %w", err)
	}

	return parsePRReviewOutput(text), nil
}

func buildPRReviewPrompt(title, body, diff string) string {
	var b strings.Builder

	b.WriteString("Please review the following pull request.\n\n")
	b.WriteString("## PR Title\n")
	b.WriteString(fallback(title, "(no title)"))
	b.WriteString("\n\n")
	b.WriteString("## PR Description\n")
	b.WriteString(fallback(strings.TrimSpace(body), "(no description)"))
	b.WriteString("\n\n")
	b.WriteString("## Diff\n```\n")
	b.WriteString(diff)
	b.WriteString("\n```\n")

	return b.String()
}

func parsePRReviewOutput(text string) PRReviewResult {
	result := PRReviewResult{}
	text = strings.TrimSpace(text)

	summaryIdx := strings.Index(text, "SUMMARY:")
	riskIdx := strings.Index(text, "RISK AREAS:")
	suggestionsIdx := strings.Index(text, "SUGGESTIONS:")

	if summaryIdx >= 0 && riskIdx > summaryIdx {
		result.ReviewSummary = strings.TrimSpace(text[summaryIdx+len("SUMMARY:") : riskIdx])
	}
	if riskIdx >= 0 && suggestionsIdx > riskIdx {
		result.RiskAreas = strings.TrimSpace(text[riskIdx+len("RISK AREAS:") : suggestionsIdx])
	}
	if suggestionsIdx >= 0 {
		result.Suggestions = strings.TrimSpace(text[suggestionsIdx+len("SUGGESTIONS:"):])
	}

	// Fallback: if parsing failed, put everything in summary
	if result.ReviewSummary == "" && result.RiskAreas == "" && result.Suggestions == "" {
		result.ReviewSummary = text
	}

	return result
}
