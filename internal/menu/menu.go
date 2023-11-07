package menu

import (
	"bufio"
	"file-system/internal/filesystem"
	"fmt"
	"log"
	"os"
)

type Menu struct {
	fileSystem *filesystem.FileSystem
}

func NewMenu() Menu {
	return Menu{}
}

func (m Menu) Start() {
	var err error
	m.fileSystem, err = filesystem.FormatFilesystem(1*1024*1024, 1024) // 1Mb - filesystem, 1kb - block
	if err != nil {
		log.Fatal(err)
	}
	defer m.fileSystem.CloseDataFile()

	for {
		fmt.Print("root@filesystem:/$ ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		err := scanner.Err()
		if err != nil {
			log.Fatal(err)
		}
		input := scanner.Text()

		if input == "exit" {
			fmt.Println("File system closed.")
			return
		} else {
			err := m.fileSystem.ExecuteCommand(input)
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}
}
