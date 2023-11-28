package filesystem

import (
	"file-system/internal/errs"
	"file-system/internal/filesystem/bitmap"
	"file-system/internal/filesystem/inode"
	"file-system/internal/filesystem/managers/blockmanager"
	"file-system/internal/filesystem/managers/directorymanager"
	"file-system/internal/filesystem/managers/inodemanager"
	"file-system/internal/filesystem/managers/usermanager"
	"file-system/internal/filesystem/superblock"
	"file-system/internal/filesystem/user"
	"file-system/internal/utils"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	FileName     string
	FileSize     uint32
	BlockSize    uint32
	RootUsername string
	RootPassword string
}

var FSConfig = Config{
	FileName:     "filesystem.data",
	FileSize:     1 * 1024 * 1024,
	BlockSize:    1024,
	RootUsername: "root",
	RootPassword: "root",
}

type FileSystem struct {
	dataFile         *os.File
	superblock       *superblock.Superblock
	blockBitmap      *bitmap.Bitmap
	inodeBitmap      *bitmap.Bitmap
	inodeManager     *inodemanager.InodeManager
	blockManager     *blockmanager.BlockManager
	directoryManager *directorymanager.DirectoryManager
	userManager      *usermanager.UserManager
}

func OpenFilesystem() (*FileSystem, error) {
	fs := FileSystem{}

	var err error
	fs.dataFile, err = os.OpenFile(FSConfig.FileName, os.O_RDWR, 0666)
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

	rootDirInode, err := fs.inodeManager.ReadInode(0)
	if err != nil {
		return nil, err
	}

	if err = fs.directoryManager.OpenDirectory(rootDirInode, "/"); err != nil {
		return nil, err
	}

	if err = fs.ChangeUser(FSConfig.RootUsername, FSConfig.RootPassword); err != nil {
		return nil, err
	}

	if err = fs.LoadUserManagerData(); err != nil {
		return nil, err
	}

	return &fs, nil
}

func FormatFilesystem(sizeInBytes uint32, blockSize uint32) (*FileSystem, error) {
	fs := FileSystem{}

	var err error
	fs.dataFile, err = os.Create(FSConfig.FileName)
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
	if err := fs.CreateHiddenDirectory(".users"); err != nil {
		return nil, err
	}
	if err := fs.AddUser(FSConfig.RootUsername, FSConfig.RootPassword); err != nil {
		return nil, err
	}
	if err := fs.ChangeUser(FSConfig.RootUsername, FSConfig.RootPassword); err != nil {
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
	fs.userManager = usermanager.NewUserManager()
}

func (fs *FileSystem) LoadUserManagerData() error {
	var err error
	if err = fs.ChangeDirectory(".users"); err != nil {
		return err
	}

	users := make(map[uint16]string)
	for _, name := range fs.GetCurrentDirectoryRecords(false) {
		if name == "." || name == ".." {
			continue
		}

		content, err := fs.ReadFile(fmt.Sprintf("/.users/%s", name))
		if err != nil {
			return err
		}
		userId, err := user.GetUserIdFromString(content)
		if err != nil {
			return err
		}
		users[userId] = name
	}
	fs.userManager.LoadUsers(users)

	if err = fs.ChangeDirectory(".."); err != nil {
		return err
	}
	return nil
}

func (fs *FileSystem) AddUser(username, password string) error {
	newUser := fs.userManager.CreateNewUser(username, password)

	if err := fs.CreateFileWithContent(fmt.Sprintf("/.users/%s", username), newUser.GetUserString()); err != nil {
		return err
	}

	if newUser.UserId != 0 {
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
	fs.userManager.Current = u

	if u.UserId != 0 {
		userDirPath := fmt.Sprintf("/%s", username)
		if err := fs.ChangeDirectory(userDirPath); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileSystem) DeleteUser(username string) error {
	if fs.userManager.Current.UserId != 0 {
		return errs.ErrPermissionDenied
	}

	content, err := fs.ReadFile(fmt.Sprintf("/.users/%s", username))
	if err != nil {
		return err
	}
	userId, err := user.GetUserIdFromString(content)
	if err != nil {
		return nil
	}

	fs.userManager.DeleteUser(userId)

	if err := fs.DeleteFile(fmt.Sprintf("/.users/%s", username)); err != nil {
		return err
	}

	if err := fs.DeleteFile(fmt.Sprintf("/%s", username)); err != nil {
		return err
	}

	return nil
}

func (fs *FileSystem) ChangeOwner(path string, username string) error {
	fs.directoryManager.SaveCurrentState()
	defer fs.directoryManager.LoadLastState()

	fileName, err := fs.evaluatePath(path)
	if err != nil {
		return err
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

	return nil
}

func (fs *FileSystem) CreateEmptyFile(path string) error {
	return fs.CreateEntity(path, true, "", false)
}

func (fs *FileSystem) CreateFileWithContent(path, content string) error {
	return fs.CreateEntity(path, true, content, false)
}

func (fs *FileSystem) CreateDirectory(path string) error {
	return fs.CreateEntity(path, false, "", false)
}

func (fs *FileSystem) CreateHiddenDirectory(path string) error {
	return fs.CreateEntity(path, false, "", true)
}

func (fs *FileSystem) CreateEntity(path string, isFile bool, content string, hidden bool) error {
	fs.directoryManager.SaveCurrentState()
	defer fs.directoryManager.LoadLastState()

	name, err := fs.evaluatePath(path)
	if err != nil {
		return err
	}

	if path != "/" {
		if _, err := fs.directoryManager.Current.GetInode(name); err == nil {
			return fmt.Errorf("%w - %s", errs.ErrRecordAlreadyExists, name)
		}

		if fs.userManager.Current != nil && !fs.directoryManager.CurrentInode.HasWritePermission(*fs.userManager.Current) {
			return fmt.Errorf("%w - %s", errs.ErrPermissionDenied, name)
		}
	}

	blockCount := (len(content)-1)/int(fs.superblock.BlockSize) + 1
	blockIndeces := make([]uint32, blockCount)
	for i := 0; i < blockCount; i++ {
		blockIndeces[i], err = fs.blockBitmap.TakeFreeBit()
		if err != nil {
			return err
		}
		fs.superblock.FreeBlockCount--
	}

	inodeIndex, err := fs.inodeBitmap.TakeFreeBit()
	if err != nil {
		return err
	}
	fs.superblock.FreeInodeCount--

	fs.superblock.Save()
	fs.blockBitmap.Save()
	fs.inodeBitmap.Save()

	var userId uint16
	if fs.userManager != nil && fs.userManager.Current != nil {
		userId = fs.userManager.Current.UserId
	}

	fileInode, err := inode.NewInode(isFile, hidden, 64, userId, blockIndeces)
	if err != nil {
		return err
	}

	fs.inodeManager.SaveInode(fileInode, inodeIndex)

	if isFile {
		if len(content) > 0 {
			err := fs.blockManager.WriteBlocks(fileInode, content)
			if err != nil {
				return err
			}
		}
	} else {
		newDir, _ := fs.directoryManager.CreateNewDirectory(inodeIndex, blockIndeces[0], path)
		if path == "/" {
			fs.directoryManager.Current = newDir
			fs.directoryManager.CurrentInode, _ = fs.inodeManager.ReadInode(inodeIndex)
		}
	}

	if path != "/" {
		fs.directoryManager.Current.AddFile(inodeIndex, name)
		fs.directoryManager.SaveCurrentDirectory()
	}

	return nil
}

func (fs *FileSystem) DeleteFile(path string) error {
	if strings.Contains(path, "/") {
		fs.directoryManager.SaveCurrentState()
		defer fs.directoryManager.LoadLastState()
	}

	name, err := fs.evaluatePath(path)
	if err != nil {
		return err
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

	if !fileInode.HasWritePermission(*fs.userManager.Current) {
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

	fs.blockManager.ResetBlocks(fileInode)
	fs.blockManager.ResetBlocks(fs.directoryManager.CurrentInode)
	fs.inodeManager.ResetInode(inodeIndex)

	fs.directoryManager.SaveCurrentDirectory()

	fs.blockBitmap.Save()
	fs.inodeBitmap.Save()
	fs.superblock.Save()

	return nil
}

func (fs *FileSystem) ChangeDirectory(path string) error {
	path = strings.TrimSuffix(path, "/")
	dirs := strings.Split(path, "/")

	for i, dirName := range dirs {
		var inodeIndex uint32
		var err error

		if dirName != "" {
			inodeIndex, err = fs.directoryManager.Current.GetInode(dirName)
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

		if fs.userManager.Current != nil && !dirInode.HasReadPermission(*fs.userManager.Current) {
			return fmt.Errorf("%w - cd %s", errs.ErrPermissionDenied, dirName)
		}

		if dirInode.IsFile() {
			return fmt.Errorf("%w - %s", errs.ErrRecordIsNotDirectory, dirName)
		}

		if dirName == "" {
			dirName = "/"
		}

		if err = fs.directoryManager.OpenDirectory(dirInode, dirName); err != nil {
			return err
		}
	}

	return nil
}

func (fs FileSystem) GetCurrentDirectoryRecords(long bool) []string {
	recordNames := fs.directoryManager.Current.GetRecords()

	result := make([]string, 0, len(recordNames))
	for _, name := range recordNames {
		recordInodeIndex, _ := fs.directoryManager.Current.GetInode(name)
		recordInode, _ := fs.inodeManager.ReadInode(recordInodeIndex)

		if recordInode.IsHidden() {
			continue
		}

		if !long {
			result = append(result, name)
			continue
		}

		tapString := recordInode.GetTypeAndPermissionString()
		ownerUsername := fs.userManager.GetUsername(recordInode.UserId)
		fileSizeInBytes := recordInode.FileSize * fs.superblock.BlockSize
		modificationTime := time.Unix(int64(recordInode.ModificationTime), 0)
		modificationTimeString := modificationTime.Format("Jan 2 15:04")

		result = append(result, fmt.Sprintf("%s\t%s\t%d\t%s\t%s", tapString, ownerUsername, fileSizeInBytes, modificationTimeString, name))
	}

	return result
}

func (fs FileSystem) ReadFile(path string) (string, error) {
	fs.directoryManager.SaveCurrentState()
	defer fs.directoryManager.LoadLastState()

	name, err := fs.evaluatePath(path)
	if err != nil {
		return "", err
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(name)
	if err != nil {
		return "", err
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
	if err != nil {
		return "", err
	}

	if fs.userManager.Current != nil && !fileInode.HasReadPermission(*fs.userManager.Current) {
		return "", fmt.Errorf("%w - read %s", errs.ErrPermissionDenied, name)
	}

	str, err := fs.blockManager.ReadBlocks(fileInode, name)
	if err != nil {
		return "", err
	}

	return str, nil
}

func (fs FileSystem) EditFile(path string, content string) error {
	fs.directoryManager.SaveCurrentState()
	defer fs.directoryManager.LoadLastState()

	name, err := fs.evaluatePath(path)
	if err != nil {
		return err
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(name)
	if err != nil {
		return err
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
	if err != nil {
		return err
	}

	if !fileInode.HasWritePermission(*fs.userManager.Current) {
		return fmt.Errorf("%w - %s", errs.ErrPermissionDenied, name)
	}

	if !fileInode.IsFile() {
		return fmt.Errorf("%w - %s", errs.ErrRecordIsNotFile, name)
	}

	fs.blockManager.ResetBlocks(fileInode)

	newFileSize := (len(content)-1)/int(fs.superblock.BlockSize) + 1
	oldFileSize := int(fileInode.FileSize)
	if newFileSize > oldFileSize {
		for i := oldFileSize; i < newFileSize; i++ {
			fileInode.Blocks[i], err = fs.blockBitmap.TakeFreeBit()
			if err != nil {
				return err
			}
			fs.superblock.FreeBlockCount--
		}

	} else if newFileSize < oldFileSize {
		for i := newFileSize; i < oldFileSize; i++ {
			err := fs.blockBitmap.SetBit(fileInode.Blocks[i], 0)
			if err != nil {
				return err
			}
			fileInode.Blocks[i] = 0
			fs.superblock.FreeBlockCount++
		}
	}
	fileInode.FileSize = uint32(newFileSize)

	fs.blockManager.WriteBlocks(fileInode, content)

	fileInode.ModificationTime = uint32(time.Now().Unix())
	fs.inodeManager.SaveInode(fileInode, inodeIndex)

	return nil
}

func (fs *FileSystem) AppendToFile(path string, content string) error {
	original, err := fs.ReadFile(path)
	if err != nil {
		return err
	}
	return fs.EditFile(path, original+content)
}

func (fs *FileSystem) MoveFile(pathFrom string, pathTo string) error {
	fs.directoryManager.SaveCurrentState()
	defer fs.directoryManager.LoadLastState()

	nameFrom, err := fs.evaluatePath(pathFrom)
	if err != nil {
		return err
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(nameFrom)
	if err != nil {
		return nil
	}

	fs.directoryManager.Current.DeleteFile(nameFrom)
	fs.directoryManager.SaveCurrentDirectory()

	fs.directoryManager.LoadLastState()

	nameTo, err := fs.evaluatePath(pathTo)
	if err != nil {
		return err
	}

	fs.directoryManager.Current.AddFile(inodeIndex, nameTo)
	fs.directoryManager.SaveCurrentDirectory()

	return nil
}

func (fs *FileSystem) CopyFile(pathFrom string, pathTo string) error {
	fs.directoryManager.SaveCurrentState()
	defer fs.directoryManager.LoadLastState()

	nameFrom, err := fs.evaluatePath(pathFrom)
	if err != nil {
		return err
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(nameFrom)
	if err != nil {
		return nil
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
	if err != nil {
		return nil
	}

	if fs.userManager.Current != nil && !fileInode.HasReadPermission(*fs.userManager.Current) {
		return fmt.Errorf("%w - copy %s", errs.ErrPermissionDenied, nameFrom)
	}

	var fileContent string
	var directoryRecordNames []string

	if fileInode.IsFile() {
		fileContent, err = fs.blockManager.ReadBlocks(fileInode, nameFrom)
		if err != nil {
			return err
		}
	} else {
		if err := fs.ChangeDirectory(nameFrom); err != nil {
			return err
		}
		directoryRecordNames = fs.directoryManager.Current.GetRecords()
	}

	fs.directoryManager.LoadLastState()

	if fileInode.IsFile() {
		if err = fs.CreateFileWithContent(pathTo, fileContent); err != nil {
			return err
		}
	} else {
		fs.CreateDirectory(pathTo)
		for _, name := range directoryRecordNames {
			if name == "." || name == ".." {
				continue
			}
			oldPath := pathFrom + "/" + name
			newPath := pathTo + "/" + name
			if err := fs.CopyFile(oldPath, newPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (fs *FileSystem) ChangePermissions(path string, value int) error {
	fs.directoryManager.SaveCurrentState()
	defer fs.directoryManager.LoadLastState()

	name, err := fs.evaluatePath(path)
	if err != nil {
		return err
	}

	inodeIndex, err := fs.directoryManager.Current.GetInode(name)
	if err != nil {
		return err
	}

	fileInode, err := fs.inodeManager.ReadInode(inodeIndex)
	if err != nil {
		return err
	}

	if !fileInode.HasWritePermission(*fs.userManager.Current) {
		return fmt.Errorf("%w - chmod %s", errs.ErrPermissionDenied, name)
	}

	err = fileInode.ChangePermissions(value)
	if err != nil {
		return err
	}

	fs.inodeManager.SaveInode(fileInode, inodeIndex)

	return nil
}

func (fs FileSystem) GetCurrentPath() string {
	return fs.directoryManager.Path
}

func (fs FileSystem) GetCurrentUserName() string {
	return fs.userManager.Current.Username
}

func (fs *FileSystem) CloseDataFile() error {
	return fs.dataFile.Close()
}

func (fs *FileSystem) evaluatePath(path string) (string, error) {
	pathToFolder, name := utils.SplitPath(path)
	if pathToFolder != "" {
		err := fs.ChangeDirectory(pathToFolder)
		if err != nil {
			return "", err
		}
	}
	return name, nil
}
