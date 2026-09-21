package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunRequestDoesNotExposeExpected(t *testing.T) {
	task := Task{
		ID:       "t",
		Category: "lookup",
		Question: "q",
		Expected: Evidence{Symbols: []string{"SECRET_ANSWER"}},
	}

	b, err := json.Marshal(RunRequest{
		Strategy: "baseline",
		Run:      1,
		Task:     task.ForAgent(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "SECRET_ANSWER") {
		t.Fatalf("expected leaked into agent request: %s", b)
	}
}
