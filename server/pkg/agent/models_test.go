package agent

import (
	"context"
	"testing"
)

func TestListModelsStaticProviders(t *testing.T) {
	ctx := context.Background()
	for _, provider := range []string{"claude", "codex", "gemini", "cursor", "copilot"} {
		got, err := ListModels(ctx, provider, "")
		if err != nil {
			t.Fatalf("ListModels(%q) error: %v", provider, err)
		}
		if len(got) == 0 {
			t.Errorf("ListModels(%q) returned no models", provider)
		}
		for i, m := range got {
			if m.ID == "" {
				t.Errorf("ListModels(%q)[%d] has empty ID", provider, i)
			}
			if m.Label == "" {
				t.Errorf("ListModels(%q)[%d] has empty Label", provider, i)
			}
		}
	}
}

func TestListModelsHermesReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	got, err := ListModels(ctx, "hermes", "")
	if err != nil {
		t.Fatalf("ListModels(hermes) error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListModels(hermes) expected empty, got %d", len(got))
	}
}

func TestListModelsUnknownProvider(t *testing.T) {
	ctx := context.Background()
	_, err := ListModels(ctx, "nonexistent", "")
	if err == nil {
		t.Fatal("ListModels(unknown) expected error")
	}
}

func TestDefaultModel(t *testing.T) {
	cases := map[string]string{
		"claude":   "claude-sonnet-4-6",
		"codex":    "gpt-5.4",
		"gemini":   "gemini-2.5-pro",
		"cursor":   "composer-1.5",
		"copilot":  "", // GitHub-routed, deliberately no opinion
		"hermes":   "", // out-of-band config
		"opencode": "", // dynamic, no shipped default
		"openclaw": "", // pre-registered agents only
	}
	for provider, want := range cases {
		got := DefaultModel(provider)
		if got != want {
			t.Errorf("DefaultModel(%q) = %q, want %q", provider, got, want)
		}
	}
}

func TestStaticCatalogsHaveAtMostOneDefault(t *testing.T) {
	// More than one Default per catalog would make the daemon's
	// fallback chain ambiguous; ensure we don't accidentally mark
	// two when adding new models.
	for _, provider := range []string{"claude", "codex", "gemini", "cursor", "copilot"} {
		count := 0
		for _, m := range defaultStaticModelsFor(provider) {
			if m.Default {
				count++
			}
		}
		if count > 1 {
			t.Errorf("%s: %d models marked Default, want 0 or 1", provider, count)
		}
	}
}

func TestParseOpenCodeModels(t *testing.T) {
	input := `PROVIDER/MODEL                     CONTEXT  MAX_OUT
openai/gpt-4o                      128000   16384
anthropic/claude-sonnet-4-6        200000   8192
openai/gpt-4o                      128000   16384
nonprefixed-line
`
	models := parseOpenCodeModels(input)
	if len(models) != 2 {
		t.Fatalf("expected 2 models (header skipped, duplicate deduped, non-slash skipped), got %d: %+v", len(models), models)
	}
	if models[0].ID != "openai/gpt-4o" || models[0].Provider != "openai" {
		t.Errorf("unexpected first model: %+v", models[0])
	}
	if models[1].ID != "anthropic/claude-sonnet-4-6" || models[1].Provider != "anthropic" {
		t.Errorf("unexpected second model: %+v", models[1])
	}
}

func TestParsePiModels(t *testing.T) {
	input := `openai:gpt-4o
anthropic:claude-opus-4-7
openai:gpt-4o
bareword
`
	models := parsePiModels(input)
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d: %+v", len(models), models)
	}
	if models[0].ID != "openai/gpt-4o" {
		t.Errorf("expected colon normalized to slash: %+v", models[0])
	}
}

func TestParseOpenclawAgents(t *testing.T) {
	input := `NAME          MODEL
deepseek-v4   deepseek-v4
---
claude-sonnet claude-sonnet-4-6
deepseek-v4   deepseek-v4
`
	models := parseOpenclawAgents(input)
	// header and separator skipped; duplicate deduped.
	if len(models) != 2 {
		t.Fatalf("expected 2 agents, got %d: %+v", len(models), models)
	}
	if models[0].ID != "deepseek-v4" {
		t.Errorf("unexpected first agent: %+v", models[0])
	}
	if models[0].Provider != "openclaw" {
		t.Errorf("expected provider openclaw, got %q", models[0].Provider)
	}
}

func TestCachedDiscovery(t *testing.T) {
	calls := 0
	fn := func() ([]Model, error) {
		calls++
		return []Model{{ID: "x", Label: "x"}}, nil
	}
	// First call populates the cache; reset for isolation.
	modelCacheMu.Lock()
	delete(modelCache, "testkey")
	modelCacheMu.Unlock()

	if _, err := cachedDiscovery("testkey", fn); err != nil {
		t.Fatal(err)
	}
	if _, err := cachedDiscovery("testkey", fn); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("expected 1 underlying call due to cache, got %d", calls)
	}
}
