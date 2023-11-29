package directorymanager

import (
	"file-system/internal/filesystem/directory"
	"file-system/internal/filesystem/inode"
	"file-system/internal/utils"
	"os"
)

type DirectoryManager struct {
	Current           *directory.Directory
	CurrentInode      *inode.Inode
	CurrentInodeIndex uint32
	Path              string

	file         *os.File
	blockSize    uint32
	blocksOffset uint32

	savedDirectory  *directory.Directory
	savedInode      *inode.Inode
	savedInodeIndex uint32
	savedPath       string
}

func NewDirectoryManager(file *os.File, blockSize, blocksOffset uint32) *DirectoryManager {
	return &DirectoryManager{
		file:         file,
		blockSize:    blockSize,
		blocksOffset: blocksOffset,
	}
}

func (dm *DirectoryManager) OpenDirectory(dirInode *inode.Inode, inodeIndex uint32, name string) error {
	var err error
	data := make([]byte, 0)

	for i := 0; i < int(dirInode.FileSize); i++ {
		blockIndex := dirInode.Blocks[i]
		dirOffset := dm.blocksOffset + blockIndex*dm.blockSize

		tmpData := make([]byte, dm.blockSize)
		_, err = dm.file.ReadAt(tmpData, int64(dirOffset))
		if err != nil {
			return err
		}
		data = append(data, tmpData...)
	}

	dm.Current, err = directory.ReadDirectoryFromBytes(data)
	if err != nil {
		return err
	}
	dm.Path = utils.ChangeDirectoryPath(dm.Path, name)
	dm.CurrentInode = dirInode
	dm.CurrentInodeIndex = inodeIndex

	return nil
}

func (dm *DirectoryManager) CreateNewDirectory(dirInode *inode.Inode, inodeIndex uint32) (*directory.Directory, error) {
	var currDirInodeIndex uint32
	var err error

	if inodeIndex != 0 {
		currDirInodeIndex, err = dm.Current.GetInode(".")
		if err != nil {
			return nil, err
		}
	}
	newDir := directory.NewDirectory(inodeIndex, currDirInodeIndex)
	dm.saveDirectory(newDir, dirInode)

	return newDir, nil
}

func (dm *DirectoryManager) SaveCurrentDirectory() error {
	return dm.saveDirectory(dm.Current, dm.CurrentInode)
}

func (dm *DirectoryManager) SaveCurrentState() {
	dm.savedDirectory = dm.Current
	dm.savedInode = dm.CurrentInode
	dm.savedInodeIndex = dm.CurrentInodeIndex
	dm.savedPath = dm.Path
}

func (dm *DirectoryManager) LoadLastState() {
	if dm.savedDirectory != nil && dm.Path != dm.savedPath {
		dm.Current = dm.savedDirectory
		dm.CurrentInode = dm.savedInode
		dm.CurrentInodeIndex = dm.savedInodeIndex
		dm.Path = dm.savedPath
	}
}

func (dm *DirectoryManager) saveDirectory(dir *directory.Directory, dirInode *inode.Inode) error {
	data := dir.Encode()
	data = fitInBlocks(data, dm.blockSize)

	for i := 0; i < int(dirInode.FileSize); i++ {
		offset := dm.blocksOffset + dirInode.Blocks[i]*dm.blockSize

		sliceStart := int(dm.blockSize) * i
		sliceEnd := int(dm.blockSize) * (i + 1)
		if sliceEnd > len(data) {
			sliceEnd = len(data)
		}

		_, err := dm.file.WriteAt(data[sliceStart:sliceEnd], int64(offset))
		if err != nil {
			return err
		}
	}

	return nil
}

func fitInBlocks(data []byte, blockSize uint32) []byte {
	currentSize := uint32(len(data))
	remainder := currentSize % blockSize

	padding := make([]byte, blockSize-remainder)
	newData := append(data, padding...)

	return newData
}
