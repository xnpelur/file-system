package filesystem

import (
	"file-system/internal/filesystem/bitmap"
	"file-system/internal/filesystem/directory"
	"file-system/internal/filesystem/inode"
	"file-system/internal/filesystem/superblock"
	"os"
)

type FileSystem struct {
	Superblock       *superblock.Superblock
	BlockBitmap      *bitmap.Bitmap
	InodeBitmap      *bitmap.Bitmap
	dataFile         *os.File
	currentDirectory directory.Directory
}

func FormatFilesystem(sizeInBytes uint32, blockSize uint32) (*FileSystem, error) {
	fileSystem := FileSystem{}
	fileSystem.Superblock = superblock.NewSuperblock(sizeInBytes, blockSize)
	fileSystem.BlockBitmap = bitmap.NewBitmap(fileSystem.Superblock.BlockCount)
	fileSystem.InodeBitmap = bitmap.NewBitmap(fileSystem.Superblock.InodeCount)

	var err error
	fileSystem.dataFile, err = os.Create("filesystem.data")
	if err != nil {
		return nil, err
	}

	fileSystem.Superblock.WriteAt(fileSystem.dataFile, 0)
	fileSystem.BlockBitmap.WriteAt(fileSystem.dataFile, fileSystem.GetBlockBitmapOffset())
	fileSystem.InodeBitmap.WriteAt(fileSystem.dataFile, fileSystem.GetInodeBitmapOffset())

	inodeTableSize := fileSystem.Superblock.InodeCount * fileSystem.Superblock.InodeSize
	fileSystem.ReserveSpaceInFile(fileSystem.GetInodeTableOffset(), inodeTableSize+sizeInBytes)

	fileSystem.CreateRootDirectory()

	return &fileSystem, nil
}

func (fs FileSystem) ReserveSpaceInFile(offset uint32, size uint32) error {
	data := make([]byte, size)

	_, err := fs.dataFile.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (fs FileSystem) CreateRootDirectory() {
	fs.BlockBitmap.SetBit(0, 1)
	fs.InodeBitmap.SetBit(0, 1)

	fs.Superblock.FreeBlockCount--
	fs.Superblock.FreeInodeCount--

	rootInode := inode.NewInode(false, 000, 0, 0, []uint32{0})
	rootDir := directory.CreateNewDirectory(0, 0)

	fs.Superblock.WriteAt(fs.dataFile, 0)
	fs.BlockBitmap.WriteAt(fs.dataFile, fs.GetBlockBitmapOffset())
	fs.InodeBitmap.WriteAt(fs.dataFile, fs.GetInodeBitmapOffset())
	rootInode.WriteAt(fs.dataFile, fs.GetInodeTableOffset())
	rootDir.WriteAt(fs.dataFile, fs.GetDataBlocksOffset())

	fs.currentDirectory = rootDir
	fs.CreateFirstFile("hello.txt")
}

func (fs FileSystem) CreateFirstFile(name string) error {
	blockIndex, err := fs.BlockBitmap.TakeFreeBit()
	if err != nil {
		return err
	}
	fs.Superblock.FreeBlockCount--

	inodeIndex, err := fs.InodeBitmap.TakeFreeBit()
	if err != nil {
		return err
	}
	fs.Superblock.FreeInodeCount--

	fileInode := inode.NewInode(true, 777, 0, 0, []uint32{blockIndex})
	inodeOffset := fs.GetInodeTableOffset() + fs.Superblock.InodeSize*inodeIndex
	fileInode.WriteAt(fs.dataFile, inodeOffset)

	fs.currentDirectory.AddFile(inodeIndex, name)

	return nil
}

func (fs FileSystem) GetBlockBitmapOffset() uint32 {
	return fs.Superblock.Size()
}

func (fs FileSystem) GetInodeBitmapOffset() uint32 {
	return fs.GetBlockBitmapOffset() + fs.BlockBitmap.Size
}

func (fs FileSystem) GetInodeTableOffset() uint32 {
	return fs.GetInodeBitmapOffset() + fs.InodeBitmap.Size
}

func (fs FileSystem) GetDataBlocksOffset() uint32 {
	return fs.GetInodeTableOffset() + fs.Superblock.InodeCount*fs.Superblock.InodeSize
}

func (fs FileSystem) CloseDataFile() error {
	return fs.dataFile.Close()
}
