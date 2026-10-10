package fonts

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"testing"

	"golang.org/x/image/font/opentype"
)

// sfntFromWOFF reassembles the tables of a WOFF 1.0 font into an SFNT.
func sfntFromWOFF(t *testing.T, woff []byte) []byte {
	t.Helper()
	numTables := int(binary.BigEndian.Uint16(woff[12:]))
	out := make([]byte, 12+16*numTables)
	copy(out, woff[4:8])
	binary.BigEndian.PutUint16(out[4:], uint16(numTables))
	for i := 0; i < numTables; i++ {
		entry := woff[44+20*i:]
		offset := binary.BigEndian.Uint32(entry[4:])
		compLength := binary.BigEndian.Uint32(entry[8:])
		origLength := binary.BigEndian.Uint32(entry[12:])
		data := woff[offset : offset+compLength]
		if compLength != origLength {
			reader, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if data, err = io.ReadAll(reader); err != nil {
				t.Fatal(err)
			}
		}
		record := out[12+16*i:]
		copy(record[0:4], entry[0:4])
		copy(record[4:8], entry[16:20])
		binary.BigEndian.PutUint32(record[8:], uint32(len(out)))
		binary.BigEndian.PutUint32(record[12:], origLength)
		out = append(out, data...)
		out = append(out, make([]byte, (4-len(data)%4)%4)...)
	}
	return out
}

func TestWOFFParsesAndPinsDate(t *testing.T) {
	for _, bold := range []bool{false, true} {
		woff, err := WOFF("beacon> zip \"C:\\Users\" [*] \u00d7\u25b2", bold)
		if err != nil {
			t.Fatal(err)
		}
		if string(woff[:4]) != "wOFF" {
			t.Fatal("missing WOFF signature")
		}
		sfnt := sfntFromWOFF(t, woff)
		if _, err := opentype.Parse(sfnt); err != nil {
			t.Fatalf("subset does not parse: %v", err)
		}
		original := regularTTF
		if bold {
			original = boldTTF
		}
		_, head, _, err := tableOffset(sfnt, "head")
		if err != nil {
			t.Fatal(err)
		}
		_, origHead, _, _ := tableOffset(original, "head")
		if !bytes.Equal(sfnt[head+28:head+36], original[origHead+28:origHead+36]) {
			t.Error("subset modification date is not the original font's date")
		}
	}
}

func TestWOFFRejectsMissingGlyph(t *testing.T) {
	if _, err := WOFF("\U0001F600", false); err == nil {
		t.Fatal("expected an error for a character the font lacks")
	}
}
