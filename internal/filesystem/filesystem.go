package filesystem

import (
	"bytes"
	"file-system/internal/errs"
	"file-system/internal/filesystem/bitmap"
	"file-system/internal/filesystem/directory"
	"file-system/internal/filesystem/inode"
	"file-system/internal/filesystem/superblock"
	"file-system/internal/filesystem/user"
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
	currentUser           *user.User
	currentPath           string
}

func OpenFilesystem() (*FileSystem, error) {
	fs := FileSystem{}

	var err error
	fs.dataFile, err = os.OpenFile(FilesystemConfig.FileName, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	fs.Superblock, err = superblock.ReadSuperblockAt(fs.dataFile, 0)
	if err != nil {
		return nil, err
	}

	fs.BlockBitmap, err = bitmap.ReadBitmapAt(
		fs.dataFile,
		fs.GetBlockBitmapOffset(),
		fs.Superblock.BlockCount,
	)
	if err != nil {
		return nil, err
	}

	fs.InodeBitmap, err = bitmap.ReadBitmapAt(
		fs.dataFile,
		fs.GetInodeBitmapOffset(),
		fs.Superblock.InodeCount,
	)
	if err != nil {
		return nil, err
	}

	fs.currentDirectoryInode, err = inode.ReadInodeAt(fs.dataFile, fs.GetInodeTableOffset())
	if err != nil {
		return nil, err
	}

	dirOffset := fs.GetDataBlocksOffset() + fs.currentDirectoryInode.Blocks[0]*fs.Superblock.BlockSize
	fs.currentDirectory, err = directory.ReadDirectoryAt(fs.dataFile, dirOffset)
	if err != nil {
		return nil, err
	}

	fs.currentPath = "/"

	return &fs, nil
}

func FormatFilesystem(sizeInBytes uint32, blockSize uint32) (*FileSystem, error) {
	fs := FileSystem{}
	fs.Superblock = superblock.NewSuperblock(sizeInBytes, blockSize)
	fs.BlockBitmap = bitmap.NewBitmap(fs.Superblock.BlockCount)
	fs.InodeBitmap = bitmap.NewBitmap(fs.Superblock.InodeCount)

	var err error
	fs.dataFile, err = os.Create(FilesystemConfig.FileName)
	if err != nil {
		return nil, err
	}

	fs.Superblock.WriteAt(fs.dataFile, 0)
	fs.BlockBitmap.WriteAt(fs.dataFile, fs.GetBlockBitmapOffset())
	fs.InodeBitmap.WriteAt(fs.dataFile, fs.GetInodeBitmapOffset())

	inodeTableSize := fs.Superblock.InodeCount * fs.Superblock.InodeSize
	fs.ReserveSpaceInFile(fs.GetInodeTableOffset(), inodeTableSize+sizeInBytes)

	fs.InitializeFileSystem()

	return &fs, nil
}

func (fs *FileSystem) InitializeFileSystem() {
	fs.CreateDirectory("/")
	fs.currentPath = "/"

	fs.CreateDirectory(".users")
	fs.ChangeDirectory(".users")

	fs.currentUser = user.NewUser("root", "root")
	fs.CreateFile("root", fs.currentUser.GetUserString())

	fs.ChangeDirectory("..")
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
		return fmt.Errorf("%w - %s", errs.ErrRecordAlreadyExists, name)
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
			return fmt.Errorf("%w - %s", errs.ErrRecordAlreadyExists, name)
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
		return fmt.Errorf("%w - %s", errs.ErrIllegalArgument, name)
	}

	if fromDirectory == nil || fromInode == nil {
		fromDirectory = fs.currentDirectory
		fromInode = fs.currentDirectoryInode
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
	fs.ReserveSpaceInFile(offset, fs.Superblock.BlockSize)
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
			return fmt.Errorf("%w - %s", errs.ErrRecordIsNotDirectory, dirName)
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

func (fs FileSystem) GetCurrentDirectoryRecords() []string {
	return fs.currentDirectory.GetRecords()
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
		return "", fmt.Errorf("%w - %s", errs.ErrRecordIsNotFile, fileName)
	}

	blockOffset := fs.GetDataBlocksOffset() + fileInode.Blocks[0]*fs.Superblock.BlockSize

	data := make([]byte, fs.Superblock.BlockSize)
	_, err = fs.dataFile.ReadAt(data, int64(blockOffset))
	if err != nil {
		return "", err
	}

	nullIndex := bytes.Index(data, []byte{0})
	if nullIndex == -1 {
		return "", fmt.Errorf("%w - %s", errs.ErrNullNotFound, fileName)
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
		return fmt.Errorf("%w - %s", errs.ErrRecordIsNotFile, name)
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

func (fs FileSystem) GetCurrentUserName() string {
	return fs.currentUser.Username
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
