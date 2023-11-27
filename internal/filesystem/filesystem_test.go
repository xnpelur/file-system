package filesystem

import (
	"errors"
	"file-system/internal/errs"
	"fmt"
	"os"
	"strings"
	"testing"
)

const filesystemSize = 1024 * 1024
const blockSize = 1024

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

	// TODO
	_ = fs
}

func TestCopyFile(t *testing.T) {
	fs, cleanup := setupFilesystem(t)
	t.Cleanup(cleanup)

	// TODO
	_ = fs
}

func setupFilesystem(t *testing.T) (*FileSystem, func()) {
	fs, _ := FormatFilesystem(filesystemSize, blockSize)

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
