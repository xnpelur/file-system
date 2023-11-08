package filesystem

import (
	"file-system/internal/filesystem/bitmap"
	"file-system/internal/filesystem/directory"
	"file-system/internal/filesystem/inode"
	"file-system/internal/filesystem/superblock"
	"fmt"
	"os"
)

type Config struct {
	FileName string
}

var FilesystemConfig = Config{
	FileName: "filesystem.data",
}

type FileSystem struct {
	Superblock            *superblock.Superblock
	BlockBitmap           *bitmap.Bitmap
	InodeBitmap           *bitmap.Bitmap
	dataFile              *os.File
	currentDirectory      *directory.Directory
	currentDirectoryInode *inode.Inode
}

func OpenFilesystem() (*FileSystem, error) {
	fileSystem := FileSystem{}

	var err error
	fileSystem.dataFile, err = os.OpenFile(FilesystemConfig.FileName, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	fileSystem.Superblock, err = superblock.ReadSuperblockAt(fileSystem.dataFile, 0)
	if err != nil {
		return nil, err
	}

	fileSystem.BlockBitmap, err = bitmap.ReadBitmapAt(
		fileSystem.dataFile,
		fileSystem.GetBlockBitmapOffset(),
		fileSystem.Superblock.BlockCount,
	)
	if err != nil {
		return nil, err
	}

	fileSystem.InodeBitmap, err = bitmap.ReadBitmapAt(
		fileSystem.dataFile,
		fileSystem.GetInodeBitmapOffset(),
		fileSystem.Superblock.InodeCount,
	)
	if err != nil {
		return nil, err
	}

	fileSystem.currentDirectoryInode, err = inode.ReadInodeAt(fileSystem.dataFile, fileSystem.GetInodeTableOffset())
	if err != nil {
		return nil, err
	}

	dirOffset := fileSystem.GetDataBlocksOffset() + fileSystem.currentDirectoryInode.Blocks[0]
	fileSystem.currentDirectory, err = directory.ReadDirectoryAt(fileSystem.dataFile, dirOffset)
	if err != nil {
		return nil, err
	}

	return &fileSystem, nil
}

func FormatFilesystem(sizeInBytes uint32, blockSize uint32) (*FileSystem, error) {
	fileSystem := FileSystem{}
	fileSystem.Superblock = superblock.NewSuperblock(sizeInBytes, blockSize)
	fileSystem.BlockBitmap = bitmap.NewBitmap(fileSystem.Superblock.BlockCount)
	fileSystem.InodeBitmap = bitmap.NewBitmap(fileSystem.Superblock.InodeCount)

	var err error
	fileSystem.dataFile, err = os.Create(FilesystemConfig.FileName)
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

func (fs *FileSystem) ExecuteCommand(command string, args []string) error {
	switch command {
	case "create":
		if len(args) < 1 {
			return fmt.Errorf("missing arguments - %s", command)
		}
		fileName := args[0]
		fileContent := ""
		if len(args) > 1 {
			fileContent = args[1]
		}
		return fs.CreateFile(fileName, fileContent)
	case "list":
		fs.currentDirectory.ListRecords()
		return nil
	default:
		return fmt.Errorf("unknown command - %s", command)
	}
}

func (fs *FileSystem) ReserveSpaceInFile(offset uint32, size uint32) error {
	data := make([]byte, size)

	_, err := fs.dataFile.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) CreateRootDirectory() {
	fs.BlockBitmap.SetBit(0, 1)
	fs.InodeBitmap.SetBit(0, 1)

	fs.Superblock.FreeBlockCount--
	fs.Superblock.FreeInodeCount--

	fs.currentDirectoryInode = inode.NewInode(false, 000, 0, 0, []uint32{0})
	fs.currentDirectory = directory.CreateNewDirectory(0, 0)

	fs.Superblock.WriteAt(fs.dataFile, 0)
	fs.BlockBitmap.WriteAt(fs.dataFile, fs.GetBlockBitmapOffset())
	fs.InodeBitmap.WriteAt(fs.dataFile, fs.GetInodeBitmapOffset())
	fs.currentDirectoryInode.WriteAt(fs.dataFile, fs.GetInodeTableOffset())
	fs.currentDirectory.WriteAt(fs.dataFile, fs.GetDataBlocksOffset())
}

func (fs *FileSystem) CreateFile(name string, content string) error {
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
	fs.currentDirectory.WriteAt(fs.dataFile, fs.GetDataBlocksOffset()+fs.currentDirectoryInode.Blocks[0])

	if len(content) > 0 {
		data := []byte(content)
		offset := fs.GetDataBlocksOffset() + blockIndex*fs.Superblock.BlockSize

		_, err := fs.dataFile.WriteAt(data, int64(offset))
		if err != nil {
			return err
		}
	}

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

func (fs *FileSystem) CloseDataFile() error {
	return fs.dataFile.Close()
}
