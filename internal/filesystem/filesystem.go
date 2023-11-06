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

	superblock.WriteSuperBlockToFile(file, 0, *fileSystem.Superblock)
	fileSystem.BlockBitmap.WriteToFile(file, fileSystem.GetBlockBitmapOffset())
	fileSystem.InodeBitmap.WriteToFile(file, fileSystem.GetInodeBitmapOffset())

	inodeTableSize := fileSystem.Superblock.InodeCount * uint32(inode.GetInodeSize())
	ReserveSpaceInFile(file, fileSystem.GetInodeTableOffset(), inodeTableSize+uint32(sizeInBytes))

	fileSystem.CreateRootDirectory(file)

	return &fileSystem, nil
}

func ReserveSpaceInFile(file *os.File, offset int, size uint32) error {
	data := make([]byte, size)

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (fs FileSystem) CreateRootDirectory(file *os.File) {
	fs.InodeBitmap.SetBit(0, 1)
	fs.BlockBitmap.SetBit(0, 1)

	fs.BlockBitmap.WriteToFile(file, fs.GetBlockBitmapOffset())
	fs.InodeBitmap.WriteToFile(file, fs.GetInodeBitmapOffset())

	rootInode := inode.NewInode(false, 000, 0, 0, 1024)
	rootInode.WriteToFile(file, fs.GetInodeTableOffset(), 0)
}

func (fs FileSystem) GetBlockBitmapOffset() int {
	return fs.Superblock.Size()
}

func (fs FileSystem) GetInodeBitmapOffset() int {
	return fs.GetBlockBitmapOffset() + int(fs.BlockBitmap.Size)
}

func (fs FileSystem) GetInodeTableOffset() int {
	return fs.GetInodeBitmapOffset() + int(fs.InodeBitmap.Size)
}
