package util

import (
	"io"
	"reflect"
	"unsafe"
)

const (
	defaultBufSize = 2048
)

type BufferError string

func (e BufferError) Error() string {
	return "websocket.buf:\n " + string(e)
}

type OutOfBufError string

func (e OutOfBufError) Error() string {
	return "websocket.buf:\n OutOfBufError: " + string(e)
}

type ReaderError string

func (e ReaderError) Error() string {
	return "websocket.buf:\n ReaderError: " + string(e)
}

const(
	ErrInSufficientBytes = BufferError("insufficient bytes available in the buffer")
	ErrReadNoEnough = BufferError("length of data read from Buffer is not enough")
)

type BaseBuf interface {
	Size() int

	Array() []byte

	Copy() BaseBuf

	WriteIndex(idx int) error

	ReadIndex(idx int) error

	GetWriteIndex() int

	GetReadIndex() int

	WriteableBytes() int

	ReadableBytes() int

	DiscardReadBytes()

	GetByte(i int) (b byte, err error)

	GetBytes(i int, dst []byte) error

	SetByte(i int, b byte) error

	SetBytes(i int, src []byte) error

	WriteTo(w io.Writer) error

	WriteToWithLen(len int, w io.Writer) error

	WriteByte(b byte) error

	WriteBytes(b []byte) (n int, err error)

	WriteString(s string) (n int, err error)

	ReadFrom(r io.Reader) (n int, err error)

	ReadFromWithLen(len int, r io.Reader) (n int, err error)

	ReadByte() (rb byte, err error)

	ReadBytes(b []byte) (n int, err error)

	ReadString(len int) (rs string, err error)
}

type ByteBuf struct {
	//reader index
	rIdx int
	//writer index
	wIdx int

	//buf
	array []byte
}

func NewBufWithSize(size int) BaseBuf {
	return &ByteBuf{
		rIdx:  0,
		wIdx:  0,
		array: make([]byte, size),
	}
}

func NewBufWithArray(b []byte) BaseBuf {
	return &ByteBuf{
		rIdx:  0,
		wIdx:  0,
		array: b,
	}
}

func NewBuf() BaseBuf {
	return NewBufWithSize(defaultBufSize)
}

func unsafeWriteString(s string, dstBuf BaseBuf) (n int, err error) {
	if dstBuf.WriteableBytes() < 1 {
		return 0, ErrInSufficientBytes
	}

	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
        Data: sh.Data,
        Len:  sh.Len,
        Cap:  sh.Len,
    }
	b := *(*[]byte)(unsafe.Pointer(&bh))

	n, err = dstBuf.WriteBytes(b)
	return
}

func unsafeReadString(len int, srcBuf BaseBuf) (str string, err error) {
	if srcBuf.ReadableBytes() < 1 {
		return "", ErrInSufficientBytes
	} else if srcBuf.ReadableBytes() < len {
		len = srcBuf.ReadableBytes()
	}

	rb := make([]byte, len)
	_, err = srcBuf.ReadBytes(rb)
	
	str = *(*string)(unsafe.Pointer(&rb))
	return
}


func checkIndex(idx int, buf BaseBuf) error {
	if idx < 0 {
		return OutOfBufError("idx must be a positive")
	}

	if idx > buf.Size() {
		return OutOfBufError("idx out of Buffer")
	}

	return nil
}

func (b *ByteBuf) ReadIndex(idx int) error {
	err := checkIndex(idx, b)

	if err != nil {
		return err
	}

	if idx > b.wIdx {
		return OutOfBufError("idx must be less than or equal to writeIndex")
	}

	b.rIdx = idx
	return nil
}

func (b *ByteBuf) WriteIndex(idx int) error {
	if idx < 0 {
		return OutOfBufError("idx must be a positive")
	}

	if idx > b.Size() {
		return OutOfBufError("idx must be less than or equal to bufsize")
	}

	b.rIdx = idx
	return nil
}

func (b *ByteBuf) GetWriteIndex() int {
	return b.wIdx
}

func (b *ByteBuf) GetReadIndex() int {
	return b.rIdx
}

func (b *ByteBuf) Size() int {
	return len(b.array)
}

func (b *ByteBuf) Array() []byte {
	return b.array
}

func (b *ByteBuf) Copy() BaseBuf {
	cp := make([]byte, b.Size())
	copy(cp, b.array)
	return NewBufWithArray(cp)
}

func (b *ByteBuf) WriteableBytes() int {
	return b.Size() - b.wIdx
}

func (b *ByteBuf) ReadableBytes() int {
	return b.wIdx - b.rIdx
}

func (buf *ByteBuf) GetByte(i int) (b byte, err error) {
	err = checkIndex(i, buf)
	if err != nil {
		return 0, err
	}

	b = buf.array[i]
	return
}

func (b *ByteBuf) GetBytes(i int, dst []byte) error {
	err := checkIndex(i, b)
	if err != nil {
		return err
	}

	copy(dst, b.array[i:])
	return nil
}

func (buf *ByteBuf) SetByte(i int, b byte) error {
	err := checkIndex(i, buf)
	if err != nil {
		return err
	}

	buf.array[i] = b
	return nil
}

func (b *ByteBuf) SetBytes(i int, src []byte) error {
	err := checkIndex(i, b)
	if err != nil {
		return err
	}

	copy(b.array[i:], src)
	return nil
}

func (b *ByteBuf) DiscardReadBytes() {
	if b.rIdx == 0 {
		return
	}

	copy(b.array, b.array[b.rIdx:b.wIdx])
}

func (b *ByteBuf) WriteTo(w io.Writer) error {
	return b.WriteToWithLen(b.ReadableBytes(), w)
}

func (b *ByteBuf) WriteToWithLen(len int, w io.Writer) (err error) {
	if len > b.ReadableBytes() {
		len = b.ReadableBytes()
		err = ErrInSufficientBytes
	}

	wb, err := w.Write(b.array[b.rIdx : b.rIdx + len])
	if wb < len && err == nil {
		err = io.ErrShortWrite
	}

	b.rIdx += wb
	return
}

func (buf *ByteBuf) WriteByte(b byte) error {
	if buf.WriteableBytes() < 1 {
		return ErrInSufficientBytes
	}

	buf.array[buf.wIdx] = b;
	buf.wIdx += 1
	return nil
}

func (buf *ByteBuf) WriteBytes(b []byte) (n int, err error) {
	if buf.WriteableBytes() < 1 {
		return 0, ErrInSufficientBytes
	}

	wb := copy(buf.array[buf.wIdx:], b)
	if wb < len(b) {
		err = ErrInSufficientBytes
	}

	buf.wIdx += wb

	return wb, err
}

func (buf *ByteBuf) WriteString(s string) (n int, err error) {
	return unsafeWriteString(s, buf)
}

func (b *ByteBuf) ReadFrom(r io.Reader) (n int, err error) {
	return b.ReadFromWithLen(b.WriteableBytes(), r)
}

func (b *ByteBuf) ReadFromWithLen(len int, r io.Reader) (n int, err error) {
	if b.WriteableBytes() < 1 {
		return 0, ErrInSufficientBytes
	}

	rb, err := r.Read(b.array[b.wIdx : b.wIdx+len])

	if rb < 0 {
		panic(ReaderError("reader returned nagative count from Read"))
	}

	b.wIdx += rb

	return rb, err
}

func (b *ByteBuf) ReadByte() (rb byte, err error) {
	if b.ReadableBytes() < 1 {
		return 0, ErrInSufficientBytes
	}

	rb = b.array[b.rIdx]
	b.rIdx += 1
	return
}

func (buf *ByteBuf) ReadBytes(b []byte) (n int, err error) {
	if buf.ReadableBytes() < 1 {
		return 0, ErrInSufficientBytes
	}

	n = copy(b, buf.array[buf.rIdx:buf.wIdx])

	if n < len(b) {
		err = ErrReadNoEnough
	}

	buf.rIdx += n
	return
}

func (b *ByteBuf) ReadString(len int) (string, error) {
	return unsafeReadString(len, b)
}