package orchestrator

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	viyerrors "github.com/oragazz0/viy/pkg/errors"

	"github.com/oragazz0/viy/internal/k8s"
)

// OverlapReport describes a single pod targeted by two or more eyes.
type OverlapReport struct {
	PodUID    types.UID
	PodName   string
	Namespace string
	Eyes      []string
}

// detectOverlap finds pods targeted by more than one eye. Keyed on pod
// UID so pod recreation (e.g. via Disintegration) doesn't poison the
// result with stale matches.
func detectOverlap(resolutions map[string]*k8s.ResolvedTarget) []OverlapReport {
	type podKey struct {
		uid       types.UID
		name      string
		namespace string
	}

	eyesByPod := make(map[types.UID][]string)
	podInfo := make(map[types.UID]podKey)

	for eyeName, resolved := range resolutions {
		for _, pod := range resolved.Pods {
			eyesByPod[pod.UID] = append(eyesByPod[pod.UID], eyeName)
			podInfo[pod.UID] = podKey{
				uid:       pod.UID,
				name:      pod.Name,
				namespace: pod.Namespace,
			}
		}
	}

	reports := make([]OverlapReport, 0)
	for uid, eyeNames := range eyesByPod {
		if len(eyeNames) < 2 {
			continue
		}

		info := podInfo[uid]
		sort.Strings(eyeNames)

		reports = append(reports, OverlapReport{
			PodUID:    uid,
			PodName:   info.name,
			Namespace: info.namespace,
			Eyes:      eyeNames,
		})
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].PodName < reports[j].PodName
	})

	return reports
}

// newContentionError produces a single error describing every overlap.
// Wraps viyerrors.ErrInvalidConfiguration so callers can match.
func newContentionError(overlaps []OverlapReport) error {
	parts := make([]string, 0, len(overlaps))
	for _, overlap := range overlaps {
		parts = append(parts, fmt.Sprintf(
			"pod %s/%s targeted by %s",
			overlap.Namespace, overlap.PodName, strings.Join(overlap.Eyes, ","),
		))
	}

	return fmt.Errorf("%w: %s",
		viyerrors.ErrInvalidConfiguration, strings.Join(parts, "; "))
}
