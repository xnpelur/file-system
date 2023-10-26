package main

import (
	"file-system/internal/filesystem"
	"log"
)

func main() {
	err := filesystem.FormatFilesystem(20*1024*1024, 1024) // 20Mb - filesystem, 1kb - block
	if err != nil {
		log.Fatal(err)
	}
}
