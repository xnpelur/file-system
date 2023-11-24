package directorymanager

import (
	"file-system/internal/filesystem/directory"
	"os"
)

type DirectoryManager struct {
	Current      *directory.Directory
	Path         string
	file         *os.File
	blockSize    uint32
	blocksOffset uint32
}

func NewDirectoryManager(file *os.File, blockSize, blocksOffset uint32) *DirectoryManager {
	return &DirectoryManager{
		file:         file,
		blockSize:    blockSize,
		blocksOffset: blocksOffset,
	}
}

func (dm *DirectoryManager) OpenDirectory(blockIndex uint32, path string) error {
	var err error
	dirOffset := dm.blocksOffset + blockIndex*dm.blockSize
	dm.Current, err = directory.ReadDirectoryAt(dm.file, dirOffset)
	if err != nil {
		return err
	}
	dm.Path = path

	return nil
}
