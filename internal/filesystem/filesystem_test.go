package filesystem

import (
	"testing"
)

func TestFilesystemIntegration(t *testing.T) {
	fs, err := FormatFilesystem(1024*1024, 1024)
	if err != nil {
		t.Fatalf("Failed to open filesystem: %v", err)
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
		if err == nil {
			t.Errorf("DeleteFile error: fs.ReadFile does not return error after file deletion. File content: %s", content)
		}
	})
}
