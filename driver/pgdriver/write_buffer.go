package pgdriver

import (
	"encoding/binary"
	"io"
	"sync"
)

var wbPool = sync.Pool{
	New: func() interface{} {
		return newWriteBuffer()
	},
}

func getWriteBuffer() *writeBuffer {
	wb := wbPool.Get().(*writeBuffer)
	return wb
}

func putWriteBuffer(wb *writeBuffer) {
	wb.Reset()
	wbPool.Put(wb)
}

type writeBuffer struct {
	Bytes []byte

	msgStart   int
	paramStart int
}

func newWriteBuffer() *writeBuffer {
	return &writeBuffer{
		Bytes: make([]byte, 0, 1024),
	}
}

func (b *writeBuffer) Reset() {
	b.Bytes = b.Bytes[:0]
}

func (b *writeBuffer) StartMessage(c byte) {
	if c == 0 {
		b.msgStart = len(b.Bytes)
		b.Bytes = append(b.Bytes, 0, 0, 0, 0)
	} else {
		b.msgStart = len(b.Bytes) + 1
		b.Bytes = append(b.Bytes, c, 0, 0, 0, 0)
	}
}

func (b *writeBuffer) FinishMessage() {
	binary.BigEndian.PutUint32(
		b.Bytes[b.msgStart:], uint32(len(b.Bytes)-b.msgStart))
}

func (b *writeBuffer) Query() []byte {
	return b.Bytes[b.msgStart+4 : len(b.Bytes)-1]
}

func (b *writeBuffer) StartParam() {
	b.paramStart = len(b.Bytes)
	b.Bytes = append(b.Bytes, 0, 0, 0, 0)
}

func (b *writeBuffer) FinishParam() {
	binary.BigEndian.PutUint32(
		b.Bytes[b.paramStart:], uint32(len(b.Bytes)-b.paramStart-4))
}

var nullParamLength = int32(-1)

func (b *writeBuffer) FinishNullParam() {
	binary.BigEndian.PutUint32(
		b.Bytes[b.paramStart:], uint32(nullParamLength))
}

func (b *writeBuffer) Write(data []byte) (int, error) {
	b.Bytes = append(b.Bytes, data...)
	return len(data), nil
}

func (b *writeBuffer) WriteInt16(num int16) {
	b.Bytes = append(b.Bytes, 0, 0)
	binary.BigEndian.PutUint16(b.Bytes[len(b.Bytes)-2:], uint16(num))
}

func (b *writeBuffer) WriteInt32(num int32) {
	b.Bytes = append(b.Bytes, 0, 0, 0, 0)
	binary.BigEndian.PutUint32(b.Bytes[len(b.Bytes)-4:], uint32(num))
}

func (b *writeBuffer) WriteString(s string) {
	b.Bytes = append(b.Bytes, s...)
	b.Bytes = append(b.Bytes, 0)
}

func (b *writeBuffer) WriteBytes(data []byte) {
	b.Bytes = append(b.Bytes, data...)
	b.Bytes = append(b.Bytes, 0)
}

func (b *writeBuffer) WriteByte(c byte) error {
	b.Bytes = append(b.Bytes, c)
	return nil
}

func (b *writeBuffer) ReadFrom(r io.Reader) (int64, error) {
	n, err := r.Read(b.Bytes[len(b.Bytes):cap(b.Bytes)])
	b.Bytes = b.Bytes[:len(b.Bytes)+n]
	return int64(n), err
}
