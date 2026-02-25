package docker

import (
	"encoding/binary"
	"fmt"
	"io"
)

// StreamType identifies which Docker multiplexed stream a frame belongs to.
type StreamType byte

const (
	StreamStdin  StreamType = 0
	StreamStdout StreamType = 1
	StreamStderr StreamType = 2
)

// Frame is a single decoded Docker multiplexed log frame.
type Frame struct {
	Type    StreamType
	Payload []byte
}

// ReadFrame reads one 8-byte header + payload from a Docker multiplexed log stream.
// The header format is:
//
//	byte 0:   stream type (0=stdin, 1=stdout, 2=stderr)
//	bytes 1-3: reserved (zero)
//	bytes 4-7: payload size (big-endian uint32)
func ReadFrame(r io.Reader) (Frame, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Frame{}, fmt.Errorf("reading frame header: %w", err)
	}

	streamType := StreamType(header[0])
	size := binary.BigEndian.Uint32(header[4:8])

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Frame{}, fmt.Errorf("reading frame payload: %w", err)
	}

	return Frame{Type: streamType, Payload: payload}, nil
}
