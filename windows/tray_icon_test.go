//go:build windows

package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// The four embedded monochrome flame PNGs must be present and look like PNGs.
func TestTrayIconsEmbedded(t *testing.T) {
	icons := map[string][]byte{
		"tray_white16": trayWhite16,
		"tray_white32": trayWhite32,
		"tray_black16": trayBlack16,
		"tray_black32": trayBlack32,
	}
	for name, data := range icons {
		if len(data) == 0 {
			t.Errorf("%s is empty (embed failed)", name)
			continue
		}
		if !bytes.HasPrefix(data, pngMagic) {
			t.Errorf("%s is not a PNG (bad magic)", name)
		}
	}
}

// iconBytes must return a structurally valid 2-frame ICO for whichever theme is
// active, so systray.SetIcon always gets usable data.
func TestIconBytes_ValidICOForCurrentTheme(t *testing.T) {
	ico := iconBytes()
	if len(ico) < 6 {
		t.Fatalf("ICO too short: %d bytes", len(ico))
	}
	if reserved := binary.LittleEndian.Uint16(ico[0:2]); reserved != 0 {
		t.Errorf("ICO reserved field: got %d, want 0", reserved)
	}
	if typ := binary.LittleEndian.Uint16(ico[2:4]); typ != 1 {
		t.Errorf("ICO type: got %d, want 1 (icon)", typ)
	}
	if count := binary.LittleEndian.Uint16(ico[4:6]); count != 2 {
		t.Errorf("ICO frame count: got %d, want 2 (16px + 32px)", count)
	}
}

// systemUsesLightTheme reads the registry; it must never panic and must return
// a usable bool regardless of whether the value exists.
func TestSystemUsesLightTheme_NoPanic(t *testing.T) {
	_ = systemUsesLightTheme()
}
