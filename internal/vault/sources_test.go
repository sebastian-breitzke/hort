package vault

import (
	"testing"
)

func TestSourceRegistryPersistence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	recs, err := ListSources()
	if err != nil {
		t.Fatalf("ListSources() empty: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("expected empty registry, got %d", len(recs))
	}

	if err := AddSource(SourceRecord{Name: "one", Path: "/tmp/one.enc", KDF: KDFRawKey}); err != nil {
		t.Fatalf("AddSource(one): %v", err)
	}
	if err := AddSource(SourceRecord{Name: "two", Path: "/tmp/two.enc", KDF: KDFRawKey}); err != nil {
		t.Fatalf("AddSource(two): %v", err)
	}

	recs, err = ListSources()
	if err != nil {
		t.Fatalf("ListSources() after adds: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(recs))
	}
	if recs[0].Name != "one" || recs[1].Name != "two" {
		t.Fatalf("unexpected order: %+v", recs)
	}

	// Replace
	if err := AddSource(SourceRecord{Name: "one", Path: "/tmp/one-new.enc", KDF: KDFRawKey}); err != nil {
		t.Fatalf("AddSource(replace): %v", err)
	}
	recs, _ = ListSources()
	if recs[0].Path != "/tmp/one-new.enc" {
		t.Fatalf("replace failed, got path %q", recs[0].Path)
	}

	// Remove
	if err := RemoveSource("one"); err != nil {
		t.Fatalf("RemoveSource(one): %v", err)
	}
	recs, _ = ListSources()
	if len(recs) != 1 || recs[0].Name != "two" {
		t.Fatalf("remove failed, got %+v", recs)
	}

	if err := RemoveSource("nope"); err == nil {
		t.Fatal("expected error removing unknown source")
	}
}

func TestInvalidSourceNames(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cases := []string{"", "primary", "UPPER", "has space", "-lead", ".lead"}
	for _, name := range cases {
		if err := AddSource(SourceRecord{Name: name, Path: "/tmp/x.enc"}); err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
}

func TestMountedRefReservesPrimary(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := MountedRefAt("primary", "/tmp/x.enc"); err == nil {
		t.Fatal("expected error mounting reserved name 'primary'")
	}
}
