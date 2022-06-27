package vxid

import "testing"

func TestEncode(t *testing.T) {
	got, err := Encode("9dd05581-2562-4142-89b5-eaa601b8dcda", PfxMap.Instrument)
	if err != nil {
		t.Error(err)
	}
	want := "inst_tjy87Sg2xF7dkXtFtrkU6W"
	if got != want {
		t.Errorf("Encode incorrect, got: %s, want: %s", got, want)
	}
}

func TestDecode(t *testing.T) {
	got, err := Decode("inst_tjy87Sg2xF7dkXtFtrkU6W")
	if err != nil {
		t.Error(err)
	}
	want := "9dd05581-2562-4142-89b5-eaa601b8dcda"
	if got != want {
		t.Errorf("Decode incorrect, got: %s, want: %s", got, want)
	}
}
