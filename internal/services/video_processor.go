package services

import (
	"bytes"
	"io"

	"github.com/abema/go-mp4"
)

type VideoProcessor struct{}

func NewVideoProcessor() *VideoProcessor {
	return &VideoProcessor{}
}

// MemoryFile is a simple in-memory implementation of io.WriteSeeker
type MemoryFile struct {
	buf []byte
	pos int64
}

func (m *MemoryFile) Write(p []byte) (n int, err error) {
	if m.pos+int64(len(p)) > int64(len(m.buf)) {
		newBuf := make([]byte, m.pos+int64(len(p)))
		copy(newBuf, m.buf)
		m.buf = newBuf
	}
	n = copy(m.buf[m.pos:], p)
	m.pos += int64(n)
	return n, nil
}

func (m *MemoryFile) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = m.pos + offset
	case io.SeekEnd:
		newPos = int64(len(m.buf)) + offset
	}
	if newPos < 0 {
		return 0, io.EOF
	}
	m.pos = newPos
	return newPos, nil
}

func (m *MemoryFile) Bytes() []byte {
	return m.buf
}

// StripAudio removes the audio track ('soun' handler) from an MP4 file
// by omitting its corresponding 'trak' box from the 'moov' box.
func (p *VideoProcessor) StripAudio(mp4Data []byte) ([]byte, error) {
	memFile := &MemoryFile{}
	w := mp4.NewWriter(memFile)

	// Check if a trak is an audio trak
	isAudioTrak := func(trakOffset uint64, trakSize uint64) bool {
		isAudio := false
		r := bytes.NewReader(mp4Data)
		r.Seek(int64(trakOffset), io.SeekStart)
		
		mp4.ReadBoxStructure(r, func(h *mp4.ReadHandle) (interface{}, error) {
			// Jika sudah di luar kotak trak, stop
			if h.BoxInfo.Offset >= trakOffset+trakSize {
				return nil, nil
			}
			if h.BoxInfo.Type == mp4.BoxTypeHdlr() {
				box, _, err := h.ReadPayload()
				if err == nil {
					hdlr, ok := box.(*mp4.Hdlr)
					if ok && hdlr.HandlerType == [4]byte{'s', 'o', 'u', 'n'} {
						isAudio = true
					}
				}
			}
			if h.BoxInfo.IsSupportedType() {
				return h.Expand()
			}
			return nil, nil
		})
		return isAudio
	}

	// Recursively copy boxes, skipping audio traks
	var copyBoxes func(offset uint64, size uint64) error
	copyBoxes = func(offset uint64, size uint64) error {
		r := bytes.NewReader(mp4Data)
		r.Seek(int64(offset), io.SeekStart)
		
		_, err := mp4.ReadBoxStructure(r, func(h *mp4.ReadHandle) (interface{}, error) {
			if h.BoxInfo.Offset >= offset+size {
				return nil, nil // Out of bounds
			}

			if h.BoxInfo.Type == mp4.BoxTypeTrak() {
				if isAudioTrak(h.BoxInfo.Offset, h.BoxInfo.Size) {
					// Skip audio trak completely
					return nil, nil
				}
			}

			isContainer := h.BoxInfo.Type == mp4.BoxTypeMoov() || 
			               h.BoxInfo.Type == mp4.BoxTypeTrak() ||
			               h.BoxInfo.Type == mp4.BoxTypeMdia() ||
			               h.BoxInfo.Type == mp4.BoxTypeMinf() ||
			               h.BoxInfo.Type == mp4.BoxTypeDinf() ||
			               h.BoxInfo.Type == mp4.BoxTypeStbl() ||
			               h.BoxInfo.Type == mp4.BoxTypeEdts()
			               
			if isContainer {
				_, err := w.StartBox(&h.BoxInfo)
				if err != nil {
					return nil, err
				}
				
				err = copyBoxes(h.BoxInfo.Offset+uint64(h.BoxInfo.HeaderSize), h.BoxInfo.Size-uint64(h.BoxInfo.HeaderSize))
				if err != nil {
					return nil, err
				}
				
				_, err = w.EndBox()
				return nil, err
			}

			// Leaf box
			_, err := w.StartBox(&h.BoxInfo)
			if err != nil {
				return nil, err
			}
			
			payloadReader := bytes.NewReader(mp4Data)
			payloadReader.Seek(int64(h.BoxInfo.Offset)+int64(h.BoxInfo.HeaderSize), io.SeekStart)
			_, err = io.CopyN(w, payloadReader, int64(h.BoxInfo.Size)-int64(h.BoxInfo.HeaderSize))
			if err != nil {
				return nil, err
			}
			
			_, err = w.EndBox()
			return nil, err
		})
		return err
	}

	err := copyBoxes(0, uint64(len(mp4Data)))
	if err != nil {
		return nil, err
	}

	return memFile.Bytes(), nil

}
