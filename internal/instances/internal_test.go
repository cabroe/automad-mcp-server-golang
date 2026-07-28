package instances

import (
	"strings"
	"testing"
)

func TestParsePsLine_Running(t *testing.T) {
	line := strings.Join([]string{
		"abc123def456",
		DefaultImage,
		"Up 3 minutes",
		"0.0.0.0:8080->80/tcp, :::8080->80/tcp",
		"2026-07-28 12:00:00 +0000 UTC",
		"myinstance",
		"/" + ContainerPrefix + "myinstance",
	}, psFieldSep)

	inst, ok := parsePsLine(line)
	if !ok {
		t.Fatalf("expected parsePsLine to succeed for a well-formed line")
	}
	if inst.ContainerID != "abc123def456" {
		t.Errorf("ContainerID = %q, want %q", inst.ContainerID, "abc123def456")
	}
	if inst.Name != "myinstance" {
		t.Errorf("Name = %q, want %q", inst.Name, "myinstance")
	}
	if inst.ContainerName != ContainerPrefix+"myinstance" {
		t.Errorf("ContainerName = %q, want %q", inst.ContainerName, ContainerPrefix+"myinstance")
	}
	if !inst.Running {
		t.Error("expected Running=true for a status starting with 'Up'")
	}
	if inst.HostPort != 8080 {
		t.Errorf("HostPort = %d, want 8080", inst.HostPort)
	}
}

func TestParsePsLine_Stopped(t *testing.T) {
	line := strings.Join([]string{
		"abc123def456",
		DefaultImage,
		"Exited (0) 2 hours ago",
		"",
		"2026-07-28 10:00:00 +0000 UTC",
		"myinstance",
		"/" + ContainerPrefix + "myinstance",
	}, psFieldSep)

	inst, ok := parsePsLine(line)
	if !ok {
		t.Fatalf("expected parsePsLine to succeed")
	}
	if inst.Running {
		t.Error("expected Running=false for an 'Exited' status")
	}
	if inst.HostPort != 0 {
		t.Errorf("HostPort = %d, want 0 for a stopped container with no port mapping", inst.HostPort)
	}
}

func TestParsePsLine_MissingNameLabelFallsBackToContainerName(t *testing.T) {
	line := strings.Join([]string{
		"abc123def456",
		DefaultImage,
		"Up 1 second",
		"",
		"2026-07-28 12:00:00 +0000 UTC",
		"", // empty name label
		"/" + ContainerPrefix + "fallback-name",
	}, psFieldSep)

	inst, ok := parsePsLine(line)
	if !ok {
		t.Fatalf("expected parsePsLine to succeed")
	}
	if inst.Name != "fallback-name" {
		t.Errorf("Name = %q, want %q (derived from container name)", inst.Name, "fallback-name")
	}
}

func TestParsePsLine_MalformedRejected(t *testing.T) {
	if _, ok := parsePsLine("not-enough-fields"); ok {
		t.Error("expected parsePsLine to reject a line with the wrong field count")
	}
}

func TestParsePsOutput_MultipleLinesAndBlanks(t *testing.T) {
	line1 := strings.Join([]string{"id1", DefaultImage, "Up 1 minute", "", "now", "a", "/" + ContainerPrefix + "a"}, psFieldSep)
	line2 := strings.Join([]string{"id2", DefaultImage, "Up 1 minute", "", "now", "b", "/" + ContainerPrefix + "b"}, psFieldSep)
	output := "\n" + line1 + "\n\n" + line2 + "\n"

	instances := parsePsOutput(output)
	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d: %+v", len(instances), instances)
	}
	if instances[0].Name != "a" || instances[1].Name != "b" {
		t.Errorf("unexpected names: %q, %q", instances[0].Name, instances[1].Name)
	}
}

func TestParsePsOutput_Empty(t *testing.T) {
	if got := parsePsOutput(""); got != nil {
		t.Errorf("expected nil for empty output, got %+v", got)
	}
}
