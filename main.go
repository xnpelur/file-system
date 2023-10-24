package main

import (
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

	return nil
}

func main() {
	err := createFileSystem()
	if err != nil {
		log.Fatal(err)
	}
}
