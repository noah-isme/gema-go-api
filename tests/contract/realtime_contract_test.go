package contract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type openAPISpec struct {
	Paths      map[string]map[string]json.RawMessage `json:"paths"`
	Components struct {
		Schemas map[string]json.RawMessage `json:"schemas"`
	} `json:"components"`
}

func TestRealtimeSpecificationIncludesRealtimeEndpoints(t *testing.T) {
	spec := loadSpec(t, "docs/api/realtime.json")

	requiredPaths := []string{
		"/api/v2/chat/ws",
		"/api/v2/chat/history",
		"/api/v2/notifications",
		"/api/v2/notifications/stream",
		"/api/v2/notifications/{id}/read",
		"/api/v2/discussion/threads",
		"/api/v2/discussion/threads/{id}",
		"/api/v2/discussion/replies",
	}

	for _, path := range requiredPaths {
		if _, ok := spec.Paths[path]; !ok {
			t.Fatalf("expected realtime spec to contain path %s", path)
		}
	}

	for _, schema := range []string{"ChatMessage", "Notification", "DiscussionThread"} {
		if _, ok := spec.Components.Schemas[schema]; !ok {
			t.Fatalf("expected realtime spec to contain schema %s", schema)
		}
	}
}

func TestSupportingSpecificationIncludesPublicEndpoints(t *testing.T) {
	spec := loadSpec(t, "docs/api/supporting.json")

	requiredPaths := []string{
		"/api/activities/active",
		"/api/announcements",
		"/api/gallery",
		"/api/contact",
		"/api/upload",
		"/api/seed/announcements",
		"/api/seed/gallery",
	}

	for _, path := range requiredPaths {
		if _, ok := spec.Paths[path]; !ok {
			t.Fatalf("expected supporting spec to contain path %s", path)
		}
	}

	for _, schema := range []string{"ActivityFeedEnvelope", "Announcement", "GalleryItem", "ContactRequest", "UploadResponse"} {
		if _, ok := spec.Components.Schemas[schema]; !ok {
			t.Fatalf("expected supporting spec to contain schema %s", schema)
		}
	}
}

func loadSpec(t *testing.T, relative string) openAPISpec {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve caller")
	}
	base := filepath.Join(filepath.Dir(filename), "..", "..")
	fullPath := filepath.Join(base, relative)

	raw, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", fullPath, err)
	}
	var spec openAPISpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("failed to unmarshal %s: %v", fullPath, err)
	}
	return spec
}
