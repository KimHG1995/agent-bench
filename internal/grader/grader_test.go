package grader

import (
	"math"
	"reflect"
	"testing"

	"github.com/KimHG1995/agent-bench/internal/domain"
)

func TestGradeExactMatch(t *testing.T) {
	expected := domain.Evidence{
		Symbols: []string{"UserService.deleteUser"},
		Paths:   []string{"./src/user/user.service.ts"},
		Relationships: []domain.Relationship{{
			From: "UserController.deleteUser",
			To:   "UserService.deleteUser",
		}},
	}
	actual := domain.Evidence{
		Symbols: []string{"UserService.deleteUser"},
		Paths:   []string{"src/user/user.service.ts"},
		Relationships: []domain.Relationship{{
			From: "UserController.deleteUser",
			To:   "UserService.deleteUser",
		}},
	}

	got := Grade(expected, actual)
	if got.Accuracy != 1 || got.Precision != 1 || got.Recall != 1 {
		t.Fatalf("expected perfect score, got %+v", got)
	}
	if len(got.Missing) != 0 || len(got.Unexpected) != 0 {
		t.Fatalf("expected no evidence gaps, got %+v", got)
	}
}

func TestGradePartialMatch(t *testing.T) {
	expected := domain.Evidence{Symbols: []string{"A", "B"}}
	actual := domain.Evidence{Symbols: []string{"A", "C"}}

	got := Grade(expected, actual)
	if math.Abs(got.Accuracy-0.5) > 1e-9 {
		t.Fatalf("expected F1=0.5, got %f", got.Accuracy)
	}
	if !reflect.DeepEqual(got.Missing, []string{"symbol:B"}) {
		t.Fatalf("unexpected missing: %#v", got.Missing)
	}
	if !reflect.DeepEqual(got.Unexpected, []string{"symbol:C"}) {
		t.Fatalf("unexpected unexpected facts: %#v", got.Unexpected)
	}
}

func TestGradeEmptyEvidence(t *testing.T) {
	got := Grade(domain.Evidence{}, domain.Evidence{})
	if got.Accuracy != 1 {
		t.Fatalf("expected empty evidence to match, got %f", got.Accuracy)
	}
}
