package theme

import (
	"testing"
)

func TestResolveSelectsPalette(t *testing.T) {
	t.Parallel()

	// "auto", "", and unknown names all resolve to dark.
	for _, name := range []string{"auto", "", "bogus"} {
		got := Resolve(name)
		dark := Resolve("dark")
		if got.Header.GetForeground() != dark.Header.GetForeground() {
			t.Errorf("Resolve(%q).Header foreground differs from dark palette", name)
		}
	}

	// "light" is genuinely different from dark.
	light := Resolve("light")
	dark := Resolve("dark")
	if light.Header.GetForeground() == dark.Header.GetForeground() {
		t.Error("Resolve(\"light\") and Resolve(\"dark\") have the same Header foreground")
	}
}

func TestDarkAndLightDifferOnForegroundFields(t *testing.T) {
	t.Parallel()
	d := darkPalette()
	l := lightPalette()

	// Each field sets Foreground to a different color between dark and light.
	// We compare the color.Color values returned by GetForeground() directly.
	if d.Header.GetForeground() == l.Header.GetForeground() {
		t.Error("Header foreground: dark and light are equal")
	}
	if d.Success.GetForeground() == l.Success.GetForeground() {
		t.Error("Success foreground: dark and light are equal")
	}
	if d.Warning.GetForeground() == l.Warning.GetForeground() {
		t.Error("Warning foreground: dark and light are equal")
	}
	if d.Error.GetForeground() == l.Error.GetForeground() {
		t.Error("Error foreground: dark and light are equal")
	}
	if d.Muted.GetForeground() == l.Muted.GetForeground() {
		t.Error("Muted foreground: dark and light are equal")
	}
	if d.TableHeader.GetForeground() == l.TableHeader.GetForeground() {
		t.Error("TableHeader foreground: dark and light are equal")
	}
	if d.FormLabel.GetForeground() == l.FormLabel.GetForeground() {
		t.Error("FormLabel foreground: dark and light are equal")
	}

	// SidebarSelected: compare both foreground and background.
	if d.SidebarSelected.GetForeground() == l.SidebarSelected.GetForeground() {
		t.Error("SidebarSelected foreground: dark and light are equal")
	}
	if d.SidebarSelected.GetBackground() == l.SidebarSelected.GetBackground() {
		t.Error("SidebarSelected background: dark and light are equal")
	}
}

func TestPalettesSetBoldOnHeader(t *testing.T) {
	t.Parallel()
	d := darkPalette()
	l := lightPalette()

	if !d.Header.GetBold() {
		t.Error("dark Header is not bold")
	}
	if !l.Header.GetBold() {
		t.Error("light Header is not bold")
	}
}
