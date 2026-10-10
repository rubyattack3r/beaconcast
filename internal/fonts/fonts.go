// Package fonts embeds JetBrains Mono and prepares it for each renderer.
package fonts

import (
	"bytes"
	"compress/zlib"
	_ "embed"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"

	tfont "github.com/tdewolff/font"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var (
	//go:embed JetBrainsMono-Regular.ttf
	regularTTF []byte
	//go:embed JetBrainsMono-Bold.ttf
	boldTTF []byte

	parseOnce sync.Once
	regular   *opentype.Font
	bold      *opentype.Font
	parseErr  error

	faceMu sync.Mutex
	faces  = map[faceKey]font.Face{}
)

type faceKey struct {
	size float64
	bold bool
}

func parsed() error {
	parseOnce.Do(func() {
		if regular, parseErr = opentype.Parse(regularTTF); parseErr != nil {
			return
		}
		bold, parseErr = opentype.Parse(boldTTF)
	})
	return parseErr
}

// Face returns a cached raster face at the given pixel size.
func Face(size float64, isBold bool) (font.Face, error) {
	if err := parsed(); err != nil {
		return nil, err
	}
	faceMu.Lock()
	defer faceMu.Unlock()
	key := faceKey{size, isBold}
	if face, ok := faces[key]; ok {
		return face, nil
	}
	source := regular
	if isBold {
		source = bold
	}
	face, err := opentype.NewFace(source, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	faces[key] = face
	return face, nil
}

// WOFF returns a WOFF 1.0 font containing only the glyphs needed for text.
func WOFF(text string, isBold bool) ([]byte, error) {
	data := regularTTF
	if isBold {
		data = boldTTF
	}
	sfnt, err := tfont.ParseSFNT(data, 0)
	if err != nil {
		return nil, err
	}
	seen := map[uint16]bool{0: true}
	ids := []uint16{0}
	// The SVG renderer writes spaces as U+00A0.
	for _, r := range text + "\u00a0" {
		id := sfnt.GlyphIndex(r)
		if id == 0 && r != ' ' {
			return nil, fmt.Errorf("JetBrains Mono has no glyph for %q", r)
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	subset, err := sfnt.Subset(ids, tfont.SubsetOptions{Tables: tfont.KeepMinTables})
	if err != nil {
		return nil, err
	}
	pinned, err := pinModified(subset.Write(), data)
	if err != nil {
		return nil, err
	}
	return wrapWOFF(pinned)
}

// tableOffset returns the directory record position, offset, and length of
// an SFNT table.
func tableOffset(font []byte, tag string) (int, int, int, error) {
	if len(font) < 12 {
		return 0, 0, 0, fmt.Errorf("font too short")
	}
	numTables := int(binary.BigEndian.Uint16(font[4:]))
	for i := 0; i < numTables; i++ {
		record := 12 + 16*i
		if record+16 > len(font) {
			break
		}
		if string(font[record:record+4]) == tag {
			offset := int(binary.BigEndian.Uint32(font[record+8:]))
			length := int(binary.BigEndian.Uint32(font[record+12:]))
			if offset+length > len(font) {
				return 0, 0, 0, fmt.Errorf("%s table out of range", tag)
			}
			return record, offset, length, nil
		}
	}
	return 0, 0, 0, fmt.Errorf("font has no %s table", tag)
}

func checksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		var word [4]byte
		copy(word[:], data[i:])
		sum += binary.BigEndian.Uint32(word[:])
	}
	return sum
}

// pinModified replaces the modification date that the subsetter stamps with
// the current time with the original font's date, keeping output
// reproducible, and recomputes the checksums that cover it.
func pinModified(subset, original []byte) ([]byte, error) {
	_, origHead, _, err := tableOffset(original, "head")
	if err != nil {
		return nil, err
	}
	record, head, length, err := tableOffset(subset, "head")
	if err != nil {
		return nil, err
	}
	if length < 36 {
		return nil, fmt.Errorf("head table too short")
	}
	out := append([]byte(nil), subset...)
	copy(out[head+28:head+36], original[origHead+28:origHead+36])
	binary.BigEndian.PutUint32(out[head+8:], 0)
	binary.BigEndian.PutUint32(out[record+4:], checksum(out[head:head+length]))
	binary.BigEndian.PutUint32(out[head+8:], 0xB1B0AFBA-checksum(out))
	return out, nil
}

// wrapWOFF converts an SFNT font to WOFF 1.0 with zlib-compressed tables.
func wrapWOFF(sfnt []byte) ([]byte, error) {
	if len(sfnt) < 12 {
		return nil, fmt.Errorf("font too short")
	}
	numTables := int(binary.BigEndian.Uint16(sfnt[4:]))
	type table struct {
		tag, checksum, offset, length uint32
		data                          []byte
	}
	tables := make([]table, numTables)
	for i := range tables {
		record := sfnt[12+16*i:]
		t := table{
			tag:      binary.BigEndian.Uint32(record[0:]),
			checksum: binary.BigEndian.Uint32(record[4:]),
			offset:   binary.BigEndian.Uint32(record[8:]),
			length:   binary.BigEndian.Uint32(record[12:]),
		}
		if int(t.offset+t.length) > len(sfnt) {
			return nil, fmt.Errorf("font table out of range")
		}
		t.data = sfnt[t.offset : t.offset+t.length]
		tables[i] = t
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].tag < tables[j].tag })

	pad4 := func(n int) int { return (n + 3) &^ 3 }
	headerSize := 44 + 20*numTables
	var body bytes.Buffer
	directory := make([]byte, 20*numTables)
	for i, t := range tables {
		stored := t.data
		var compressed bytes.Buffer
		writer, err := zlib.NewWriterLevel(&compressed, zlib.BestCompression)
		if err != nil {
			return nil, err
		}
		if _, err := writer.Write(t.data); err != nil {
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		if compressed.Len() < len(t.data) {
			stored = compressed.Bytes()
		}
		entry := directory[20*i:]
		binary.BigEndian.PutUint32(entry[0:], t.tag)
		binary.BigEndian.PutUint32(entry[4:], uint32(headerSize+body.Len()))
		binary.BigEndian.PutUint32(entry[8:], uint32(len(stored)))
		binary.BigEndian.PutUint32(entry[12:], t.length)
		binary.BigEndian.PutUint32(entry[16:], t.checksum)
		body.Write(stored)
		body.Write(make([]byte, pad4(body.Len())-body.Len()))
	}
	totalSFNT := 12 + 16*numTables
	for _, t := range tables {
		totalSFNT += pad4(int(t.length))
	}
	header := make([]byte, 44)
	copy(header[0:], "wOFF")
	copy(header[4:], sfnt[0:4])
	binary.BigEndian.PutUint32(header[8:], uint32(headerSize+body.Len()))
	binary.BigEndian.PutUint16(header[12:], uint16(numTables))
	binary.BigEndian.PutUint32(header[16:], uint32(totalSFNT))
	binary.BigEndian.PutUint16(header[20:], 1)

	out := append(header, directory...)
	return append(out, body.Bytes()...), nil
}
