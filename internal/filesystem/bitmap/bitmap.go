package bitmap

import (
	"errors"
	"os"
)

type Bitmap struct {
	Data   []uint8
	size   uint32
	file   *os.File
	offset uint32
}

func NewBitmap(size uint32, file *os.File, offset uint32) *Bitmap {
	data := make([]uint8, (size+7)/8)
	return &Bitmap{data, size, file, offset}
}

func (b Bitmap) Size() uint32 {
	return b.size / 8
}

func (b *Bitmap) SetBit(index uint32, value int) error {
	if index >= b.size {
		return errors.New("index out of bounds")
	}
	if value != 0 && value != 1 {
		return errors.New("invalid bit value")
	}

	byteIndex, bitOffset := index/8, index%8
	if value == 1 {
		b.Data[byteIndex] |= 1 << (7 - bitOffset)
	} else {
		b.Data[byteIndex] &^= 1 << (7 - bitOffset)
	}

	return nil
}

func (b *Bitmap) GetBit(index uint32) (int, error) {
	if index >= b.size {
		return 0, errors.New("index out of bounds")
	}
	byteIndex, bitOffset := index/8, 7-index%8

	return int((b.Data[byteIndex] >> bitOffset) & 1), nil
}

func (b *Bitmap) TakeFreeBit() (uint32, error) {
	for i := uint32(0); i < b.size; i++ {
		bit, err := b.GetBit(i)
		if err != nil {
			return 0, err
		}
		if bit == 0 {
			err := b.SetBit(i, 1)
			if err != nil {
				return 0, err
			}
			return i, nil
		}
	}
	return 0, errors.New("no zero bits found")
}

func ReadBitmapAt(file *os.File, offset uint32, size uint32) (*Bitmap, error) {
	data := make([]uint8, size)

	_, err := file.ReadAt(data, int64(offset))
	if err != nil {
		return nil, err
	}

	return &Bitmap{data, size, file, offset}, nil
}

func (b Bitmap) Save() error {
	return b.writeAt(b.file, b.offset)
}

func (b Bitmap) writeAt(file *os.File, offset uint32) error {
	_, err := file.WriteAt(b.Data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}
