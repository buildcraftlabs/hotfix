//go:build windows

package main

import (
	"encoding/binary"
	"testing"
)

func TestPngToICO_ValidICOHeader(t *testing.T) {
	// Minimal 1x1 PNG (valid PNG bytes).
	fakePNG := minimalPNG()

	ico := pngToICO(pngFrame{size: 16, data: fakePNG})

	if len(ico) < 6 {
		t.Fatal("ICO output too short to contain header")
	}

	// Bytes 0-1: reserved (must be 0).
	reserved := binary.LittleEndian.Uint16(ico[0:2])
	if reserved != 0 {
		t.Errorf("ICO reserved field: got %d, want 0", reserved)
	}

	// Bytes 2-3: type (must be 1 for ICO).
	icoType := binary.LittleEndian.Uint16(ico[2:4])
	if icoType != 1 {
		t.Errorf("ICO type field: got %d, want 1", icoType)
	}

	// Bytes 4-5: image count (we passed 1 frame).
	count := binary.LittleEndian.Uint16(ico[4:6])
	if count != 1 {
		t.Errorf("ICO image count: got %d, want 1", count)
	}
}

func TestPngToICO_TotalSize(t *testing.T) {
	fakePNG := minimalPNG()
	ico := pngToICO(pngFrame{size: 32, data: fakePNG})

	// ICO header (6) + 1 ICONDIRENTRY (16) + PNG data
	expectedSize := 6 + 16 + len(fakePNG)
	if len(ico) != expectedSize {
		t.Errorf("ICO size: got %d, want %d", len(ico), expectedSize)
	}
}

func TestPngToICO_MultiFrame(t *testing.T) {
	fakePNG := minimalPNG()
	ico := pngToICO(
		pngFrame{size: 16, data: fakePNG},
		pngFrame{size: 32, data: fakePNG},
	)

	count := binary.LittleEndian.Uint16(ico[4:6])
	if count != 2 {
		t.Errorf("ICO image count for 2-frame ICO: got %d, want 2", count)
	}

	expectedSize := 6 + 16*2 + len(fakePNG)*2
	if len(ico) != expectedSize {
		t.Errorf("2-frame ICO size: got %d, want %d", len(ico), expectedSize)
	}
}

func TestPngToICO_ImageDataOffset(t *testing.T) {
	fakePNG := minimalPNG()
	ico := pngToICO(pngFrame{size: 16, data: fakePNG})

	// ICONDIRENTRY offset field is at bytes 18-21 (6 header + 12 into entry).
	offset := binary.LittleEndian.Uint32(ico[18:22])
	// For 1 frame: header (6) + 1 entry (16) = 22.
	if offset != 22 {
		t.Errorf("ICO image data offset: got %d, want 22", offset)
	}
}

func TestPngToICO_ImageDataMatchesPNG(t *testing.T) {
	fakePNG := minimalPNG()
	ico := pngToICO(pngFrame{size: 16, data: fakePNG})

	// The PNG data should start at byte 22.
	embedded := ico[22:]
	if len(embedded) != len(fakePNG) {
		t.Errorf("embedded PNG length: got %d, want %d", len(embedded), len(fakePNG))
	}
	for i := range fakePNG {
		if embedded[i] != fakePNG[i] {
			t.Errorf("embedded PNG byte %d: got %x, want %x", i, embedded[i], fakePNG[i])
			break
		}
	}
}

// minimalPNG returns a minimal valid 1x1 PNG for use in tests.
func minimalPNG() []byte {
	// Hardcoded 1x1 transparent PNG (67 bytes).
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // PNG signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk length + type
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // width=1, height=1
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, // bitdepth=8, colortype=6 (RGBA)
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, // IHDR CRC, IDAT chunk
		0x54, 0x78, 0x9c, 0x62, 0x00, 0x00, 0x00, 0x02, // IDAT data
		0x00, 0x01, 0xe2, 0x21, 0xbc, 0x33, 0x00, 0x00, // IDAT CRC
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, // IEND chunk
		0x60, 0x82, // IEND CRC
	}
}
