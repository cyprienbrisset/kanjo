package frontend

import (
	"io/fs"
	"strings"
	"testing"
)

func TestNativeBridgeEmbedded(t *testing.T) {
	b, err := fs.ReadFile(Assets, "dist/native-bridge.js")
	if err != nil {
		t.Fatalf("native-bridge.js doit être embarqué: %v", err)
	}
	s := string(b)
	for _, want := range []string{"window.kanjoOpenFiles", "window.go.main.App.OpenFiles", "kanjo:open-files"} {
		if !strings.Contains(s, want) {
			t.Errorf("native-bridge.js doit référencer %q", want)
		}
	}
}
