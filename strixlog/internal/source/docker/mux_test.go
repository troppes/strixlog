package docker

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func buildFrame(streamType byte, payload []byte) []byte {
	header := make([]byte, 8)
	header[0] = streamType
	size := uint32(len(payload))
	header[4] = byte(size >> 24)
	header[5] = byte(size >> 16)
	header[6] = byte(size >> 8)
	header[7] = byte(size)
	return append(header, payload...)
}

func TestReadFrame(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		wantType   StreamType
		wantPayload string
		wantErr    bool
	}{
		{
			name:        "stdout frame",
			data:        buildFrame(1, []byte("hello stdout\n")),
			wantType:    StreamStdout,
			wantPayload: "hello stdout\n",
		},
		{
			name:        "stderr frame",
			data:        buildFrame(2, []byte("error msg\n")),
			wantType:    StreamStderr,
			wantPayload: "error msg\n",
		},
		{
			name:        "stdin frame",
			data:        buildFrame(0, []byte("input")),
			wantType:    StreamStdin,
			wantPayload: "input",
		},
		{
			name:    "empty reader returns error",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "truncated header returns error",
			data:    []byte{1, 0, 0},
			wantErr: true,
		},
		{
			name:    "header with truncated payload returns error",
			data:    buildFrame(1, []byte("hello"))[:10], // cut payload short
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(tc.data)
			frame, err := ReadFrame(r)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if frame.Type != tc.wantType {
				t.Errorf("Type = %d, want %d", frame.Type, tc.wantType)
			}
			if string(frame.Payload) != tc.wantPayload {
				t.Errorf("Payload = %q, want %q", frame.Payload, tc.wantPayload)
			}
		})
	}
}

func TestReadFrameMultiple(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildFrame(1, []byte("line1\n")))
	buf.Write(buildFrame(2, []byte("line2\n")))

	f1, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("first frame: %v", err)
	}
	if f1.Type != StreamStdout || string(f1.Payload) != "line1\n" {
		t.Errorf("unexpected first frame: type=%d payload=%q", f1.Type, f1.Payload)
	}

	f2, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("second frame: %v", err)
	}
	if f2.Type != StreamStderr || string(f2.Payload) != "line2\n" {
		t.Errorf("unexpected second frame: type=%d payload=%q", f2.Type, f2.Payload)
	}

	_, err = ReadFrame(&buf)
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected EOF after frames exhausted, got %v", err)
	}
}
