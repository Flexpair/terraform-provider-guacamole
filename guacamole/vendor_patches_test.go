package guacamole

import (
	"os"
	"strings"
	"testing"
)

// readVendorFile reads a file from the repository's vendor directory. Go
// tests run with the package source directory as working directory, so the
// vendor directory is one level up.
func readVendorFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run scripts/restore-vendored-guacamole-patches.sh if a vendor refresh dropped our patches)", path, err)
	}
	return string(content)
}

// TestVendoredGuacamoleClientPatches guards the local modifications documented
// in the README (see scripts/restore-vendored-guacamole-patches.sh). It fails
// when a `go mod vendor` refresh silently drops them, or when the vendored
// client's error format changes out from under isNotFoundError.
func TestVendoredGuacamoleClientPatches(t *testing.T) {
	clientSrc := readVendorFile(t, "../vendor/github.com/techBeck03/guacamole-api-client/client.go")
	usersSrc := readVendorFile(t, "../vendor/github.com/techBeck03/guacamole-api-client/types/users.go")
	connectionsSrc := readVendorFile(t, "../vendor/github.com/techBeck03/guacamole-api-client/types/connections.go")

	// isNotFoundError matches "status code 404" against the Call error format.
	if !strings.Contains(clientSrc, "failed with status code %d") {
		t.Error("vendored client error format changed; update isNotFoundError and TestIsNotFoundError")
	}
	if strings.Contains(clientSrc, "ioutil.ReadAll") || !strings.Contains(clientSrc, "io.ReadAll(resp.Body)") {
		t.Error("vendored client is missing the io.ReadAll patch (see scripts/restore-vendored-guacamole-patches.sh)")
	}
	if !strings.Contains(usersSrc, "LastActive int64") {
		t.Error("vendored users type is missing the LastActive int64 patch (see scripts/restore-vendored-guacamole-patches.sh)")
	}
	if !strings.Contains(connectionsSrc, "xterm-256color") || strings.Contains(connectionsSrc, `"xterm-25color"`) {
		t.Error("vendored connections type is missing the xterm-256color patch (see scripts/restore-vendored-guacamole-patches.sh)")
	}
}
