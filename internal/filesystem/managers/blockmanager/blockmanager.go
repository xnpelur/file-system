package blockmanager

import (
	"bytes"
	"file-system/internal/utils"
	"os"
)

type BlockManager struct {
	file         *os.File
	blockSize    uint32
	blocksOffset uint32
}

func NewBlockManager(file *os.File, blockSize, blocksOffset uint32) *BlockManager {
	return &BlockManager{file, blockSize, blocksOffset}
}

func (bm BlockManager) ReadBlock(blockIndex uint32, name string) (string, error) {
	offset := bm.blocksOffset + blockIndex*bm.blockSize

	data := make([]byte, bm.blockSize)
	_, err := bm.file.ReadAt(data, int64(offset))
	if err != nil {
		return "", err
	}

	contentEnd := bytes.Index(data, []byte{0})
	if contentEnd == -1 {
		contentEnd = int(bm.blockSize)
	}

	return string(data[:contentEnd]), nil
}

func (bm BlockManager) WriteBlock(blockIndex uint32, content string) error {
	data := utils.StringToByteBlock(content, bm.blockSize)
	offset := bm.blocksOffset + blockIndex*bm.blockSize

	_, err := bm.file.WriteAt(data, int64(offset))
	return err
}

// func (bm BlockManager) SaveInode(value *inode.Inode, inodeIndex uint32) error {
// 	offset := im.inodeTableOffset + im.inodeSize*inodeIndex
// 	return value.WriteAt(im.file, offset)
// }

func (bm BlockManager) ReserveBlocksSpace(blockCount uint32) error {
	data := make([]byte, blockCount*bm.blockSize)

	_, err := bm.file.WriteAt(data, int64(bm.blocksOffset))
	if err != nil {
		return err
	}

	return nil
}

func (bm BlockManager) ResetBlock(blockIndex uint32) error {
	data := make([]byte, bm.blockSize)
	offset := bm.blocksOffset + blockIndex*bm.blockSize

	_, err := bm.file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}
