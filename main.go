package main

import (
	"file-system/bitmap"
	"file-system/superblock"
	"log"
	"os"
)

func createFileSystem() error {
	file, err := os.Create("filesystem.data")
	if err != nil {
		return err
	}
	defer file.Close()

	sb := superblock.NewSuperblock(
		100,
		200,
		300,
		400,
		500,
		600,
	)

	err = superblock.WriteSuperBlockToFile(file, 0, sb)
	if err != nil {
		return err
	}

	blockBitmap := bitmap.NewBitmap(16)
	blockBitmap.SetBit(1, 1)
	blockBitmap.SetBit(3, 1)
	blockBitmap.SetBit(5, 1)
	blockBitmap.SetBit(6, 1)
	blockBitmap.SetBit(6, 0)
	blockBitmap.SetBit(7, 1)

	err = bitmap.WriteBitmapToFile(file, 26, *blockBitmap)
	if err != nil {
		return err
	}

	inodeBitmap := bitmap.NewBitmap(16)
	inodeBitmap.SetBit(0, 1)
	inodeBitmap.SetBit(2, 1)
	inodeBitmap.SetBit(4, 1)
	inodeBitmap.SetBit(8, 1)
	inodeBitmap.SetBit(10, 1)
	inodeBitmap.SetBit(16, 1)

	err = bitmap.WriteBitmapToFile(file, 26+2, *inodeBitmap)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	err := createFileSystem()
	if err != nil {
		log.Fatal(err)
	}
}
