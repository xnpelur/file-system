package filesystem

import (
	"errors"
	"file-system/internal/errs"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestFilesystemIntegration(t *testing.T) {
	fs, err := FormatFilesystem(1024*1024, 1024)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}
	defer fs.CloseDataFile()

	const fileContent = "Hello, World!"
	const updatedFileContent = "Updated file content"

	t.Run("TestCreateFile", func(t *testing.T) {
		err := fs.CreateFile("test.txt", fileContent)
		if err != nil {
			t.Errorf("CreateFile error: %v", err)
		}
	})

	t.Run("TestCreateDirectory", func(t *testing.T) {
		err := fs.CreateDirectory("/testdir")
		if err != nil {
			t.Errorf("CreateDirectory error: %v", err)
		}
	})

	t.Run("TestReadFile", func(t *testing.T) {
		content, err := fs.ReadFile("test.txt")
		if err != nil {
			t.Errorf("ReadFile error: %v", err)
		}
		if content != fileContent {
			t.Errorf("ReadFile content mismatch: expected \"%s\", got \"%s\"", fileContent, content)
		}
	})

	t.Run("TestEditFile", func(t *testing.T) {
		err := fs.EditFile("test.txt", updatedFileContent)
		if err != nil {
			t.Errorf("EditFile error: %v", err)
		}

		content, err := fs.ReadFile("test.txt")
		if err != nil {
			t.Errorf("ReadFile error: %v", err)
		}
		if content != updatedFileContent {
			t.Errorf("ReadFile content mismatch: expected \"%s\", got \"%s\"", updatedFileContent, content)
		}
	})

	t.Run("TestDeleteFile", func(t *testing.T) {
		err := fs.DeleteFile("test.txt", fs.currentDirectory, fs.currentDirectoryInode)
		if err != nil {
			t.Errorf("DeleteFile error: %v", err)
		}

		content, err := fs.ReadFile("test.txt")
		if !errors.Is(err, errs.ErrRecordNotFound) {
			t.Errorf("DeleteFile error: record was not properly deleted. File content: %s", content)
		}
	})
}

func TestDeleteDirectoryWithNestedFiles(t *testing.T) {
	fs, err := FormatFilesystem(1024*1024, 1024)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}
	defer fs.CloseDataFile()

	nestingLevel := 10

	for i := 1; i <= nestingLevel; i++ {
		fileName := fmt.Sprintf("file%d", i)
		dirName := fmt.Sprintf("dir%d", i)

		if err = fs.CreateDirectory(dirName); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dirName, err)
		}
		if err = fs.ChangeDirectory(dirName); err != nil {
			t.Fatalf("Failed to change directory to %s: %v", dirName, err)
		}
		if err = fs.CreateFile(fileName, ""); err != nil {
			t.Fatalf("Failed to create file %s: %v", fileName, err)
		}
	}

	dotDotComponents := make([]string, nestingLevel)
	for i := range dotDotComponents {
		dotDotComponents[i] = ".."
	}
	pathToRoot := strings.Join(dotDotComponents, "/")

	if err = fs.ChangeDirectory(pathToRoot); err != nil {
		t.Fatalf("Failed to go back to root - cd %s: %v", pathToRoot, err)
	}

	if err = fs.DeleteFile("dir1", fs.currentDirectory, fs.currentDirectoryInode); err != nil {
		t.Fatalf("Failed to delete dir1: %v", err)
	}
}

func TestDataFileIdempotency(t *testing.T) {
	fs, err := FormatFilesystem(1024*1024, 1024)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}
	defer fs.CloseDataFile()

	savedContent, err := io.ReadAll(fs.dataFile)
	if err != nil {
		t.Fatalf("Failed to read data file: %v", err)
	}

	if err = fs.CreateDirectory("dir"); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err = fs.CreateFile("file", "file content"); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err = fs.ChangeDirectory("dir"); err != nil {
		t.Fatalf("Failed to change current directory: %v", err)
	}
	if err = fs.CreateFile("otherfile", "other file content"); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err = fs.CreateDirectory("otherdir"); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err = fs.ChangeDirectory(".."); err != nil {
		t.Fatalf("Failed to change current directory: %v", err)
	}

	if err = fs.DeleteFile("dir", fs.currentDirectory, fs.currentDirectoryInode); err != nil {
		t.Fatalf("Failed to delete directory: %v", err)
	}
	if err = fs.DeleteFile("file", fs.currentDirectory, fs.currentDirectoryInode); err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	currentContent, err := io.ReadAll(fs.dataFile)
	if err != nil {
		t.Fatalf("Failed to read data file: %v", err)
	}

	if string(currentContent) != string(savedContent) {
		t.Errorf("File content mismatch. Expected: %s, Got: %s", savedContent, currentContent)
	}
	diffIndex := findFirstDifference(savedContent, currentContent)

	if diffIndex != -1 {
		expectedByte := savedContent[diffIndex]
		gotByte := currentContent[diffIndex]
		t.Errorf("File content mismatch at byte index %d. Expected: %x, Got: %x", diffIndex, expectedByte, gotByte)
	}
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
