package filesystem

import (
	"file-system/internal/errs"
	"file-system/internal/filesystem/bitmap"
	"file-system/internal/filesystem/inode"
	"file-system/internal/filesystem/managers/blockmanager"
	"file-system/internal/filesystem/managers/directorymanager"
	"file-system/internal/filesystem/managers/inodemanager"
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
	dataFile         *os.File
	superblock       *superblock.Superblock
	blockBitmap      *bitmap.Bitmap
	inodeBitmap      *bitmap.Bitmap
	currentUser      *user.User
	nextUserId       uint16
	inodeManager     *inodemanager.InodeManager
	blockManager     *blockmanager.BlockManager
	directoryManager *directorymanager.DirectoryManager
}

func OpenFilesystem() (*FileSystem, error) {
	fs := FileSystem{}

	var err error
	fs.dataFile, err = os.OpenFile(FilesystemConfig.FileName, os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}

	fs.superblock, err = superblock.ReadSuperblockAt(fs.dataFile, 0)
	if err != nil {
		return nil, err
	}

	blockBitmapOffset := fs.superblock.Size()
	fs.blockBitmap, err = bitmap.ReadBitmapAt(
		fs.dataFile,
		blockBitmapOffset,
		fs.superblock.BlockCount,
	)
	if err != nil {
		return nil, err
	}

	inodeBitmapOffset := blockBitmapOffset + fs.blockBitmap.Size()
	fs.inodeBitmap, err = bitmap.ReadBitmapAt(
		fs.dataFile,
		inodeBitmapOffset,
		fs.superblock.InodeCount,
	)
	if err != nil {
		return nil, err
	}

	fs.InitializeManagers()

	fs.directoryManager.CurrentInode, err = fs.inodeManager.ReadInode(0)
	if err != nil {
		return nil, err
	}

	if err = fs.directoryManager.OpenDirectory(fs.directoryManager.CurrentInode.Blocks[0], "/"); err != nil {
		return nil, err
	}

	fs.ChangeUser("root", "root")

	return &fs, nil
}

func FormatFilesystem(sizeInBytes uint32, blockSize uint32) (*FileSystem, error) {
	fs := FileSystem{}

	var err error
	fs.dataFile, err = os.Create(FilesystemConfig.FileName)
	if err != nil {
		return nil, err
	}

	fs.superblock = superblock.NewSuperblock(sizeInBytes, blockSize, fs.dataFile)
	blockBitmapOffset := fs.superblock.Size()
	fs.blockBitmap = bitmap.NewBitmap(fs.superblock.BlockCount, fs.dataFile, blockBitmapOffset)
	inodeBitmapOffset := blockBitmapOffset + fs.blockBitmap.Size()
	fs.inodeBitmap = bitmap.NewBitmap(fs.superblock.InodeCount, fs.dataFile, inodeBitmapOffset)

	fs.superblock.Save()
	fs.blockBitmap.Save()
	fs.inodeBitmap.Save()

	fs.InitializeManagers()

	fs.inodeManager.ReserveInodeTableSpace(fs.superblock.InodeCount)
	fs.blockManager.ReserveBlocksSpace(fs.superblock.BlockCount)

	fs.directoryManager.Path = "/"
	if err := fs.CreateDirectory("/"); err != nil {
		return nil, err
	}
	if err := fs.CreateDirectory(".users"); err != nil {
		return nil, err
	}
	if err := fs.AddUser("root", "root", false); err != nil {
		return nil, err
	}
	if err := fs.ChangeUser("root", "root"); err != nil {
		return nil, err
	}

	return &fs, nil
}

func (fs *FileSystem) InitializeManagers() {
	inodeTableOffset := fs.superblock.Size() + fs.blockBitmap.Size() + fs.inodeBitmap.Size()
	blocksOffset := inodeTableOffset + fs.superblock.InodeCount*fs.superblock.InodeSize

	fs.inodeManager = inodemanager.NewInodeManager(fs.dataFile, fs.superblock.InodeSize, inodeTableOffset)
	fs.blockManager = blockmanager.NewBlockManager(fs.dataFile, fs.superblock.BlockSize, blocksOffset)
	fs.directoryManager = directorymanager.NewDirectoryManager(fs.dataFile, fs.superblock.BlockSize, blocksOffset)
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
	currDir := fs.directoryManager.Current
	currDirInode := fs.directoryManager.CurrentInode
	currPath := fs.directoryManager.Path

	pathToFolder, fileName := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(fileName)
	if err != nil {
		return err
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
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

	fs.inodeManager.SaveInode(fileInode, inodeIndex)

	fs.directoryManager.Current = currDir
	fs.directoryManager.CurrentInode = currDirInode
	fs.directoryManager.Path = currPath

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
	currDir := fs.directoryManager.Current
	currDirInode := fs.directoryManager.CurrentInode
	currPath := fs.directoryManager.Path

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	if path != "/" {
		if _, err := fs.directoryManager.Current.GetInode(name); err == nil {
			return fmt.Errorf("%w - %s", errs.ErrRecordAlreadyExists, name)
		}

		if fs.currentUser != nil && !fs.directoryManager.CurrentInode.HasWritePermission(*fs.currentUser) {
			return fmt.Errorf("%w - %s", errs.ErrPermissionDenied, name)
		}
	}

	blockIndex, err := fs.blockBitmap.TakeFreeBit()
	if err != nil {
		return err
	}
	fs.superblock.FreeBlockCount--

	inodeIndex, err := fs.inodeBitmap.TakeFreeBit()
	if err != nil {
		return err
	}
	fs.superblock.FreeInodeCount--

	fs.superblock.Save()
	fs.blockBitmap.Save()
	fs.inodeBitmap.Save()

	var userId uint16
	if fs.currentUser != nil {
		userId = fs.currentUser.UserId
	}

	fileInode, err := inode.NewInode(isFile, 64, userId, []uint32{blockIndex})
	if err != nil {
		return err
	}

	fs.inodeManager.SaveInode(fileInode, inodeIndex)

	if isFile {
		if len(content) > 0 {
			err := fs.blockManager.WriteBlock(blockIndex, content)
			if err != nil {
				return err
			}
		}
	} else {
		newDir, _ := fs.directoryManager.CreateNewDirectory(inodeIndex, blockIndex, path)
		if path == "/" {
			fs.directoryManager.Current = newDir
			fs.directoryManager.CurrentInode, _ = fs.inodeManager.ReadInode(inodeIndex)
		}
	}

	if path != "/" {
		fs.directoryManager.Current.AddFile(inodeIndex, name)
		fs.directoryManager.SaveCurrentDirectory()

		fs.directoryManager.Current = currDir
		fs.directoryManager.CurrentInode = currDirInode
		fs.directoryManager.Path = currPath
	}

	return nil
}

func (fs *FileSystem) DeleteFile(path string) error {
	currDir := fs.directoryManager.Current
	currDirInode := fs.directoryManager.CurrentInode
	currPath := fs.directoryManager.Path

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	if name == "." || name == ".." {
		return fmt.Errorf("%w - %s", errs.ErrIllegalArgument, name)
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(name)
	if err != nil {
		return err
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
	if err != nil {
		return err
	}

	if !fileInode.HasWritePermission(*fs.currentUser) {
		return fmt.Errorf("%w - %s", errs.ErrPermissionDenied, name)
	}

	if !fileInode.IsFile() {
		fs.ChangeDirectory(name)
		for _, name := range fs.directoryManager.Current.GetRecords() {
			if name == "." || name == ".." {
				continue
			}
			err := fs.DeleteFile(name)
			if err != nil {
				return err
			}
		}
		fs.ChangeDirectory("..")
	}

	fs.directoryManager.Current.DeleteFile(name)

	fs.blockBitmap.SetBit(fileInode.Blocks[0], 0)
	fs.inodeBitmap.SetBit(inodeIndex, 0)

	fs.superblock.FreeBlockCount++
	fs.superblock.FreeInodeCount++

	fs.blockManager.ResetBlock(fileInode.Blocks[0])
	fs.blockManager.ResetBlock(fs.directoryManager.CurrentInode.Blocks[0])
	fs.inodeManager.ResetInode(inodeIndex)

	fs.directoryManager.SaveCurrentDirectory()

	fs.blockBitmap.Save()
	fs.inodeBitmap.Save()
	fs.superblock.Save()

	if pathToFolder != "" {
		fs.directoryManager.Current = currDir
		fs.directoryManager.CurrentInode = currDirInode
		fs.directoryManager.Path = currPath
	}

	return nil
}

func (fs *FileSystem) ChangeDirectory(path string) error {
	path = strings.TrimSuffix(path, "/")
	dirs := strings.Split(path, "/")

	currDir := fs.directoryManager.Current
	currDirInode := fs.directoryManager.CurrentInode
	currPath := fs.directoryManager.Path

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

		dirInode, err := fs.inodeManager.ReadInode(inodeIndex)
		if err != nil {
			return err
		}

		if fs.currentUser != nil && !dirInode.HasReadPermission(*fs.currentUser) {
			return fmt.Errorf("%w - cd %s", errs.ErrPermissionDenied, dirName)
		}

		if dirInode.IsFile() {
			return fmt.Errorf("%w - %s", errs.ErrRecordIsNotDirectory, dirName)
		}

		dir, err := fs.directoryManager.ReadDirectory(dirInode.Blocks[0])
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

	fs.directoryManager.Current = currDir
	fs.directoryManager.CurrentInode = currDirInode
	fs.directoryManager.Path = currPath

	return nil
}

func (fs FileSystem) GetCurrentDirectoryRecords(long bool) []string {
	if !long {
		return fs.directoryManager.Current.GetRecords()
	}

	recordNames := fs.directoryManager.Current.GetRecords()

	result := make([]string, len(recordNames))
	for i, name := range recordNames {
		recordInodeIndex, _ := fs.directoryManager.Current.GetInode(name)
		recordInode, _ := fs.inodeManager.ReadInode(recordInodeIndex)
		tapString := recordInode.GetTypeAndPermissionString()
		fileSizeInBytes := recordInode.FileSize * fs.superblock.BlockSize
		modificationTime := time.Unix(int64(recordInode.ModificationTime), 0)
		modificationTimeString := modificationTime.Format("Jan 2 15:04")
		result[i] = fmt.Sprintf("%s %d %d %s %s", tapString, recordInode.UserId, fileSizeInBytes, modificationTimeString, name)
	}

	return result
}

func (fs FileSystem) ReadFile(path string) (string, error) {
	currDir := fs.directoryManager.Current
	currDirInode := fs.directoryManager.CurrentInode
	currPath := fs.directoryManager.Path

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(name)
	if err != nil {
		return "", err
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
	if err != nil {
		return "", err
	}

	if fs.currentUser != nil && !fileInode.HasReadPermission(*fs.currentUser) {
		return "", fmt.Errorf("%w - read %s", errs.ErrPermissionDenied, name)
	}

	str, err := fs.blockManager.ReadBlock(fileInode.Blocks[0], name)
	if err != nil {
		return "", err
	}

	fs.directoryManager.Current = currDir
	fs.directoryManager.CurrentInode = currDirInode
	fs.directoryManager.Path = currPath

	return str, nil
}

func (fs FileSystem) EditFile(path string, content string) error {
	currDir := fs.directoryManager.Current
	currDirInode := fs.directoryManager.CurrentInode
	currPath := fs.directoryManager.Path

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(name)
	if err != nil {
		return err
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
	if err != nil {
		return err
	}

	if !fileInode.HasWritePermission(*fs.currentUser) {
		return fmt.Errorf("%w - %s", errs.ErrPermissionDenied, name)
	}

	if !fileInode.IsFile() {
		return fmt.Errorf("%w - %s", errs.ErrRecordIsNotFile, name)
	}

	fs.blockManager.ResetBlock(fileInode.Blocks[0])
	fs.blockManager.WriteBlock(fileInode.Blocks[0], content)

	fileInode.ModificationTime = uint32(time.Now().Unix())
	fs.inodeManager.SaveInode(fileInode, inodeIndex)

	fs.directoryManager.Current = currDir
	fs.directoryManager.CurrentInode = currDirInode
	fs.directoryManager.Path = currPath

	return nil
}

func (fs *FileSystem) ChangePermissions(path string, value int) error {
	currDir := fs.directoryManager.Current
	currDirInode := fs.directoryManager.CurrentInode
	currPath := fs.directoryManager.Path

	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		fs.ChangeDirectory(pathToFolder)
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(name)
	if err != nil {
		return err
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
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

	fs.inodeManager.SaveInode(fileInode, inodeIndex)

	fs.directoryManager.Current = currDir
	fs.directoryManager.CurrentInode = currDirInode
	fs.directoryManager.Path = currPath

	return nil
}

func (fs FileSystem) GetCurrentPath() string {
	return fs.directoryManager.Path
}

func (fs FileSystem) GetCurrentUserName() string {
	return fs.currentUser.Username
}

func (fs *FileSystem) CloseDataFile() error {
	return fs.dataFile.Close()
}
