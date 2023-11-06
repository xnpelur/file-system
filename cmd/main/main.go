package main

import (
	"file-system/internal/filesystem"
	"log"
)

func main() {
	fileSystem, err := filesystem.FormatFilesystem(1*1024*1024, 1024) // 1Mb - filesystem, 1kb - block
	if err != nil {
		log.Fatal(err)
	}
	defer fileSystem.CloseDataFile()
}
