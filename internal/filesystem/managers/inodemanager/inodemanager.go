package inodemanager

import (
	"file-system/internal/filesystem/inode"
	"os"
)

type InodeManager struct {
	file             *os.File
	inodeSize        uint32
	inodeTableOffset uint32
}

func NewInodeManager(file *os.File, inodeSize, inodeTableOffset uint32) *InodeManager {
	return &InodeManager{file, inodeSize, inodeTableOffset}
}

func (im InodeManager) ReadInode(inodeIndex uint32) (*inode.Inode, error) {
	offset := im.inodeTableOffset + im.inodeSize*inodeIndex
	return inode.ReadInodeAt(im.file, offset)
}

func (im InodeManager) SaveInode(value *inode.Inode, inodeIndex uint32) error {
	offset := im.inodeTableOffset + im.inodeSize*inodeIndex
	return value.WriteAt(im.file, offset)
}

func (im InodeManager) ReserveInodeTableSpace(inodeCount uint32) error {
	data := make([]byte, inodeCount*im.inodeSize)

	_, err := im.file.WriteAt(data, int64(im.inodeTableOffset))
	if err != nil {
		return err
	}

	return nil
}

func (im InodeManager) ResetInode(inodeIndex uint32) error {
	data := make([]byte, im.inodeSize)
	offset := im.inodeTableOffset + inodeIndex*im.inodeSize

	_, err := im.file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}
