package menu

import (
	"bufio"
	"file-system/internal/errs"
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
		fmt.Printf("root@filesystem:%s$ ", m.fileSystem.GetCurrentPath())
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
		} else if parts[0] == "format" {
			ans := getYesOrNo("Вы уверены, что хотите форматировать файловую систему (все данные будут потеряны)? (y/n): ")
			if ans {
				m.fileSystem, err = filesystem.FormatFilesystem(1*1024*1024, 1024)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println("Файловая система форматирована.")
			}
		} else {
			err := m.executeCommand(parts[0], parts[1:])
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}
}

func (m *Menu) executeCommand(command string, args []string) error {
	switch command {
	case "create":
		if len(args) < 1 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		fileName := args[0]

		slashCount := strings.Count(fileName, "/")
		if slashCount > 0 {
			if slashCount == 1 && strings.HasSuffix(fileName, "/") {
				return m.fileSystem.CreateDirectory(fileName[:len(fileName)-1])
			}
			return fmt.Errorf("%w - %s", errs.ErrIncorrectFileName, fileName)
		}

		if strings.HasSuffix(fileName, ".") {
			return fmt.Errorf("%w - %s", errs.ErrIncorrectFileName, fileName)
		}

		fileContent := ""
		if len(args) > 1 {
			fileContent = args[1]
		}
		return m.fileSystem.CreateFile(fileName, fileContent)
	case "edit":
		if len(args) < 1 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		fileName := args[0]
		fileContent := ""
		if len(args) > 1 {
			fileContent = args[1]
		}
		return m.fileSystem.EditFile(fileName, fileContent)
	case "read":
		if len(args) < 1 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		fileName := args[0]
		content, err := m.fileSystem.ReadFile(fileName)
		if err != nil {
			return err
		}
		fmt.Println(content)
		return nil
	case "delete":
		if len(args) < 1 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		fileName := args[0]
		return m.fileSystem.DeleteFile(fileName, nil, nil)
	case "list":
		for _, name := range m.fileSystem.GetCurrentDirectoryRecords() {
			fmt.Println(name)
		}
		return nil
	case "cd":
		if len(args) < 1 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		return m.fileSystem.ChangeDirectory(args[0])
	default:
		return fmt.Errorf("%w - %s", errs.ErrUnknownCommand, command)
	}
}

func parseCommand(input string) []string {
	var parts []string
	var currentPart string
	inQuotes := false

	for _, part := range strings.Fields(input) {
		if strings.HasPrefix(part, `"`) {
			if strings.HasSuffix(part, `"`) {
				parts = append(parts, part[1:len(part)-1])
				continue
			}
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
