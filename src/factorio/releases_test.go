package factorio

import (
	"strings"
	"testing"
)

func TestParseArchiveVersionsSortsAndDeduplicates(t *testing.T) {
	html := `<a href="/download/archive/2.0.72">2.0.72</a>
<a href="/download/archive/2.1.7">2.1.7</a>
<a href="/download/archive/2.0.77">2.0.77</a>
<a href="/download/archive/2.0.72">2.0.72</a>`
	versions, err := parseArchiveVersions(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2.1.7", "2.0.77", "2.0.72"}
	if len(versions) != len(want) {
		t.Fatalf("versions = %v, want %v", versions, want)
	}
	for index := range want {
		if versions[index] != want[index] {
			t.Fatalf("versions = %v, want %v", versions, want)
		}
	}
}

func TestParseArchiveVersionsRejectsUnexpectedPage(t *testing.T) {
	if _, err := parseArchiveVersions(strings.NewReader("<html>maintenance</html>")); err == nil {
		t.Fatal("expected an error for an archive without versions")
	}
}
