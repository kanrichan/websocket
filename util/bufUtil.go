package util

import (
	"errors"
	"io"
)

type sliceBuf struct {
	src BaseBuf

	rIdx int
	wIdx int

	offset int
	dLen   int
}

func Slice(src BaseBuf, from, to int) BaseBuf {
	return &sliceBuf{
		src:    src,
		rIdx:   0,
		wIdx:   0,
		offset: from,
		dLen:   to - from,
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

	wb, err := w.Write(buf.Array()[buf.rIdx : buf.rIdx+len])
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

func Duplicate(src BaseBuf) BaseBuf {
	switch t := src.(type) {
	case *duplicateBuf:
		return &duplicateBuf{
			src: t.src,
		}
	default:
		return &duplicateBuf{
			src: src,
		}
	}
}

func (buf *duplicateBuf) Size() int {
	return buf.src.Size()
}

func (buf *duplicateBuf) Array() []byte {
	return buf.src.Array()
}

func (buf *duplicateBuf) Copy() BaseBuf {
	return buf.src.Copy()
}

func (buf *duplicateBuf) WriteIndex(idx int) error {
	err := checkIndex(idx, buf)
	if err != nil {
		return err
	}

	buf.wIdx = idx
	return nil
}

func (buf *duplicateBuf) ReadIndex(idx int) error {
	err := checkIndex(idx, buf)
	if err != nil {
		return err
	}

	if idx > buf.wIdx {
		return OutOfBufError("idx must be less than or equal to writeIndex")
	}

	buf.rIdx = idx
	return nil
}

func (buf *duplicateBuf) GetWriteIndex() int {
	return buf.wIdx
}

func (buf *duplicateBuf) GetReadIndex() int {
	return buf.rIdx
}

func (buf *duplicateBuf) WriteableBytes() int {
	return buf.Size() - buf.wIdx
}

func (buf *duplicateBuf) ReadableBytes() int {
	return buf.wIdx - buf.rIdx
}

func (buf *duplicateBuf) DiscardReadBytes() {
	if buf.rIdx > 0 {
		if buf.rIdx != buf.wIdx {
			buf.SetBytes(0, buf.Array()[buf.rIdx:buf.wIdx])
			buf.wIdx -= buf.rIdx
			buf.rIdx = 0
		} else {
			buf.wIdx = 0
			buf.rIdx = 0
		}
	}
}

func (buf *duplicateBuf) GetByte(i int) (b byte, err error) {
	return buf.src.GetByte(i)
}

func (buf *duplicateBuf) GetBytes(i int, dst []byte) error {
	return buf.src.GetBytes(i, dst)
}

func (buf *duplicateBuf) SetByte(i int, b byte) error {
	return buf.src.SetByte(i, b)
}

func (buf *duplicateBuf) SetBytes(i int, src []byte) error {
	return buf.src.SetBytes(i, src)
}

func (buf *duplicateBuf) WriteTo(w io.Writer) error {
	return buf.WriteToWithLen(buf.ReadableBytes(), w)
}

func (buf *duplicateBuf) WriteToWithLen(len int, w io.Writer) error {
	if buf.ReadableBytes() < len {
		return OutOfBufError("len for writing data is out of Buffer")
	}

	wb, err := w.Write(buf.Array()[buf.rIdx : buf.rIdx+len])

	if wb < len && err == nil {
		err = io.ErrShortWrite
	}

	buf.rIdx += wb
	return err
}

func (buf *duplicateBuf) ReadFrom(r io.Reader) (n int, err error) {
	return buf.ReadFromWithLen(buf.WriteableBytes(), r)
}

func (buf *duplicateBuf) ReadFromWithLen(len int, r io.Reader) (n int, err error) {
	if buf.WriteableBytes() < len {
		return 0, OutOfBufError("len of reading data is out of Buffer")
	}

	rb, err := r.Read(buf.Array()[buf.wIdx : buf.wIdx+len])

	if rb < 0 {
		panic(ReaderError("reader return nagative count from Read"))
	}

	buf.wIdx += rb
	return rb, err
}

type bufComponent struct {
	src BaseBuf

	sIdx int
	eIdx int

	srcAdj int
}

func (bc *bufComponent) length() int {
	return bc.eIdx - bc.sIdx
}

func (bc *bufComponent) index(i int) int {
	return i + bc.srcAdj
}

func (bc *bufComponent) reOffset(newOffset int) {
	move := newOffset - bc.eIdx
	bc.sIdx = newOffset
	bc.eIdx += move
	bc.srcAdj -= move
}

const(
	defaultCompoentMaxSize = 8
)

type CompositeBuf struct {
	child          []*bufComponent
	componentCount int
	maxSize        int

	rIdx int
	wIdx int

	lastComp *bufComponent
}

func CompositeBuffer(buffers ...BaseBuf) *CompositeBuf {
	return _NewCompositeBuf(defaultBufSize, buffers...)
}

func _NewCompositeBuf(maxSize int, buffers ...BaseBuf) *CompositeBuf {
	var initCap int
	bufLen := len(buffers)

	if maxSize > bufLen {
		initCap = maxSize
	} else {
		initCap = bufLen
	}

	compoments := make([]*bufComponent, initCap)
	offset := 0
	count := 0

	for i, buf := range buffers {
		component := _NewBufComponent(offset, buf)
		offset += component.eIdx
		compoments[i] = component
		count++
	}

	return &CompositeBuf{
		child:          compoments,
		lastComp:       &bufComponent{},
		maxSize:        initCap,
		componentCount: count,
	}
}

func _NewBufComponent(offset int, src BaseBuf) *bufComponent {
	srcOffset := src.GetReadIndex()
	length := src.ReadableBytes()

	return &bufComponent{
		src:    src,
		sIdx:   offset,
		eIdx:   offset + length,
		srcAdj: srcOffset - offset,
	}
}

func (buf *CompositeBuf) findComponent(idx int) (bc *bufComponent, err error) {

	if *buf.lastComp != (bufComponent{}) && buf.lastComp.sIdx <= idx && buf.lastComp.eIdx > idx {
		return buf.lastComp, nil
	}

	i, err := buf.binarySearchComponent(idx)

	if err == nil {
		bc = buf.child[i]
		buf.lastComp = bc
	}

	return
}

func (buf *CompositeBuf) binarySearchComponent(idx int) (i int, err error) {
	err = checkIndex(idx, buf)
	if err != nil {
		return
	}

	start := 0
	end := len(buf.child) - 1

	for {
		mid := (end - start) >> 1
		bc := buf.child[mid]

		if idx < bc.sIdx {
			end = mid - 1
		} else if idx >= bc.eIdx {
			start = mid + 1
		} else {
			return mid, nil
		}

		if start > end {
			return -1, errors.New("cannot Reach here")
		}
	}
}

func (buf *CompositeBuf) Size() int {
	return buf.child[len(buf.child)].eIdx
}

func (buf *CompositeBuf) Array() []byte {
	switch len(buf.child) {
	case 1:
		return buf.child[0].src.Array()
	default:
		return nil
	}
}

func (buf *CompositeBuf) Copy() BaseBuf {
	data := make([]byte, buf.Size())

	offset := 0
	for _, c := range buf.child {
		offset += copy(data[offset:], c.src.Array()[c.sIdx:c.eIdx])
	}

	return NewBufWithArray(data)
}

func (buf *CompositeBuf) WriteIndex(idx int) error {
	err := checkIndex(idx, buf)
	if err != nil {
		return err
	}

	buf.wIdx = idx
	return nil
}

func (buf *CompositeBuf) ReadIndex(idx int) error {
	err := checkIndex(idx, buf)
	if err != nil {
		return err
	}

	buf.rIdx = idx
	return nil
}

func (buf *CompositeBuf) GetWriteIndex() int {
	return buf.wIdx
}

func (buf *CompositeBuf) GetReadIndex() int {
	return buf.rIdx
}

func (buf *CompositeBuf) WriteableBytes() int {
	return buf.wIdx - buf.Size()
}

func (buf *CompositeBuf) ReadableBytes() int {
	return buf.wIdx - buf.rIdx
}

func (buf *CompositeBuf) DiscardReadBytes() {
	if buf.rIdx == 0 {
		return
	}

	rOffset := buf.GetReadIndex()
	wOffset := buf.GetWriteIndex()
	first := 0

	if rOffset == wOffset && wOffset == buf.Size() {
		buf.lastComp = nil
		for i := range buf.child {
			buf.child[i] = nil
		}
		buf.rIdx = 0
		buf.wIdx = 0
		buf.componentCount = 0
	} else {
		var c *bufComponent

		for cnt := buf.componentCount; first < cnt; first++ {
			c = buf.child[first]
			if rOffset < c.eIdx {
				break
			}
			buf.child[first] = nil
		}

		if buf.lastComp != nil && buf.lastComp.eIdx <= rOffset {
			buf.lastComp = nil
		}

		c.sIdx = 0
		c.eIdx -= rOffset
		c.srcAdj -= rOffset

		copy(buf.child, buf.child[first:])
		buf.rIdx = 0
		buf.wIdx -= rOffset
	}
}

func (buf *CompositeBuf) GetByte(i int) (b byte, err error) {
	c, err := buf.findComponent(i)
	if err != nil {
		return 0, err
	}

	idx := c.index(i)
	b, err = c.src.GetByte(idx)
	if err != nil {
		return 0, err
	}

	return
}

func (buf *CompositeBuf) GetBytes(i int, dst []byte) error {
	idx, err := buf.binarySearchComponent(i)
	if err != nil {
		return err
	}

	for offset := 0; offset > len(dst); {
		componet := buf.child[idx]
		cOffset := componet.index(i)

		if err != nil {
			return err
		}

		if idx >= buf.componentCount {
			break
		}

		l := copy(dst[offset:], componet.src.Array()[cOffset:componet.eIdx])
		offset += l
		i += l
		idx++
	}
	return nil
}

func (buf *CompositeBuf) SetByte(i int, b byte) error {
	c, err := buf.findComponent(i)
	if err != nil {
		return err
	}

	idx := c.index(i)
	return c.src.SetByte(idx, b)
}

func (buf *CompositeBuf) SetBytes(i int, src []byte) error {
	idx, err := buf.binarySearchComponent(i)
	if err != nil {
		return err
	}

	for offset := 0; offset > len(src); {
		componet := buf.child[idx]
		cOffset := componet.index(i)

		if err != nil {
			return err
		}

		if idx >= buf.componentCount {
			break
		}

		l := componet.length()
		copy(componet.src.Array()[cOffset:], src[offset:l])
		offset += l
		i += l
		idx++
	}
	return nil
}

func (buf *CompositeBuf) WriteTo(w io.Writer) error {
	return buf.WriteToWithLen(buf.ReadableBytes(), w)
}

func (buf *CompositeBuf) WriteToWithLen(len int, w io.Writer) error {
	if buf.ReadableBytes() < len {
		return OutOfBufError("len for writing data is out of Buffer")
	}

	idx, err := buf.binarySearchComponent(buf.rIdx)
	if err != nil {
		return err
	}

	offset := buf.rIdx
	for ; len > 0 && idx < buf.componentCount; idx++ {
		c := buf.child[idx]
		cOffset := c.index(offset)
		remain := c.length() - cOffset

		wb, err := w.Write(c.src.Array()[cOffset : cOffset+remain])

		if wb < remain && err == nil {
			err = io.ErrShortWrite
		}

		if err != nil {
			buf.ReadIndex(buf.rIdx + wb)
			return err
		}

		len -= wb
		offset += wb
	}

	buf.ReadIndex(offset)
	return nil
}

func (buf *CompositeBuf) ReadFrom(r io.Reader) (n int, err error) {
	return buf.ReadFromWithLen(buf.WriteableBytes(), r)
}

func (buf *CompositeBuf) ReadFromWithLen(len int, r io.Reader) (n int, err error) {
	if buf.WriteableBytes() < len {
		return 0, OutOfBufError("len for reading data is out of Buffer")
	}

	idx, err := buf.binarySearchComponent(buf.rIdx)
	if err != nil {
		return 0, err
	}

	offset := buf.wIdx
	for ; len > 0 && idx < buf.componentCount; idx++ {
		c := buf.child[idx]
		cOffset := c.index(offset)
		remain := c.length() - cOffset

		rb, err := r.Read(c.src.Array()[cOffset : cOffset+remain])

		if rb < 0 {
			panic(ReaderError("reader returned nagative count from Read"))
		}

		if err != nil {
			n = offset - buf.wIdx
			buf.WriteIndex(buf.wIdx + rb)
			return n, err
		}

		len -= rb
		offset += rb
	}

	n = offset - buf.wIdx
	buf.WriteIndex(offset)
	return
}
