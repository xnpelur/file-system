package filesystem

import (
	"errors"
	"file-system/internal/errs"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestFilesystemIntegration(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	const fileContent = "Hello, World!"
	const updatedFileContent = "Updated file content"

	t.Run("TestCreateFile", func(t *testing.T) {
		fs.CreateFileWithContent("test.txt", fileContent)
	})

	t.Run("TestCreateDirectory", func(t *testing.T) {
		fs.CreateDirectory("/testdir")
	})

	t.Run("TestReadFile", func(t *testing.T) {
		content, _ := fs.ReadFile("test.txt")
		if content != fileContent {
			t.Errorf("ReadFile content mismatch: expected \"%s\", got \"%s\"", fileContent, content)
		}
	})

	t.Run("TestEditFile", func(t *testing.T) {
		err := fs.EditFile("test.txt", updatedFileContent)
		if err != nil {
			t.Errorf("EditFile error: %v", err)
		}

		content, _ := fs.ReadFile("test.txt")
		if content != updatedFileContent {
			t.Errorf("ReadFile content mismatch: expected \"%s\", got \"%s\"", updatedFileContent, content)
		}
	})

	t.Run("TestDeleteFile", func(t *testing.T) {
		fs.DeleteFile("test.txt")

		content, err := fs.ReadFile("test.txt")
		if !errors.Is(err, errs.ErrRecordNotFound) {
			t.Errorf("DeleteFile error: record was not properly deleted. File content: %s", content)
		}
	})
}

func TestDeleteDirectoryWithNestedFiles(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	nestingLevel := 10

	for i := 1; i <= nestingLevel; i++ {
		fileName := fmt.Sprintf("file%d", i)
		dirName := fmt.Sprintf("dir%d", i)

		fs.CreateDirectory(dirName)
		fs.ChangeDirectory(dirName)
		fs.CreateEmptyFile(fileName)
	}

	dotDotComponents := make([]string, nestingLevel)
	for i := range dotDotComponents {
		dotDotComponents[i] = ".."
	}
	pathToRoot := strings.Join(dotDotComponents, "/")

	fs.ChangeDirectory(pathToRoot)
	fs.DeleteFile("dir1")
}

func TestDataFileSimpleIdempotency(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	savedContent, _ := os.ReadFile(fs.dataFile.Name())

	fs.CreateFileWithContent("file", "file content")
	fs.DeleteFile("file")

	currentContent, _ := os.ReadFile(fs.dataFile.Name())

	diffIndex := findFirstDifference(savedContent, currentContent)

	if diffIndex != -1 {
		expectedByte := savedContent[diffIndex]
		gotByte := currentContent[diffIndex]
		t.Errorf("File content mismatch at byte index %d. Expected: %x, Got: %x", diffIndex, expectedByte, gotByte)
	}
}

func TestDataFileComplexIdempotency(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	savedContent, _ := os.ReadFile(fs.dataFile.Name())

	fs.CreateDirectory("dir")
	fs.CreateFileWithContent("file", "file content")
	fs.ChangeDirectory("dir")
	fs.CreateFileWithContent("otherfile", "other file content")
	fs.CreateDirectory("otherdir")
	fs.ChangeDirectory("..")
	fs.DeleteFile("dir")
	fs.DeleteFile("file")

	currentContent, _ := os.ReadFile(fs.dataFile.Name())

	diffIndex := findFirstDifference(savedContent, currentContent)

	if diffIndex != -1 {
		expectedByte := savedContent[diffIndex]
		gotByte := currentContent[diffIndex]
		t.Errorf("File content mismatch at byte index %d. Expected: %x, Got: %x", diffIndex, expectedByte, gotByte)
	}
}

func TestAppendToFile(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	const fileName = "file.txt"
	const fileContent = "Hello, "
	const append = "world!"
	const updatedFileContent = fileContent + append

	fs.CreateFileWithContent(fileName, fileContent)
	fs.AppendToFile(fileName, append)
	content, _ := fs.ReadFile(fileName)
	if content != updatedFileContent {
		t.Errorf("AppendToFile content mismatch: expected \"%s\", got \"%s\"", updatedFileContent, content)
	}
}

func TestMoveFile(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	fileContent := "Test string"

	fs.CreateDirectory("dir1")
	fs.CreateFileWithContent("dir1/file1", fileContent)
	fs.CreateDirectory("dir2")

	fs.MoveFile("dir1", "dir2/dir1")

	content, _ := fs.ReadFile("dir2/dir1/file1")
	if content != fileContent {
		t.Errorf("MoveFile content mismatch: expected \"%s\", got \"%s\"", fileContent, content)
	}
}

func TestCopyFileSimple(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	fileContent := "Test string"

	fs.CreateFileWithContent("file1", fileContent)
	fs.CopyFile("file1", "file2")
	content, _ := fs.ReadFile("file2")
	if content != fileContent {
		t.Errorf("CopyFile content mismatch: expected \"%s\", got \"%s\"", fileContent, content)
	}
}

func TestAddUser(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	username := "user"
	password := "user"

	fs.AddUser(username, password)

	err := fs.ChangeDirectory("/" + username)
	if err != nil {
		t.Errorf("Error on user directory changing: %s", username)
	}

	content, err := fs.ReadFile(fmt.Sprintf("/.users/%s", username))
	if err != nil || len(content) == 0 {
		t.Errorf("Error on user file reading: %s", username)
	}
}

func TestChangeUser(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	fs.AddUser("user1", "password")
	fs.AddUser("user2", "password")

	fs.ChangeUser("user1", "password")
	fs.ChangeUser("user2", "password")
	fs.ChangeUser(FSConfig.RootUsername, FSConfig.RootPassword)
}

func TestDeleteUser(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	fs.AddUser("user", "password")
	fs.ChangeUser("user", "password")
	fs.ChangeUser(FSConfig.RootUsername, FSConfig.RootPassword)
	fs.DeleteUser("user")

	err := fs.ChangeUser("user", "password")
	if !errors.Is(err, errs.ErrRecordNotFound) {
		t.Errorf("ChangeUser to deleted user error mismatch: expected \"%v\", got \"%v\"", errs.ErrRecordNotFound, err)
	}
}

func TestPermissions(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	fileContent := "Test string"

	fs.CreateFileWithContent("file", fileContent)
	fs.AddUser("user", "password")
	fs.ChangeUser("user", "password")

	// ReadFile should work
	content, _ := fs.ReadFile("/file")
	if content != fileContent {
		t.Errorf("ReadFile after ChangeUser content mismatch: expected \"%s\", got \"%s\"", fileContent, content)
	}

	// CreateFile should not work
	err := fs.CreateEmptyFile("/newfile")
	if !errors.Is(err, errs.ErrPermissionDenied) {
		t.Errorf("CreateEmptyFile after ChangeUser error mismatch: expected \"%v\", got \"%v\"", errs.ErrPermissionDenied, err)
	}
}

func TestChangePermissions(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	file1Content := "Test string 1"
	file2Content := "Test string 2"

	fs.CreateEmptyFile("file1")
	fs.CreateFileWithContent("file2", file2Content)
	fs.ChangePermissions("file1", 66)
	fs.ChangePermissions("file2", 60)

	fs.AddUser("user", "password")
	fs.ChangeUser("user", "password")

	fs.EditFile("/file1", file1Content)
	content, _ := fs.ReadFile("/file1")
	if content != file1Content {
		t.Errorf("ReadFile after EditFile content mismatch: expected \"%s\", got \"%s\"", file1Content, content)
	}

	_, err := fs.ReadFile("/file2")
	if !errors.Is(err, errs.ErrPermissionDenied) {
		t.Errorf("ReadFile on file with 60 permissions error mismatch: expected \"%v\", got \"%v\"", errs.ErrPermissionDenied, err)
	}
}

func TestCopyFileComplex(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	fileContent := "Test string"

	fs.CreateDirectory("dir1")
	fs.CreateFileWithContent("dir1/file1", fileContent)
	fs.CreateDirectory("dir2")

	fs.CopyFile("dir1", "dir2/dir1copy")

	content, _ := fs.ReadFile("dir2/dir1copy/file1")
	if content != fileContent {
		t.Errorf("CopyFile content mismatch: expected \"%s\", got \"%s\"", fileContent, content)
	}
}

func TestReadLargeFile(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	for blockCount := 0; blockCount < 10; blockCount++ {
		fileContent := strings.Repeat("#", blockCount*int(FSConfig.BlockSize))

		fileName := fmt.Sprintf("test%d.txt", blockCount)
		fs.CreateFileWithContent(fileName, fileContent)
		content, _ := fs.ReadFile(fileName)

		if content != fileContent {
			t.Errorf("ReadFile error on %d blocks long file", blockCount)
		}
	}
}

func TestDataFileIdempotencyWithLargeFile(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	savedContent, _ := os.ReadFile(fs.dataFile.Name())

	blockCount := 10
	fileContent := strings.Repeat("#", blockCount*int(FSConfig.BlockSize))

	fileName := fmt.Sprintf("test%d.txt", blockCount)
	fs.CreateFileWithContent(fileName, fileContent)
	fs.DeleteFile(fileName)

	currentContent, _ := os.ReadFile(fs.dataFile.Name())

	diffIndex := findFirstDifference(savedContent, currentContent)

	if diffIndex != -1 {
		expectedByte := savedContent[diffIndex]
		gotByte := currentContent[diffIndex]
		t.Errorf("File content mismatch at byte index %d. Expected: %x, Got: %x", diffIndex, expectedByte, gotByte)
	}
}

func TestReadLargeDirectory(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	fileCount := 500

	fs.CreateDirectory("dir")
	fs.ChangeDirectory("dir")
	for i := 0; i < fileCount; i++ {
		if i == 1019 {
			fmt.Print("hi")
		}
		fs.CreateEmptyFile(fmt.Sprintf("file%d", i))
	}
	fs.ChangeDirectory("..")
	fs.ChangeDirectory("dir")

	currentRecords := fs.GetCurrentDirectoryRecords(false)
	currentFileCount := len(currentRecords) - 2
	if currentFileCount != fileCount {
		t.Errorf("Directory records count mismatch: expected %d, got %d", fileCount, currentFileCount)
	}
}

func setupFilesystem(t *testing.T) (*FileSystem, func()) {
	fs, _ := FormatFilesystem(FSConfig.FileSize, FSConfig.BlockSize)

	cleanup := func() {
		fs.CloseDataFile()
		os.Remove(fs.dataFile.Name())
	}

	return fs, cleanup
}

func findFirstDifference(slice1, slice2 []byte) int {
	minLen := len(slice1)
	if len(slice2) < minLen {
		minLen = len(slice2)
	}

	for i := 0; i < minLen; i++ {
		if slice1[i] != slice2[i] {
			return i
		}
	}

	if len(slice1) == len(slice2) {
		return -1
	}

	return minLen
}
