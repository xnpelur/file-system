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
		parts := parseCommandLine(input)

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

func parseCommandLine(command string) []string {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	escapeNext := true
	for _, c := range command {
		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}
		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}
		if c == '\\' {
			escapeNext = true
			continue
		}
		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}
		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}
		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
		}
	}
	if current != "" {
		args = append(args, current)
	}
	return args
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
