package main

import (
	"encoding/binary"
	"log"
	"os"
)

const (
	magicNumberOffset = 0
	inodeCountOffset  = 2
	blockCountOffset  = 6
)

func writeUint16ToFile(file *os.File, offset int, value uint16) error {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, value)
	_, err := file.WriteAt(b, int64(offset))
	return err
}

func writeUint32ToFile(file *os.File, offset int, value uint32) error {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, value)
	_, err := file.WriteAt(b, int64(offset))
	return err
}

func createFileSystem() error {
	file, err := os.Create("data")
	if err != nil {
		return err
	}
	defer file.Close()

	// Make this struct later...
	magicNumber := uint16(13)
	inodeCount := uint32(100)
	blockCount := uint32(200)

	if err := writeUint16ToFile(file, magicNumberOffset, magicNumber); err != nil {
		return err
	}

	if err := writeUint32ToFile(file, inodeCountOffset, inodeCount); err != nil {
		return err
	}

	if err := writeUint32ToFile(file, blockCountOffset, blockCount); err != nil {
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
