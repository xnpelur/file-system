package bitmap

import (
	"fmt"
	"os"
)

type Bitmap struct {
	data []uint8
	size int
}

func NewBitmap(size int) *Bitmap {
	// Calculate the number of bytes required to store the specified number of bits.
	data := make([]uint8, (size+7)/8)
	return &Bitmap{data, size}
}

func (b *Bitmap) SetBit(index int, value int) {
	if index < 0 || index >= b.size || (value != 0 && value != 1) {
		return // Handle out-of-bounds index or invalid value
	}
	byteIndex, bitOffset := index/8, uint(index%8)
	if value == 1 {
		b.data[byteIndex] |= 1 << bitOffset
	} else {
		b.data[byteIndex] &^= 1 << bitOffset
	}
}

func (b *Bitmap) GetBit(index int) int {
	if index < 0 || index >= b.size {
		return -1 // Handle out-of-bounds index
	}
	byteIndex, bitOffset := index/8, uint(index%8)
	return int((b.data[byteIndex] >> bitOffset) & 1)
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
