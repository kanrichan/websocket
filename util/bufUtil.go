package util

import "io"

type sliceBuf struct {
	src BaseBuf

	rIdx int
	wIdx int

	offset int
	dLen   int
}

func slice(src BaseBuf, from, to int) BaseBuf {
	return &sliceBuf{
		src: src,
		rIdx: 0,
		wIdx: 0,
		offset: from,
		dLen: to - from,
	}
}

func (buf *sliceBuf) Size() int {
	return buf.dLen
}

func (buf *sliceBuf) Array() []byte {
	return buf.src.Array()[buf.offset : buf.offset+buf.dLen]
}

func (buf *sliceBuf) Copy() BaseBuf {
	cp := make([]byte, buf.Size())
	copy(cp, buf.Array())
	return NewBufWithArray(cp)
}

func (buf *sliceBuf) WriteIndex(idx int) error {
	err := checkIndex(idx, buf)
	if err != nil {
		return err
	}

	buf.rIdx = idx
	return nil
}

func (buf *sliceBuf) ReadIndex(idx int) error {
	err := checkIndex(idx, buf)
	if err != nil {
		return err
	}

	buf.wIdx = idx
	return nil
}

func (buf *sliceBuf) GetWriteIndex() int {
	return buf.wIdx
}

func (buf *sliceBuf) GetReadIndex() int {
	return buf.rIdx
}

func (buf *sliceBuf) WriteableBytes() int {
	return buf.Size() - buf.wIdx
}

func (buf *sliceBuf) ReadableBytes() int {
	return buf.wIdx - buf.rIdx
}

func (buf *sliceBuf) DiscardReadBytes() {
	if buf.rIdx == 0 {
		return
	}

	copy(buf.Array(), buf.Array()[buf.rIdx:buf.wIdx])
}

func (buf *sliceBuf) GetByte(i int) (b byte, err error) {
	err = checkIndex(i, buf)
	if err != nil {
		return 0, err
	}

	b = buf.Array()[i+buf.offset]
	return
}

func (buf *sliceBuf) GetBytes(i int, dst []byte) error {
	err := checkIndex(i, buf)
	if err != nil {
		return err
	}

	copy(dst, buf.Array()[i:])
	return nil
}

func (buf *sliceBuf) SetByte(i int, b byte) error {
	err := checkIndex(i, buf)
	if err != nil {
		return err
	}

	buf.Array()[i] = b
	return nil
}

func (buf *sliceBuf) SetBytes(i int, src []byte) error {
	err := checkIndex(i, buf)
	if err != nil {
		return err
	}

	copy(buf.Array()[i:], src)
	return err
}

func (buf *sliceBuf) WriteTo(w io.Writer) error {
	return buf.WriteToWithLen(buf.WriteableBytes(), w)
}

func (buf *sliceBuf) WriteToWithLen(len int, w io.Writer) error {
	if buf.ReadableBytes() < len {
		return OutOfBufError("len for writing data is out of Buffer")
	}

	wb, err := w.Write(buf.Array()[buf.rIdx:buf.rIdx + len])
	if wb < len && err == nil {
		err = io.ErrShortWrite
	}

	buf.rIdx += wb
	return err
}

func (buf *sliceBuf) ReadFrom(r io.Reader) (n int, err error) {
	return buf.ReadFromWithLen(buf.WriteableBytes(), r)
}

func (buf *sliceBuf) ReadFromWithLen(len int, r io.Reader) (n int, err error) {
	if buf.WriteableBytes() < len {
		return 0, OutOfBufError("len for reading data is out of Buffer")
	}

	rb, err := r.Read(buf.Array()[buf.wIdx : buf.wIdx+len])

	if rb < 0 {
		panic(ReaderError("reader returned nagative count from Read"))
	}

	buf.wIdx += rb

	return rb, err
}

type duplicateBuf struct {
	src BaseBuf

	rIdx int
	wIdx int
}

func duplicate(src BaseBuf) BaseBuf {

}

func (buf *duplicateBuf)Size() int {

}

func (buf *duplicateBuf)Array() []byte {

}

func (buf *duplicateBuf)Copy() BaseBuf {

}

func (buf *duplicateBuf)WriteIndex(idx int) error {

}

func (buf *duplicateBuf)ReadIndex(idx int) error {

}

func (buf *duplicateBuf)GetWriteIndex() int {

}

func (buf *duplicateBuf)GetReadIndex() int {

}

func (buf *duplicateBuf)WriteableBytes() int {

}

func (buf *duplicateBuf)ReadableBytes() int {

}

func (buf *duplicateBuf)DiscardReadBytes() {

}

func (buf *duplicateBuf)GetByte(i int) (b byte, err error) {

}

func (buf *duplicateBuf)GetBytes(i int, dst []byte) error {

}

func (buf *duplicateBuf)SetByte(i int, b byte) error {

}

func (buf *duplicateBuf)SetBytes(i int, src []byte) error {

}

func (buf *duplicateBuf)WriteTo(w io.Writer) error {

}

func (buf *duplicateBuf)WriteToWithLen(len int, w io.Writer) error {

}

func (buf *duplicateBuf)ReadFrom(r io.Reader) (n int, err error) {

}

func (buf *duplicateBuf)ReadFromWithLen(len int, r io.Reader) (n int, err error) {

}

type bufComponent struct {
	src BaseBuf

	sIdx int
	eIdx int
}

type CompositeBuf struct {
	child []bufComponent

	rIdx int
	wIdx int

	lastComp bufComponent
}

func (buf *CompositeBuf)Size() int {
	
}

func (buf *CompositeBuf)Array() []byte {
	
}

func (buf *CompositeBuf)Copy() BaseBuf {
	
}

func (buf *CompositeBuf)WriteIndex(idx int) error {
	
}

func (buf *CompositeBuf)ReadIndex(idx int) error {
	
}

func (buf *CompositeBuf)GetWriteIndex() int {
	
}

func (buf *CompositeBuf)GetReadIndex() int {
	
}

func (buf *CompositeBuf)WriteableBytes() int {
	
}

func (buf *CompositeBuf)ReadableBytes() int {
	
}

func (buf *CompositeBuf)DiscardReadBytes() {
	
}

func (buf *CompositeBuf)GetByte(i int) (b byte, err error) {
	
}

func (buf *CompositeBuf)GetBytes(i int, dst []byte) error {
	
}

func (buf *CompositeBuf)SetByte(i int, b byte) error {
	
}

func (buf *CompositeBuf)SetBytes(i int, src []byte) error {
	
}

func (buf *CompositeBuf)WriteTo(w io.Writer) error {
	
}

func (buf *CompositeBuf)WriteToWithLen(len int, w io.Writer) error {
	
}

func (buf *CompositeBuf)ReadFrom(r io.Reader) (n int, err error) {
	
}

func (buf *CompositeBuf)ReadFromWithLen(len int, r io.Reader) (n int, err error) {
	
}