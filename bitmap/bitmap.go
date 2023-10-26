package bitmap

import (
	"errors"
	"fmt"
	"os"
)

type Bitmap struct {
	data []uint8
	size int
}

func NewBitmap(size int) *Bitmap {
	data := make([]uint8, (size+7)/8)
	return &Bitmap{data, size}
}

func (b *Bitmap) SetBit(index int, value int) error {
	if index < 0 || index >= b.size {
		return errors.New("Index out of bounds")
	}
	if value != 0 && value != 1 {
		return errors.New("Invalid bit value")
	}

	byteIndex, bitOffset := index/8, uint(index%8)
	if value == 1 {
		b.data[byteIndex] |= 1 << (7 - bitOffset)
	} else {
		b.data[byteIndex] &^= 1 << (7 - bitOffset)
	}

	return nil
}

func (b *Bitmap) GetBit(index int) (int, error) {
	if index < 0 || index >= b.size {
		return 0, errors.New("Index out of bounds")
	}
	byteIndex, bitOffset := index/8, uint(index%8)

	return int((b.data[byteIndex] >> bitOffset) & 1), nil
}

func (b *Bitmap) ToByteArray() []byte {
	return b.data
}

func WriteBitmapToFile(file *os.File, offset int, value Bitmap) error {
	data := value.ToByteArray()
	fmt.Println(data)

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}
