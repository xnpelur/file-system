package filesystem

import (
	"file-system/internal/filesystem/bitmap"
	"file-system/internal/filesystem/inode"
	"file-system/internal/filesystem/superblock"
	"os"
)

func FormatFilesystem(sizeInBytes int, blockSize int) error {
	file, err := os.Create("filesystem.data")
	if err != nil {
		return err
	}
	defer file.Close()

	superblockInstance := superblock.NewSuperblock(uint32(sizeInBytes), uint32(blockSize))

	offset := 0

	superblock.WriteSuperBlockToFile(file, offset, superblockInstance)
	offset += superblockInstance.Size()

	blockBitmap := bitmap.NewBitmap(superblockInstance.BlockCount)
	blockBitmap.SetBit(0, 1)

	bitmap.WriteBitmapToFile(file, offset, *blockBitmap)
	offset += int(blockBitmap.Size)

	inodeBitmap := bitmap.NewBitmap(superblockInstance.InodeCount)
	inodeBitmap.SetBit(0, 1)

	bitmap.WriteBitmapToFile(file, offset, *inodeBitmap)
	offset += int(inodeBitmap.Size)

	inodeTableSize, _ := inode.WriteInodeTable(file, offset, int(superblockInstance.InodeCount))
	offset += int(inodeTableSize)

	return nil
}
