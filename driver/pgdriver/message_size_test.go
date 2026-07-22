package pgdriver

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

func TestFinishMessage_TooLargeSetsError(t *testing.T) {
	orig := maxMessageSize
	maxMessageSize = 8
	defer func() { maxMessageSize = orig }()

	wb := newWriteBuffer()
	wb.StartMessage(queryMsg)
	wb.WriteString("some query that exceeds the tiny test limit")
	wb.FinishMessage()

	if !errors.Is(wb.err, errMessageTooLarge) {
		t.Fatalf("wb.err = %v, want errMessageTooLarge", wb.err)
	}
}

func TestConnWrite_RefusesOversizedMessage(t *testing.T) {
	wb := newWriteBuffer()
	wb.setErr(errMessageTooLarge)

	// wb.err is set, so write must return it without touching the (nil) netConn.
	if err := (&Conn{}).write(context.Background(), wb); !errors.Is(err, errMessageTooLarge) {
		t.Fatalf("write() = %v, want errMessageTooLarge", err)
	}
}

func TestFinishParam_TooLargeSetsError(t *testing.T) {
	orig := maxMessageSize
	maxMessageSize = 4
	defer func() { maxMessageSize = orig }()

	wb := newWriteBuffer()
	wb.StartParam()
	wb.Write([]byte("larger than the limit"))
	wb.FinishParam()

	if !errors.Is(wb.err, errMessageTooLarge) {
		t.Fatalf("wb.err = %v, want errMessageTooLarge", wb.err)
	}
}

func TestWriteBindExecute_TooManyParams(t *testing.T) {
	args := make([]driver.NamedValue, math.MaxInt16+1)
	err := writeBindExecute(context.Background(), &Conn{}, "", args)
	if err == nil {
		t.Fatal("writeBindExecute with > MaxInt16 params: got nil error, want an error")
	}
}

func TestReadMessageType_RejectsShortLength(t *testing.T) {
	// type byte + 4-byte length field of 0 (< 4, invalid).
	buf := []byte{'X', 0, 0, 0, 0}
	rd := newReader(bytes.NewReader(buf), 1024)
	if _, _, err := readMessageType(rd); !errors.Is(err, errInvalidMessageLength) {
		t.Fatalf("readMessageType short length: err = %v, want errInvalidMessageLength", err)
	}

	// A valid length (>= 4) is accepted and the body length is length-4.
	valid := []byte{'X', 0, 0, 0, 0}
	binary.BigEndian.PutUint32(valid[1:], 10)
	rd = newReader(bytes.NewReader(valid), 1024)
	c, n, err := readMessageType(rd)
	if err != nil || c != 'X' || n != 6 {
		t.Fatalf("readMessageType valid: (%q, %d, %v), want ('X', 6, nil)", c, n, err)
	}
}

func TestReadTemp_RejectsNegative(t *testing.T) {
	rd := newReader(bytes.NewReader(nil), 1024)
	if _, err := rd.ReadTemp(-1); !errors.Is(err, errInvalidMessageLength) {
		t.Fatalf("ReadTemp(-1) = %v, want errInvalidMessageLength", err)
	}
}
