package blockmanager

import (
	"bytes"
	"file-system/internal/filesystem/inode"
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

func (bm BlockManager) ReadBlocks(fileInode *inode.Inode, name string) (string, error) {
	data := make([]byte, 0)
	lastBlockEnd := bm.blockSize

	for i := 0; i < int(fileInode.FileSize); i++ {
		blockIndex := fileInode.Blocks[i]
		offset := bm.blocksOffset + blockIndex*bm.blockSize

		tmpData := make([]byte, bm.blockSize)
		_, err := bm.file.ReadAt(tmpData, int64(offset))
		if err != nil {
			return "", err
		}

		data = append(data, tmpData...)

		contentEnd := bytes.Index(tmpData, []byte{0})
		if contentEnd != -1 {
			lastBlockEnd = uint32(contentEnd)
		}
	}

	end := bm.blockSize*(fileInode.FileSize-1) + lastBlockEnd
	return string(data[:end]), nil
}

func (bm BlockManager) WriteBlocks(fileInode *inode.Inode, content string) error {
	for i := 0; i < int(fileInode.FileSize); i++ {
		blockIndex := fileInode.Blocks[i]
		offset := bm.blocksOffset + blockIndex*bm.blockSize

		sliceStart := int(bm.blockSize) * i
		sliceEnd := int(bm.blockSize) * (i + 1)
		if sliceEnd > len(content) {
			sliceEnd = len(content)
		}
		tmpData := utils.StringToByteBlock(content[sliceStart:sliceEnd], bm.blockSize)

		_, err := bm.file.WriteAt(tmpData, int64(offset))
		if err != nil {
			return err
		}
	}
	return nil
}

func (bm BlockManager) ReserveBlocksSpace(blockCount uint32) error {
	data := make([]byte, blockCount*bm.blockSize)

	_, err := bm.file.WriteAt(data, int64(bm.blocksOffset))
	if err != nil {
		return err
	}

	return nil
}

func (bm BlockManager) ResetBlocks(fileInode *inode.Inode) error {
	for i := 0; i < int(fileInode.FileSize); i++ {
		blockIndex := fileInode.Blocks[i]
		data := make([]byte, bm.blockSize)
		offset := bm.blocksOffset + blockIndex*bm.blockSize

		_, err := bm.file.WriteAt(data, int64(offset))
		if err != nil {
			return err
		}
	}

	return nil
}
