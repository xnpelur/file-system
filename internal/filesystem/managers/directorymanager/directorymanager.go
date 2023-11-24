package directorymanager

import (
	"file-system/internal/filesystem/directory"
	"file-system/internal/filesystem/inode"
	"os"
)

type DirectoryManager struct {
	Current      *directory.Directory
	CurrentInode *inode.Inode
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

func (dm *DirectoryManager) CreateNewDirectory(inodeIndex, blockIndex uint32, path string) (*directory.Directory, error) {
	var currDirInodeIndex uint32
	var err error

	if path != "/" {
		currDirInodeIndex, err = dm.Current.GetInode(".")
		if err != nil {
			return nil, err
		}
	}
	newDir := directory.NewDirectory(inodeIndex, currDirInodeIndex)
	newDir.WriteAt(dm.file, dm.blocksOffset+blockIndex*dm.blockSize)

	return newDir, nil
}

func (dm DirectoryManager) ReadDirectory(blockIndex uint32) (*directory.Directory, error) {
	offset := dm.blocksOffset + blockIndex*dm.blockSize
	return directory.ReadDirectoryAt(dm.file, offset)
}

func (dm DirectoryManager) SaveCurrentDirectory() error {
	offset := dm.blocksOffset + dm.CurrentInode.Blocks[0]*dm.blockSize
	return dm.Current.WriteAt(dm.file, offset)
}
