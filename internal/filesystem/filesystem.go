package filesystem

import (
	"bytes"
	"file-system/internal/filesystem/bitmap"
	"file-system/internal/filesystem/directory"
	"file-system/internal/filesystem/inode"
	"file-system/internal/filesystem/superblock"
	"file-system/internal/utils"
	"fmt"
	"os"
	"strings"
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
	currentPath           string
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

	dirOffset := fileSystem.GetDataBlocksOffset() + fileSystem.currentDirectoryInode.Blocks[0]*fileSystem.Superblock.BlockSize
	fileSystem.currentDirectory, err = directory.ReadDirectoryAt(fileSystem.dataFile, dirOffset)
	if err != nil {
		return nil, err
	}

	fileSystem.currentPath = "/"

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

	fileSystem.CreateDirectory("/")
	fileSystem.currentPath = "/"

	return &fileSystem, nil
}

func (fs *FileSystem) ExecuteCommand(command string, args []string) error {
	switch command {
	case "create":
		if len(args) < 1 {
			return fmt.Errorf("missing arguments - %s", command)
		}
		fileName := args[0]

		slashCount := strings.Count(fileName, "/")
		if slashCount > 0 {
			if slashCount == 1 && strings.HasSuffix(fileName, "/") {
				return fs.CreateDirectory(fileName[:len(fileName)-1])
			}
			return fmt.Errorf("incorrect file name - %s", fileName)
		}

		if strings.HasSuffix(fileName, ".") {
			return fmt.Errorf("incorrect file name - %s", fileName)
		}

		fileContent := ""
		if len(args) > 1 {
			fileContent = args[1]
		}
		return fs.CreateFile(fileName, fileContent)
	case "edit":
		if len(args) < 1 {
			return fmt.Errorf("missing arguments - %s", command)
		}
		fileName := args[0]
		fileContent := ""
		if len(args) > 1 {
			fileContent = args[1]
		}
		return fs.EditFile(fileName, fileContent)
	case "read":
		if len(args) < 1 {
			return fmt.Errorf("missing arguments - %s", command)
		}
		fileName := args[0]
		content, err := fs.ReadFile(fileName)
		if err != nil {
			return err
		}
		fmt.Println(content)
		return nil
	case "delete":
		if len(args) < 1 {
			return fmt.Errorf("missing arguments - %s", command)
		}
		fileName := args[0]
		return fs.DeleteFile(fileName, fs.currentDirectory, fs.currentDirectoryInode)
	case "list":
		for _, name := range fs.currentDirectory.GetRecords() {
			fmt.Println(name)
		}
		return nil
	case "cd":
		if len(args) < 1 {
			return fmt.Errorf("missing arguments - %s", command)
		}
		return fs.ChangeDirectory(args[0])
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

func (fs *FileSystem) CreateFile(name string, content string) error {
	if _, err := fs.currentDirectory.GetInode(name); err == nil {
		return fmt.Errorf("record with name \"%s\" already exists", name)
	}

	blockIndex, inodeIndex, err := fs.CreateFileOrDirectory(true)
	if err != nil {
		return err
	}

	fs.currentDirectory.AddFile(inodeIndex, name)
	offset := fs.GetDataBlocksOffset() + fs.currentDirectoryInode.Blocks[0]*fs.Superblock.BlockSize
	fs.currentDirectory.WriteAt(fs.dataFile, offset)

	if len(content) > 0 {
		data := utils.StringToByteBlock(content, fs.Superblock.BlockSize)
		offset := fs.GetDataBlocksOffset() + blockIndex*fs.Superblock.BlockSize

		_, err := fs.dataFile.WriteAt(data, int64(offset))
		if err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) CreateDirectory(name string) error {
	if name != "/" {
		if _, err := fs.currentDirectory.GetInode(name); err == nil {
			return fmt.Errorf("record with name \"%s\" already exists", name)
		}
	}

	blockIndex, inodeIndex, err := fs.CreateFileOrDirectory(false)
	if err != nil {
		return err
	}

	var currDirInodeIndex uint32
	if name != "/" {
		currDirInodeIndex, err = fs.currentDirectory.GetInode(".")
		if err != nil {
			return err
		}
	}

	newDir := directory.CreateNewDirectory(inodeIndex, currDirInodeIndex)
	newDir.WriteAt(fs.dataFile, fs.GetDataBlocksOffset()+blockIndex*fs.Superblock.BlockSize)

	if name == "/" {
		fs.currentDirectory = newDir
		inodeOffset := fs.GetInodeTableOffset() + fs.Superblock.InodeSize*inodeIndex
		fs.currentDirectoryInode, err = inode.ReadInodeAt(fs.dataFile, inodeOffset)
		if err != nil {
			return err
		}
	} else {
		fs.currentDirectory.AddFile(inodeIndex, name)
		fs.currentDirectory.WriteAt(fs.dataFile, fs.GetDataBlocksOffset()+fs.currentDirectoryInode.Blocks[0]*fs.Superblock.BlockSize)
	}

	return nil
}

func (fs *FileSystem) CreateFileOrDirectory(isFile bool) (uint32, uint32, error) {
	blockIndex, err := fs.BlockBitmap.TakeFreeBit()
	if err != nil {
		return 0, 0, err
	}
	fs.Superblock.FreeBlockCount--

	inodeIndex, err := fs.InodeBitmap.TakeFreeBit()
	if err != nil {
		return 0, 0, err
	}
	fs.Superblock.FreeInodeCount--

	fs.Superblock.WriteAt(fs.dataFile, 0)
	fs.BlockBitmap.WriteAt(fs.dataFile, fs.GetBlockBitmapOffset())
	fs.InodeBitmap.WriteAt(fs.dataFile, fs.GetInodeBitmapOffset())

	fileInode := inode.NewInode(isFile, 777, 0, 0, []uint32{blockIndex})
	inodeOffset := fs.GetInodeTableOffset() + fs.Superblock.InodeSize*inodeIndex
	fileInode.WriteAt(fs.dataFile, inodeOffset)

	return blockIndex, inodeIndex, nil
}

func (fs *FileSystem) DeleteFile(name string, fromDirectory *directory.Directory, fromInode *inode.Inode) error {
	if name == "." || name == ".." {
		return fmt.Errorf("illegal argument - %s", name)
	}

	inodeIndex, err := fromDirectory.GetInode(name)
	if err != nil {
		return err
	}

	offset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
	fileInode, err := inode.ReadInodeAt(fs.dataFile, offset)
	if err != nil {
		return err
	}

	if !inode.UnpackTypeAndPermissions(fileInode.TypeAndPermissions).IsFile {
		dirOffset := fs.GetDataBlocksOffset() + fileInode.Blocks[0]*fs.Superblock.BlockSize
		dir, err := directory.ReadDirectoryAt(fs.dataFile, dirOffset)
		if err != nil {
			return err
		}
		for _, name := range dir.GetRecords() {
			if name == "." || name == ".." {
				continue
			}
			err := fs.DeleteFile(name, dir, fileInode)
			if err != nil {
				return err
			}
		}
	}

	fromDirectory.DeleteFile(name)

	err = fs.BlockBitmap.SetBit(fileInode.Blocks[0], 0)
	if err != nil {
		return err
	}
	err = fs.InodeBitmap.SetBit(inodeIndex, 0)
	if err != nil {
		return err
	}
	fs.Superblock.FreeBlockCount++
	fs.Superblock.FreeInodeCount++

	offset = fs.GetDataBlocksOffset() + fileInode.Blocks[0]*fs.Superblock.BlockSize
	fs.ReserveSpaceInFile(offset, fs.Superblock.BlockSize)

	offset = fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
	fs.ReserveSpaceInFile(offset, fs.Superblock.InodeSize)

	offset = fs.GetDataBlocksOffset() + fromInode.Blocks[0]*fs.Superblock.BlockSize
	fromDirectory.WriteAt(fs.dataFile, offset)

	fs.BlockBitmap.WriteAt(fs.dataFile, fs.GetBlockBitmapOffset())
	fs.InodeBitmap.WriteAt(fs.dataFile, fs.GetInodeBitmapOffset())
	fs.Superblock.WriteAt(fs.dataFile, 0)

	return nil
}

func (fs *FileSystem) ChangeDirectory(path string) error {
	dirs := strings.Split(path, "/")

	currDir := fs.currentDirectory
	currDirInode := fs.currentDirectoryInode
	currPath := fs.currentPath

	for _, dirName := range dirs {
		inodeIndex, err := currDir.GetInode(dirName)
		if err != nil {
			return err
		}

		inodeOffset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
		dirInode, err := inode.ReadInodeAt(fs.dataFile, inodeOffset)
		if err != nil {
			return err
		}

		if inode.UnpackTypeAndPermissions(dirInode.TypeAndPermissions).IsFile {
			return fmt.Errorf("record is not a directory - %s", dirName)
		}

		dirOffset := fs.GetDataBlocksOffset() + dirInode.Blocks[0]*fs.Superblock.BlockSize
		dir, err := directory.ReadDirectoryAt(fs.dataFile, dirOffset)
		if err != nil {
			return err
		}

		currDir = dir
		currDirInode = dirInode
		currPath = utils.ChangeDirectoryPath(currPath, dirName)
	}

	fs.currentDirectory = currDir
	fs.currentDirectoryInode = currDirInode
	fs.currentPath = currPath

	return nil
}

func (fs FileSystem) ReadFile(fileName string) (string, error) {
	inodeIndex, err := fs.currentDirectory.GetInode(fileName)
	if err != nil {
		return "", err
	}

	inodeOffset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
	fileInode, err := inode.ReadInodeAt(fs.dataFile, inodeOffset)
	if err != nil {
		return "", err
	}

	if !inode.UnpackTypeAndPermissions(fileInode.TypeAndPermissions).IsFile {
		return "", fmt.Errorf("record is not a file - %s", fileName)
	}

	blockOffset := fs.GetDataBlocksOffset() + fileInode.Blocks[0]*fs.Superblock.BlockSize

	data := make([]byte, fs.Superblock.BlockSize)
	_, err = fs.dataFile.ReadAt(data, int64(blockOffset))
	if err != nil {
		return "", err
	}

	nullIndex := bytes.Index(data, []byte{0})
	if nullIndex == -1 {
		return "", fmt.Errorf("null terminator not found in file - %s", fileName)
	}

	str := string(data[:nullIndex])
	return str, nil
}

func (fs FileSystem) EditFile(name string, content string) error {
	inodeIndex, err := fs.currentDirectory.GetInode(name)
	if err != nil {
		return err
	}

	inodeOffset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
	fileInode, err := inode.ReadInodeAt(fs.dataFile, inodeOffset)
	if err != nil {
		return err
	}

	if !inode.UnpackTypeAndPermissions(fileInode.TypeAndPermissions).IsFile {
		return fmt.Errorf("record is not a file - %s", name)
	}

	blockOffset := fs.GetDataBlocksOffset() + fileInode.Blocks[0]*fs.Superblock.BlockSize
	data := utils.StringToByteBlock(content, fs.Superblock.BlockSize)

	_, err = fs.dataFile.WriteAt(data, int64(blockOffset))
	if err != nil {
		return err
	}

	return nil
}

func (fs FileSystem) GetCurrentPath() string {
	return fs.currentPath
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
