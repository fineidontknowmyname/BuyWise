package provenance

import "testing"

func TestIsReal(t *testing.T) {
	cases := map[Source]bool{Stub: false, Fixture: false, Manual: true, Calculated: true, Source(""): false}
	for source, want := range cases {
		if got := source.IsReal(); got != want {
			t.Errorf("%q.IsReal() = %v, want %v", source, got, want)
		}
	}
}

func TestAllReal(t *testing.T) {
	real := Labelled{Key: "a", Source: Manual}
	stub := Labelled{Key: "b", Source: Stub}
	if !AllReal(nil) {
		t.Error("no values should count as all real")
	}
	if !AllReal([]Labelled{real}) {
		t.Error("manual value should be real")
	}
	if AllReal([]Labelled{real, stub}) {
		t.Error("a stub value makes the set not real")
	}
}
