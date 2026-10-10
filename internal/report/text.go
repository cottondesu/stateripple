package report

import (
	"fmt"
	"strconv"
	"strings"
)

func Text(r Report) string {
	r = ordered(r)
	var b strings.Builder
	fmt.Fprintf(&b, "StateRipple v0.1.0\nStatus: %s (%s)\nRuleset: %s\nUpdates: %d\nInput SHA-256: %s\n", escaped(r.Status), escaped(r.StopReason), escaped(r.Ruleset), r.GenerationCount, escaped(r.InputSHA256))
	counts := [4]int{}
	for _, c := range r.Cells {
		if c.Risk >= 0 && c.Risk <= 3 {
			counts[c.Risk]++
		}
	}
	fmt.Fprintf(&b, "Critical: %d  Active: %d  Watch: %d  Dormant: %d\n", counts[3], counts[2], counts[1], counts[0])
	// Git and test sections belong to the software adapter. A generic snapshot
	// simulation shows them only when its graph has kind "test" targets.
	analysis := r.BuildContext != nil || len(r.Changes) > 0
	tests := analysis || len(r.TestTargets) > 0
	if analysis {
		b.WriteString("Changed files:\n")
		for _, c := range r.Changes {
			fmt.Fprintf(&b, "  %s %s (%s)\n", escaped(c.Status), quoted(c.Path), escaped(c.Classification))
		}
		if len(r.Changes) == 0 {
			b.WriteString("  none\n")
		}
	}
	if tests {
		b.WriteString("Recommended test targets:\n")
		for _, target := range r.TestTargets {
			fmt.Fprintf(&b, "  risk %d %s\n", target.Risk, escaped(target.ID))
		}
		if len(r.TestTargets) == 0 {
			b.WriteString("  none; this does not prove safety\n")
		}
	}
	b.WriteString("Cells and propagation reasons:\n")
	for _, c := range r.Cells {
		fmt.Fprintf(&b, "  %s risk=%d", escaped(c.ID), c.Risk)
		if c.FirstActiveGeneration != nil {
			fmt.Fprintf(&b, " first_active=%d", *c.FirstActiveGeneration)
		}
		b.WriteByte('\n')
		for _, cause := range c.Causes {
			if cause.Edge != nil {
				fmt.Fprintf(&b, "    %s via %s (attenuation %d): %s\n", escaped(cause.Source), escaped(cause.Edge.Kind), cause.Edge.Attenuation, escaped(cause.Reason))
			} else {
				fmt.Fprintf(&b, "    %s: %s\n", escaped(cause.Source), escaped(cause.Reason))
			}
		}
	}
	if bc := r.BuildContext; bc != nil {
		fmt.Fprintf(&b, "Build context: %s %s/%s CGO_ENABLED=%s module=%s\nBuild tags: %s\nTool tags: %s\nRelease tags: %s\n", escaped(bc.GoVersion), escaped(bc.GOOS), escaped(bc.GOARCH), escaped(bc.CGOEnabled), escaped(bc.Module), escapedList(bc.BuildTags), escapedList(bc.ToolTags), escapedList(bc.ReleaseTags))
	}
	for _, p := range r.UnknownChanges {
		fmt.Fprintf(&b, "Unknown change: %s\n", quoted(p))
	}
	for _, warning := range r.Warnings {
		fmt.Fprintf(&b, "Warning: %s\n", escaped(warning))
	}
	if tests {
		b.WriteString("Risk is not bug probability. Selected tests have not been executed.\n")
	} else {
		b.WriteString("Risk is a propagated priority, not a probability.\n")
	}
	return b.String()
}

func quoted(value string) string {
	return strconv.QuoteToGraphic(value)
}

func escaped(value string) string {
	value = quoted(value)
	return value[1 : len(value)-1]
}

func escapedList(values []string) string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = escaped(value)
	}
	return strings.Join(result, ",")
}
