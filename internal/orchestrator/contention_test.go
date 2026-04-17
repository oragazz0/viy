package orchestrator

import (
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/oragazz0/viy/internal/k8s"
	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

func TestDetectOverlap_NoOverlap(t *testing.T) {
	resolutions := map[string]*k8s.ResolvedTarget{
		"disintegration": resolvedFor("api", "default",
			podWithUID("pod-1", "default", "uid-1"),
			podWithUID("pod-2", "default", "uid-2"),
		),
		"charm": resolvedFor("worker", "default",
			podWithUID("pod-3", "default", "uid-3"),
		),
	}

	overlaps := detectOverlap(resolutions)
	if len(overlaps) != 0 {
		t.Fatalf("expected no overlap, got %+v", overlaps)
	}
}

func TestDetectOverlap_PairOverlap(t *testing.T) {
	shared := podWithUID("shared", "default", "uid-shared")

	resolutions := map[string]*k8s.ResolvedTarget{
		"disintegration": resolvedFor("api", "default",
			shared,
			podWithUID("pod-disintegration-only", "default", "uid-dis"),
		),
		"charm": resolvedFor("api", "default",
			shared,
			podWithUID("pod-charm-only", "default", "uid-charm"),
		),
	}

	overlaps := detectOverlap(resolutions)
	if len(overlaps) != 1 {
		t.Fatalf("expected 1 overlap, got %d: %+v", len(overlaps), overlaps)
	}

	got := overlaps[0]
	if got.PodUID != types.UID("uid-shared") {
		t.Errorf("PodUID = %q, want uid-shared", got.PodUID)
	}

	if got.PodName != "shared" {
		t.Errorf("PodName = %q, want shared", got.PodName)
	}

	if len(got.Eyes) != 2 {
		t.Fatalf("Eyes = %v, want 2 entries", got.Eyes)
	}

	// detectOverlap sorts eye names so comparisons are stable.
	if got.Eyes[0] != "charm" || got.Eyes[1] != "disintegration" {
		t.Errorf("Eyes = %v, want [charm, disintegration] (sorted)", got.Eyes)
	}
}

func TestDetectOverlap_TripleOverlap(t *testing.T) {
	shared := podWithUID("shared", "default", "uid-shared")

	resolutions := map[string]*k8s.ResolvedTarget{
		"disintegration": resolvedFor("api", "default", shared),
		"charm":          resolvedFor("api", "default", shared),
		"death":          resolvedFor("api", "default", shared),
	}

	overlaps := detectOverlap(resolutions)
	if len(overlaps) != 1 {
		t.Fatalf("expected 1 overlap entry with 3 eyes, got %+v", overlaps)
	}

	if len(overlaps[0].Eyes) != 3 {
		t.Errorf("Eyes = %v, want 3 entries", overlaps[0].Eyes)
	}
}

func TestDetectOverlap_Empty(t *testing.T) {
	overlaps := detectOverlap(map[string]*k8s.ResolvedTarget{})
	if len(overlaps) != 0 {
		t.Errorf("expected no overlaps for empty input, got %+v", overlaps)
	}
}

func TestDetectOverlap_SameNameDifferentUID(t *testing.T) {
	// Pod recreation: same name, different UID. Must NOT count as overlap.
	resolutions := map[string]*k8s.ResolvedTarget{
		"disintegration": resolvedFor("api", "default",
			podWithUID("api-abc", "default", "uid-1"),
		),
		"charm": resolvedFor("api", "default",
			podWithUID("api-abc", "default", "uid-2"),
		),
	}

	overlaps := detectOverlap(resolutions)
	if len(overlaps) != 0 {
		t.Fatalf("same-name/different-UID should not overlap, got %+v", overlaps)
	}
}

func TestNewContentionError_WrapsInvalidConfiguration(t *testing.T) {
	overlaps := []OverlapReport{{
		PodUID:    "uid-1",
		PodName:   "pod-1",
		Namespace: "default",
		Eyes:      []string{"charm", "disintegration"},
	}}

	err := newContentionError(overlaps)
	if err == nil {
		t.Fatal("newContentionError should return non-nil")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("error should wrap ErrInvalidConfiguration, got %v", err)
	}
}
