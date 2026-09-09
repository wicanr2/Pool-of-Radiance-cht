package gamepack

import (
	"path/filepath"
	"testing"
)

func TestContinuePromptsComeFromOverlay3(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	prompts, err := ReadDOSContinuePrompts(zipPath)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	want := map[ContinuePromptKey]string{
		ContinuePromptButton:   "PRESS <RETURN> OR BUTTON TO CONTINUE",
		ContinuePromptKeyboard: "PRESS <ENTER>/<RETURN> TO CONTINUE",
	}
	if len(prompts) != len(want) {
		t.Fatalf("prompts=%+v", prompts)
	}
	for _, prompt := range prompts {
		if prompt.Text != want[prompt.Key] {
			t.Fatalf("%s at %#x is %q, want %q", prompt.Key, prompt.Offset, prompt.Text, want[prompt.Key])
		}
	}
	keyboard, err := DOSContinuePrompt(zipPath, ContinuePromptKeyboard)
	if err != nil || keyboard != want[ContinuePromptKeyboard] {
		t.Fatalf("keyboard prompt %q err=%v", keyboard, err)
	}
}
