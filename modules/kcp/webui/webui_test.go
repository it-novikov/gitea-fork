package webui

import (
	"strings"
	"testing"
)

func TestRenderCapsuleRegistryPage(t *testing.T) {
	html, err := Render(PageCapsules, DemoDataSet())
	if err != nil {
		t.Fatalf("render capsules page: %v", err)
	}
	for _, expected := range []string{"Capsule Registry", "kyba.backend.task-workflow.api.v1", "kyba-desktop", "KYBa KCP"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected rendered page to contain %q", expected)
		}
	}
}

func TestRenderExportPage(t *testing.T) {
	html, err := Render(PageExport, DemoDataSet())
	if err != nil {
		t.Fatalf("render export page: %v", err)
	}
	for _, expected := range []string{"Archive Export Preview", "gitea-fork", "kyba-backend", "Ownership digest"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected rendered page to contain %q", expected)
		}
	}
}
