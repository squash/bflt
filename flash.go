package main

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
)

type FlashBlock struct {
	Number      int
	Offset      int
	Data        []byte
	PadToSize   int
	PadWithData byte
	Filename    string
}
type Flash struct {
	Blocks          []*FlashBlock
	Size            int
	count           int
	AutomaticOffset bool
}

func NewFlash() *Flash {
	var f Flash
	f.count = 0
	return &f
}

func (f *Flash) SetSize(s int) {
	f.Size = s
}


func (f *Flash) DeleteBlock(b *FlashBlock) {
	var newb []*FlashBlock
	for x := range f.Blocks {
		if f.Blocks[x].Number != b.Number {
			newb = append(newb, f.Blocks[x])
		}
	}
	f.Blocks = newb
}

func (f *Flash) Sort() {
	sort.Slice(f.Blocks, func(x, y int) bool {
		return f.Blocks[x].Number > f.Blocks[y].Number
	})
}

func (f *Flash) NewBlock() *FlashBlock {
	var b FlashBlock
	b.Number = f.count
	f.count++
	f.Blocks=append(f.Blocks, &b)
	return &b
}

// TODO: don't allow blocks to overlap
func (f *Flash) Assemble() ([]byte, []byte, error) {
	image := make([]byte, f.Size)
	x := 0
	for x = 0; x < f.Size; x++ {
		image[x] = 0
	}
	log.Printf("Image is %d bytes", x)
	if f.AutomaticOffset {
		var locations []string
		locations = append(locations, "Start Address, Length, End Address, Filename")

		start := 0
		for _, b := range f.Blocks {
			nun := copy(image[start:], b.Data[:])
			if nun != len(b.Data) {
				return nil, nil, fmt.Errorf("expected %d bytes but got %d instead", len(b.Data), nun)
			}
			if b.PadToSize != nun {
				for x := start + nun; x < start+b.PadToSize; x++ {
					image[x] = b.PadWithData
				}

			}
			end := start + b.PadToSize
			locations = append(locations, fmt.Sprintf("0x%x,0x%x,0x%x,%s", start, b.PadToSize, end, b.Filename))
			start = end
		}
		l := strings.Join(locations, "\n")
		return image, []byte(l), nil
	} else {
		for _, b := range f.Blocks {
			// Make sure the block fits
			if len(b.Data) > f.Size-b.Offset {
				return nil, nil, errors.New("block length would overrun image size")
			}
			nun := copy(image[b.Offset:], b.Data[:])
			if nun != len(b.Data) {
				return nil, nil, errors.New("short copy")
			}
		}
	}
	return image, nil, nil
}
