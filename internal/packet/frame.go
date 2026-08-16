package packet

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ReadFrame reads a uint16-LE length-prefixed L2 packet (length includes the 2-byte header).
func ReadFrame(r io.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := int(binary.LittleEndian.Uint16(hdr[:]))
	if n < 2 {
		return nil, fmt.Errorf("invalid packet length %d", n)
	}
	body := make([]byte, n-2)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}

// WriteFrame writes a uint16-LE length-prefixed L2 packet.
func WriteFrame(w io.Writer, payload []byte) error {
	n := len(payload) + 2
	if n > 0xFFFF {
		return fmt.Errorf("packet too large: %d", n)
	}
	var hdr [2]byte
	binary.LittleEndian.PutUint16(hdr[:], uint16(n))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
