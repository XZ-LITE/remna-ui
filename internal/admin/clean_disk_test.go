package admin

import (
	"fmt"
	"strings"
	"testing"
)

func TestCPUStealParsing(t *testing.T) {
	sampleOutput := "Linux VPS\n---\n250|1024\n---\n12G|40G|30%\n---\nup 3 days\n---\nremnanode\tUp 3 days\tremnanode:latest\n---\nbbr\n---\nyes\n---\n\n---\n0\n---\n1\n---\nyes\n---\nyes\n---\n14.8"
	parts := strings.Split(sampleOutput, "---")

	if len(parts) < 13 {
		t.Fatalf("Expected at least 13 parts, got %d", len(parts))
	}

	stealStr := strings.TrimSpace(parts[12])
	var stealVal float64
	_, err := fmt.Sscanf(stealStr, "%f", &stealVal)
	if err != nil {
		t.Fatalf("Failed to parse stealVal: %v", err)
	}

	if stealVal < 14.7 || stealVal > 14.9 {
		t.Errorf("Expected stealVal ~14.8, got %f", stealVal)
	}
}

func TestCleanDiskOutputParsing(t *testing.T) {
	mockOutput := "BEFORE:12G\nDeleted Images: ...\nVacuuming logs...\nAFTER:15G\n"
	spaceBefore := ""
	spaceAfter := ""

	for _, line := range strings.Split(mockOutput, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "BEFORE:") {
			spaceBefore = strings.TrimPrefix(trimmed, "BEFORE:")
		} else if strings.HasPrefix(trimmed, "AFTER:") {
			spaceAfter = strings.TrimPrefix(trimmed, "AFTER:")
		}
	}

	if spaceBefore != "12G" {
		t.Errorf("Expected spaceBefore 12G, got %s", spaceBefore)
	}
	if spaceAfter != "15G" {
		t.Errorf("Expected spaceAfter 15G, got %s", spaceAfter)
	}
}
