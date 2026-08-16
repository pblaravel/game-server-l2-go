package packet

import (
	"encoding/binary"
	"math"
	"unicode/utf16"
)

// Writer builds little-endian L2 packets (Java SendablePacket / GameServerBasePacket).
type Writer struct {
	buf []byte
}

func NewWriter() *Writer {
	return &Writer{buf: make([]byte, 0, 64)}
}

func (w *Writer) Bytes() []byte { return w.buf }
func (w *Writer) Len() int      { return len(w.buf) }

func (w *Writer) WriteC(v int) {
	w.buf = append(w.buf, byte(v))
}

func (w *Writer) WriteB(data []byte) {
	w.buf = append(w.buf, data...)
}

func (w *Writer) WriteH(v int) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], uint16(v))
	w.buf = append(w.buf, b[:]...)
}

func (w *Writer) WriteD(v int32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], uint32(v))
	w.buf = append(w.buf, b[:]...)
}

func (w *Writer) WriteQ(v int64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(v))
	w.buf = append(w.buf, b[:]...)
}

func (w *Writer) WriteF32(v float32) {
	w.WriteD(int32(math.Float32bits(v)))
}

func (w *Writer) WriteF64(v float64) {
	w.WriteQ(int64(math.Float64bits(v)))
}

// WriteS writes a UTF-16LE null-terminated string (Java writeS).
func (w *Writer) WriteS(s string) {
	if s != "" {
		u16 := utf16.Encode([]rune(s))
		for _, r := range u16 {
			w.WriteH(int(r))
		}
	}
	w.WriteH(0)
}

// PadTo8 appends a 4-byte checksum placeholder (zeros) and pads to 8 bytes.
// Matches Java SendablePacket.buildPacket(true) and GameServerBasePacket.getBytes().
func (w *Writer) PadTo8() {
	w.WriteD(0)
	if rem := len(w.buf) % 8; rem != 0 {
		w.buf = append(w.buf, make([]byte, 8-rem)...)
	}
}

// Reader parses little-endian L2 packets (Java ReceivablePacket).
type Reader struct {
	data []byte
	pos  int
}

func NewReader(data []byte) *Reader {
	return &Reader{data: data}
}

func (r *Reader) Remaining() int { return len(r.data) - r.pos }
func (r *Reader) Pos() int       { return r.pos }
func (r *Reader) Bytes() []byte  { return r.data }

func (r *Reader) SkipOpcode() {
	if r.pos < len(r.data) {
		r.pos++
	}
}

func (r *Reader) ReadC() byte {
	if r.pos >= len(r.data) {
		return 0
	}
	v := r.data[r.pos]
	r.pos++
	return v
}

func (r *Reader) ReadB(n int) []byte {
	if n < 0 || r.pos+n > len(r.data) {
		n = len(r.data) - r.pos
		if n < 0 {
			return nil
		}
	}
	v := r.data[r.pos : r.pos+n]
	r.pos += n
	return append([]byte(nil), v...)
}

func (r *Reader) ReadH() int {
	if r.pos+2 > len(r.data) {
		return 0
	}
	v := int(binary.LittleEndian.Uint16(r.data[r.pos:]))
	r.pos += 2
	return v
}

func (r *Reader) ReadD() int32 {
	if r.pos+4 > len(r.data) {
		return 0
	}
	v := int32(binary.LittleEndian.Uint32(r.data[r.pos:]))
	r.pos += 4
	return v
}

func (r *Reader) ReadQ() int64 {
	if r.pos+8 > len(r.data) {
		return 0
	}
	v := int64(binary.LittleEndian.Uint64(r.data[r.pos:]))
	r.pos += 8
	return v
}

func (r *Reader) ReadF32() float32 {
	return math.Float32frombits(uint32(r.ReadD()))
}

func (r *Reader) ReadF64() float64 {
	return math.Float64frombits(uint64(r.ReadQ()))
}

// ReadS reads a UTF-16LE null-terminated string.
func (r *Reader) ReadS() string {
	start := r.pos
	end := start
	for end+1 < len(r.data) && (r.data[end] != 0 || r.data[end+1] != 0) {
		end += 2
	}
	r.pos = end + 2
	if r.pos > len(r.data) {
		r.pos = len(r.data)
	}
	if end <= start {
		return ""
	}
	n := (end - start) / 2
	u16 := make([]uint16, n)
	for i := 0; i < n; i++ {
		u16[i] = binary.LittleEndian.Uint16(r.data[start+i*2:])
	}
	return string(utf16.Decode(u16))
}
