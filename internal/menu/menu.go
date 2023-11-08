package menu

import (
	"bufio"
	"file-system/internal/filesystem"
	"fmt"
	"log"
	"os"
	"strings"
)

type Menu struct {
	fileSystem *filesystem.FileSystem
}

func NewMenu() Menu {
	return Menu{}
}

func (m Menu) Start() {
	var err error
	m.fileSystem, err = filesystem.OpenFilesystem()
	if err != nil {
		fmt.Printf("Не удалось открыть файловую систему из файла %s\n", filesystem.FilesystemConfig.FileName)
		ans := getYesOrNo("Форматировать новую файловую систему (все данные будут потеряны)? (y/n): ")
		if ans {
			m.fileSystem, err = filesystem.FormatFilesystem(1*1024*1024, 1024) // 1Mb - filesystem, 1kb - block
			if err != nil {
				log.Fatal(err)
			}
		} else {
			return
		}
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
		parts := parseCommand(input)

		if parts[0] == "exit" {
			fmt.Println("File system closed.")
			return
		} else {
			err := m.fileSystem.ExecuteCommand(parts[0], parts[1:])
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}
}

func parseCommand(input string) []string {
	var parts []string
	var currentPart string
	inQuotes := false

	for _, part := range strings.Fields(input) {
		if strings.HasPrefix(part, `"`) {
			inQuotes = true
			currentPart = part[1:]
		} else if strings.HasSuffix(part, `"`) {
			inQuotes = false
			currentPart += " " + part[:len(part)-1]
			parts = append(parts, currentPart)
			currentPart = ""
		} else if inQuotes {
			currentPart += " " + part
		} else {
			parts = append(parts, part)
		}
	}

	if len(currentPart) > 0 {
		parts = append(parts, currentPart)
	}

	return parts
}

func getYesOrNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		input = strings.ToLower(input)

		if input == "y" {
			return true
		} else if input == "n" {
			return false
		}
		fmt.Println("Некорректный ввод, попробуйте ещё раз.")
	}
}
