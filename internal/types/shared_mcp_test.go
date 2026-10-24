package types

import "testing"

func TestDefaultSharedMCPToolsAreReadOnlyThree(t *testing.T) {
	got := DefaultSharedMCPTools()
	want := []string{"list_knowledge_bases", "search_knowledge", "read_document"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %s want %s", i, got[i], want[i])
		}
	}
}
