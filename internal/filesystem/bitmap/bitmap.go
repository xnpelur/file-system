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

func (b *Bitmap) SetBit(index int, value int) error {
	if index < 0 || index >= int(b.Size) {
		return errors.New("index out of bounds")
	}
	if value != 0 && value != 1 {
		return errors.New("invalid bit value")
	}

	byteIndex, bitOffset := index/8, uint(index%8)
	if value == 1 {
		b.Data[byteIndex] |= 1 << (7 - bitOffset)
	} else {
		b.Data[byteIndex] &^= 1 << (7 - bitOffset)
	}

	return nil
}

func (b *Bitmap) GetBit(index int) (int, error) {
	if index < 0 || index >= int(b.Size) {
		return 0, errors.New("index out of bounds")
	}
	byteIndex, bitOffset := index/8, uint(index%8)

	return int((b.Data[byteIndex] >> bitOffset) & 1), nil
}

func (b *Bitmap) ToByteArray() []byte {
	return b.Data
}

func (b Bitmap) WriteToFile(file *os.File, offset int) error {
	data := b.ToByteArray()

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}
