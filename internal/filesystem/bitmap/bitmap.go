package bitmap

import (
	"errors"
	"os"
)

type Bitmap struct {
	Data []uint8
	Size uint32
}

func NewBitmap(size uint32) *Bitmap {
	data := make([]uint8, (size+7)/8)
	return &Bitmap{data, size}
}

func (b *Bitmap) SetBit(index uint32, value int) error {
	if index >= b.Size {
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
	if index >= b.Size {
		return 0, errors.New("index out of bounds")
	}
	byteIndex, bitOffset := index/8, 7-index%8

	return int((b.Data[byteIndex] >> bitOffset) & 1), nil
}

func (b *Bitmap) TakeFreeBit() (uint32, error) {
	for i := uint32(0); i < b.Size; i++ {
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

func (b *Bitmap) ToByteArray() []byte {
	return b.Data
}

func (b Bitmap) WriteAt(file *os.File, offset uint32) error {
	data := b.ToByteArray()

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}
