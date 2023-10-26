package main

import (
	"file-system/bitmap"
	"file-system/superblock"
	"log"
	"os"
)

const (
	magicNumberOffset = 0
	inodeCountOffset  = 2
	blockCountOffset  = 6
)

func createFileSystem() error {
	file, err := os.Create("data")
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

	bm := bitmap.NewBitmap(16)
	bm.SetBit(1, 1)
	bm.SetBit(3, 1)
	bm.SetBit(5, 1)
	bm.SetBit(6, 1)
	bm.SetBit(6, 0)
	bm.SetBit(7, 1)

	err = bitmap.WriteBitmapToFile(file, 26, *bm)
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
