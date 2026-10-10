package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sceneText = `
[session]
computer = "WKS01"

[[command]]
input = "whoami"
  [[command.event]]
  at = 0.5
  kind = "output"
  text = "contoso\\alice"
`

func execute(t *testing.T, arguments ...string) (string, error) {
	t.Helper()
	command := New(Application{Version: "test"})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(arguments)
	err := command.Execute()
	return output.String(), err
}

func writeScene(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "demo.toml")
	if err := os.WriteFile(path, []byte(sceneText), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVersion(t *testing.T) {
	output, err := execute(t, "--version")
	if err != nil || output != "beaconcast test\n" {
		t.Fatalf("version = %q, %v", output, err)
	}
}

func TestRenderWritesDefaultOutputs(t *testing.T) {
	path := writeScene(t)
	for _, mode := range []string{"dark", "light"} {
		if _, err := execute(t, "render", "--fps", "4", "--mode", mode, path); err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		for _, extension := range []string{".svg", ".gif"} {
			info, err := os.Stat(strings.TrimSuffix(path, ".toml") + extension)
			if err != nil || info.Size() == 0 {
				t.Fatalf("%s: missing %s output: %v", mode, extension, err)
			}
		}
	}
}

func TestRenderSkipsOutputs(t *testing.T) {
	path := writeScene(t)
	if _, err := execute(t, "render", "--gif", "-", path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(strings.TrimSuffix(path, ".toml") + ".gif"); !os.IsNotExist(err) {
		t.Fatalf("GIF written despite --gif -: %v", err)
	}
}

func TestFrameFormats(t *testing.T) {
	path := writeScene(t)
	dir := filepath.Dir(path)
	if _, err := execute(t, "frame", "--at", "1", "-o", filepath.Join(dir, "f.png"), path); err != nil {
		t.Fatal(err)
	}
	if _, err := execute(t, "frame", "-o", filepath.Join(dir, "f.svg"), path); err != nil {
		t.Fatal(err)
	}
	png, _ := os.ReadFile(filepath.Join(dir, "f.png"))
	svg, _ := os.ReadFile(filepath.Join(dir, "f.svg"))
	if !bytes.HasPrefix(png, []byte("\x89PNG")) || !bytes.HasPrefix(svg, []byte("<svg")) {
		t.Fatal("frame outputs have the wrong format")
	}
	if _, err := execute(t, "frame", "--at", "999", path); err == nil {
		t.Fatal("expected an error for a time past the end")
	}
}

func TestSceneValidate(t *testing.T) {
	output, err := execute(t, "scene", "validate", writeScene(t))
	if err != nil || !strings.Contains(output, "1 commands, 1 events") {
		t.Fatalf("validate = %q, %v", output, err)
	}
	bad := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(bad, []byte("[[command]]\ninput = \"x\"\ncolour = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execute(t, "scene", "validate", bad); err == nil || !strings.Contains(err.Error(), "colour") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestUnknownMode(t *testing.T) {
	if _, err := execute(t, "render", "--mode", "dim", writeScene(t)); err == nil || !strings.Contains(err.Error(), "light") {
		t.Fatalf("expected mode error, got %v", err)
	}
}
