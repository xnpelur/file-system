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
	"time"
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
	nextUserId            uint16
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
	fs.ChangeUser("root", "root")

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

	if err := fs.InitializeFileSystem(); err != nil {
		return nil, err
	}

	return &fs, nil
}

func (fs *FileSystem) InitializeFileSystem() error {
	if err := fs.CreateDirectory("/"); err != nil {
		return err
	}
	fs.currentPath = "/"

	if err := fs.CreateDirectory(".users"); err != nil {
		return err
	}

	if err := fs.AddUser("root", "root", false); err != nil {
		return err
	}
	return fs.ChangeUser("root", "root")
}

func (fs *FileSystem) AddUser(username, password string, withDirectory bool) error {
	newUser := user.NewUser(username, fs.nextUserId, password)
	fs.nextUserId++

	if err := fs.CreateFileWithContent(fmt.Sprintf("/.users/%s", username), newUser.GetUserString()); err != nil {
		return err
	}

	if withDirectory {
		userDirPath := fmt.Sprintf("/%s", username)

		if err := fs.CreateDirectory(userDirPath); err != nil {
			return err
		}

		if err := fs.ChangeOwner(userDirPath, username); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) ChangeUser(username, password string) error {
	content, err := fs.ReadFile(fmt.Sprintf("/.users/%s", username))
	if err != nil {
		return err
	}

	u, err := user.ReadUserFromString(content, password)
	if err != nil {
		return err
	}

	fs.ChangeDirectory("/")
	fs.currentUser = u

	if username != "root" {
		userDirPath := fmt.Sprintf("/%s", username)
		if err := fs.ChangeDirectory(userDirPath); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) ChangeOwner(path string, username string) error {
	currDir := fs.currentDirectory
	currDirInode := fs.currentDirectoryInode
	currPath := fs.currentPath

	pathToFolder, fileName := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	inodeIndex, err := fs.currentDirectory.GetInode(fileName)
	if err != nil {
		return err
	}

	offset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
	fileInode, err := inode.ReadInodeAt(fs.dataFile, offset)
	if err != nil {
		return err
	}

	content, err := fs.ReadFile(fmt.Sprintf("/.users/%s", username))
	if err != nil {
		return err
	}

	fileInode.UserId, err = user.GetUserIdFromString(content)
	if err != nil {
		return err
	}

	fileInode.WriteAt(fs.dataFile, offset)

	fs.currentDirectory = currDir
	fs.currentDirectoryInode = currDirInode
	fs.currentPath = currPath

	return nil
}

func (fs *FileSystem) ReserveSpaceInFile(offset uint32, size uint32) error {
	data := make([]byte, size)

	_, err := fs.dataFile.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) CreateEmptyFile(path string) error {
	return fs.CreateEntity(path, true, "")
}

func (fs *FileSystem) CreateFileWithContent(path, content string) error {
	return fs.CreateEntity(path, true, content)
}

func (fs *FileSystem) CreateDirectory(path string) error {
	return fs.CreateEntity(path, false, "")
}

func (fs *FileSystem) CreateEntity(path string, isFile bool, content string) error {
	currDir := fs.currentDirectory
	currDirInode := fs.currentDirectoryInode
	currPath := fs.currentPath

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	if path != "/" {
		if _, err := fs.currentDirectory.GetInode(name); err == nil {
			return fmt.Errorf("%w - %s", errs.ErrRecordAlreadyExists, name)
		}

		if fs.currentUser != nil && !fs.currentDirectoryInode.HasWritePermission(*fs.currentUser) {
			return fmt.Errorf("%w - %s", errs.ErrPermissionDenied, name)
		}
	}

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

	fs.Superblock.WriteAt(fs.dataFile, 0)
	fs.BlockBitmap.WriteAt(fs.dataFile, fs.GetBlockBitmapOffset())
	fs.InodeBitmap.WriteAt(fs.dataFile, fs.GetInodeBitmapOffset())

	var userId uint16
	if fs.currentUser != nil {
		userId = fs.currentUser.UserId
	}

	fileInode, err := inode.NewInode(isFile, 64, userId, []uint32{blockIndex})
	if err != nil {
		return err
	}

	inodeOffset := fs.GetInodeTableOffset() + fs.Superblock.InodeSize*inodeIndex
	fileInode.WriteAt(fs.dataFile, inodeOffset)

	if isFile {
		if len(content) > 0 {
			data := utils.StringToByteBlock(content, fs.Superblock.BlockSize)
			offset := fs.GetDataBlocksOffset() + blockIndex*fs.Superblock.BlockSize

			_, err := fs.dataFile.WriteAt(data, int64(offset))
			if err != nil {
				return err
			}
		}
	} else {
		var currDirInodeIndex uint32
		if path != "/" {
			currDirInodeIndex, err = fs.currentDirectory.GetInode(".")
			if err != nil {
				return err
			}
		}
		newDir := directory.NewDirectory(inodeIndex, currDirInodeIndex)
		newDir.WriteAt(fs.dataFile, fs.GetDataBlocksOffset()+blockIndex*fs.Superblock.BlockSize)
		if path == "/" {
			fs.currentDirectory = newDir
			inodeOffset := fs.GetInodeTableOffset() + fs.Superblock.InodeSize*inodeIndex
			fs.currentDirectoryInode, err = inode.ReadInodeAt(fs.dataFile, inodeOffset)
			if err != nil {
				return err
			}
		}
	}

	if path != "/" {
		fs.currentDirectory.AddFile(inodeIndex, name)
		offset := fs.GetDataBlocksOffset() + fs.currentDirectoryInode.Blocks[0]*fs.Superblock.BlockSize
		fs.currentDirectory.WriteAt(fs.dataFile, offset)

		fs.currentDirectory = currDir
		fs.currentDirectoryInode = currDirInode
		fs.currentPath = currPath
	}

	return nil
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

	if !fileInode.IsFile() {
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
	path = strings.TrimSuffix(path, "/")
	dirs := strings.Split(path, "/")

	currDir := fs.currentDirectory
	currDirInode := fs.currentDirectoryInode
	currPath := fs.currentPath

	for i, dirName := range dirs {
		var inodeIndex uint32
		var err error

		if dirName != "" {
			inodeIndex, err = currDir.GetInode(dirName)
			if err != nil {
				return err
			}
		} else if i != 0 {
			return fmt.Errorf("incorrect path - %s", path)
		}

		inodeOffset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
		dirInode, err := inode.ReadInodeAt(fs.dataFile, inodeOffset)
		if err != nil {
			return err
		}

		if fs.currentUser != nil && !dirInode.HasReadPermission(*fs.currentUser) {
			return fmt.Errorf("%w - cd %s", errs.ErrPermissionDenied, dirName)
		}

		if dirInode.IsFile() {
			return fmt.Errorf("%w - %s", errs.ErrRecordIsNotDirectory, dirName)
		}

		dirOffset := fs.GetDataBlocksOffset() + dirInode.Blocks[0]*fs.Superblock.BlockSize
		dir, err := directory.ReadDirectoryAt(fs.dataFile, dirOffset)
		if err != nil {
			return err
		}

		currDir = dir
		currDirInode = dirInode

		if dirName == "" {
			dirName = "/"
		}
		currPath = utils.ChangeDirectoryPath(currPath, dirName)
	}

	fs.currentDirectory = currDir
	fs.currentDirectoryInode = currDirInode
	fs.currentPath = currPath

	return nil
}

func (fs FileSystem) GetCurrentDirectoryRecords(long bool) []string {
	if !long {
		return fs.currentDirectory.GetRecords()
	}

	recordNames := fs.currentDirectory.GetRecords()

	result := make([]string, len(recordNames))
	for i, name := range recordNames {
		recordInodeIndex, _ := fs.currentDirectory.GetInode(name)
		offset := fs.GetInodeTableOffset() + recordInodeIndex*fs.Superblock.InodeSize
		recordInode, _ := inode.ReadInodeAt(fs.dataFile, offset)
		tapString := recordInode.GetTypeAndPermissionString()
		fileSizeInBytes := recordInode.FileSize * fs.Superblock.BlockSize
		modificationTime := time.Unix(int64(recordInode.ModificationTime), 0)
		modificationTimeString := modificationTime.Format("Jan 2 15:04")
		result[i] = fmt.Sprintf("%s %d %d %s %s", tapString, recordInode.UserId, fileSizeInBytes, modificationTimeString, name)
	}

	return result
}

func (fs FileSystem) ReadFile(path string) (string, error) {
	currDir := fs.currentDirectory
	currDirInode := fs.currentDirectoryInode
	currPath := fs.currentPath

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	inodeIndex, err := fs.currentDirectory.GetInode(name)
	if err != nil {
		return "", err
	}

	inodeOffset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
	fileInode, err := inode.ReadInodeAt(fs.dataFile, inodeOffset)
	if err != nil {
		return "", err
	}

	if fs.currentUser != nil && !fileInode.HasReadPermission(*fs.currentUser) {
		return "", fmt.Errorf("%w - read %s", errs.ErrPermissionDenied, name)
	}

	blockOffset := fs.GetDataBlocksOffset() + fileInode.Blocks[0]*fs.Superblock.BlockSize

	data := make([]byte, fs.Superblock.BlockSize)
	_, err = fs.dataFile.ReadAt(data, int64(blockOffset))
	if err != nil {
		return "", err
	}

	nullIndex := bytes.Index(data, []byte{0})
	if nullIndex == -1 {
		return "", fmt.Errorf("%w - %s", errs.ErrNullNotFound, name)
	}

	str := string(data[:nullIndex])

	fs.currentDirectory = currDir
	fs.currentDirectoryInode = currDirInode
	fs.currentPath = currPath

	return str, nil
}

func (fs FileSystem) EditFile(path string, content string) error {
	currDir := fs.currentDirectory
	currDirInode := fs.currentDirectoryInode
	currPath := fs.currentPath

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	inodeIndex, err := fs.currentDirectory.GetInode(name)
	if err != nil {
		return err
	}

	inodeOffset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
	fileInode, err := inode.ReadInodeAt(fs.dataFile, inodeOffset)
	if err != nil {
		return err
	}

	if !fileInode.HasWritePermission(*fs.currentUser) {
		return fmt.Errorf("%w - %s", errs.ErrPermissionDenied, name)
	}

	if !fileInode.IsFile() {
		return fmt.Errorf("%w - %s", errs.ErrRecordIsNotFile, name)
	}

	blockOffset := fs.GetDataBlocksOffset() + fileInode.Blocks[0]*fs.Superblock.BlockSize
	data := utils.StringToByteBlock(content, fs.Superblock.BlockSize)

	_, err = fs.dataFile.WriteAt(data, int64(blockOffset))
	if err != nil {
		return err
	}

	fileInode.ModificationTime = uint32(time.Now().Unix())
	err = fileInode.WriteAt(fs.dataFile, inodeOffset)
	if err != nil {
		return err
	}

	fs.currentDirectory = currDir
	fs.currentDirectoryInode = currDirInode
	fs.currentPath = currPath

	return nil
}

func (fs *FileSystem) ChangePermissions(path string, value int) error {
	currDir := fs.currentDirectory
	currDirInode := fs.currentDirectoryInode
	currPath := fs.currentPath

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	inodeIndex, err := fs.currentDirectory.GetInode(name)
	if err != nil {
		return err
	}

	inodeOffset := fs.GetInodeTableOffset() + inodeIndex*fs.Superblock.InodeSize
	fileInode, err := inode.ReadInodeAt(fs.dataFile, inodeOffset)
	if err != nil {
		return err
	}

	if fs.currentUser.UserId != fileInode.UserId {
		return fmt.Errorf("%w - chmod %s", errs.ErrPermissionDenied, name)
	}

	err = fileInode.ChangePermissions(value)
	if err != nil {
		return err
	}

	err = fileInode.WriteAt(fs.dataFile, inodeOffset)
	if err != nil {
		return err
	}

	fs.currentDirectory = currDir
	fs.currentDirectoryInode = currDirInode
	fs.currentPath = currPath

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
