//go:build darwin

package governedcheck

import (
	"context"
	"strings"
	"testing"
)

func TestSeatbeltAllowsSystemPythonRuntimeWithoutWideningNetwork(t *testing.T) {
	enforcement := seatbelt{}
	profile := enforcement.profile()
	if !strings.Contains(profile, `/Applications/Xcode.app/Contents/Developer`) ||
		!strings.Contains(profile, `/Library/Developer/CommandLineTools`) {
		t.Fatalf("profile omits Apple developer runtime roots:\n%s", profile)
	}
	if !strings.Contains(profile, "(deny network*)") {
		t.Fatalf("profile widened network while allowing interpreter resources:\n%s", profile)
	}
	cmd, err := enforcement.Command(context.Background(), Request{
		Argv: []string{"python3", "-c", "print('governed-python-ok')"},
	}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("system Python could not load inside governed check sandbox: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != "governed-python-ok" {
		t.Fatalf("system Python output = %q", output)
	}
}
