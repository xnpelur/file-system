package filesystem

import (
	"file-system/internal/filesystem/bitmap"
	"file-system/internal/filesystem/inode"
	"file-system/internal/filesystem/superblock"
	"os"
)

type FileSystem struct {
	Superblock  *superblock.Superblock
	BlockBitmap *bitmap.Bitmap
	InodeBitmap *bitmap.Bitmap
}

func FormatFilesystem(sizeInBytes int, blockSize int) (*FileSystem, error) {
	file, err := os.Create("filesystem.data")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileSystem := FileSystem{}
	fileSystem.Superblock = superblock.NewSuperblock(uint32(sizeInBytes), uint32(blockSize))
	fileSystem.BlockBitmap = bitmap.NewBitmap(fileSystem.Superblock.BlockCount)
	fileSystem.InodeBitmap = bitmap.NewBitmap(fileSystem.Superblock.InodeCount)

	offset := 0

	superblock.WriteSuperBlockToFile(file, offset, *fileSystem.Superblock)
	offset += fileSystem.Superblock.Size()

	bitmap.WriteBitmapToFile(file, offset, *fileSystem.BlockBitmap)
	offset += int(fileSystem.BlockBitmap.Size)

	bitmap.WriteBitmapToFile(file, offset, *fileSystem.InodeBitmap)
	offset += int(fileSystem.InodeBitmap.Size)

	inodeTableSize, _ := inode.WriteInodeTable(file, offset, int(fileSystem.Superblock.InodeCount))
	offset += inodeTableSize

	WriteInfoBlock(file, offset, sizeInBytes)

	fileSystem.CreateRootDirectory()

	return &fileSystem, nil
}

func WriteInfoBlock(file *os.File, offset int, size int) error {
	data := make([]byte, size)

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (fs FileSystem) CreateRootDirectory() {
	fs.InodeBitmap.SetBit(0, 1)
	fs.BlockBitmap.SetBit(0, 1)
}
