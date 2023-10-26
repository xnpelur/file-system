package filesystem

import (
	"file-system/internal/filesystem/bitmap"
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

	err = superblock.WriteSuperBlockToFile(file, 0, superblockInstance)
	if err != nil {
		return err
	}

	blockBitmap := bitmap.NewBitmap(superblockInstance.BlockCount)

	err = bitmap.WriteBitmapToFile(file, superblockInstance.Size(), *blockBitmap)
	if err != nil {
		return err
	}

	inodeBitmap := bitmap.NewBitmap(superblockInstance.InodeCount)

	err = bitmap.WriteBitmapToFile(file, superblockInstance.Size()+int(blockBitmap.Size), *inodeBitmap)
	if err != nil {
		return err
	}

	return nil
}
