package cli

import (
	"testing"
)

func TestGetGlobalFlags(t *testing.T) {
	flags := getGlobalFlags()
	if len(flags) == 0 {
		t.Fatal("expected some global flags")
	}
	flagNames := map[string]bool{}
	for _, f := range flags {
		for _, name := range f.Names() {
			flagNames[name] = true
		}
	}
	for _, name := range []string{"duration", "header", "insecure", "log-level", "output", "protocol", "rate", "service-name"} {
		if !flagNames[name] {
			t.Errorf("expected flag %q to be present", name)
		}
	}
}
