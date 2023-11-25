package menu

import (
	"bufio"
	"errors"
	"file-system/internal/errs"
	"file-system/internal/filesystem"
	"fmt"
	"log"
	"os"
	"strconv"
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
		fmt.Printf("%s@filesystem:%s$ ", m.fileSystem.GetCurrentUserName(), m.fileSystem.GetCurrentPath())
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		err := scanner.Err()
		if err != nil {
			log.Fatal(err)
		}
		input := scanner.Text()
		parts := parseCommandLine(input)

		if len(parts) == 0 {
			continue
		}

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

		if strings.HasSuffix(fileName, ".") {
			return fmt.Errorf("%w - %s", errs.ErrIncorrectFileName, fileName)
		}

		if strings.HasSuffix(fileName, "/") {
			return m.fileSystem.CreateDirectory(fileName[:len(fileName)-1])
		}

		if len(args) > 1 {
			fileContent := args[1]
			return m.fileSystem.CreateFileWithContent(fileName, fileContent)
		}
		return m.fileSystem.CreateEmptyFile(fileName)
	case "edit":
		if len(args) < 2 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		return m.fileSystem.EditFile(args[0], args[1])
	case "append":
		if len(args) < 2 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		return m.fileSystem.AppendToFile(args[0], args[1])
	case "move":
		if len(args) < 2 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		return m.fileSystem.MoveFile(args[0], args[1])
	case "copy":
		if len(args) < 2 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		return m.fileSystem.CopyFile(args[0], args[1])
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
		return m.fileSystem.DeleteFile(fileName)
	case "list":
		long := len(args) > 0 && args[0] == "-l"
		for _, name := range m.fileSystem.GetCurrentDirectoryRecords(long) {
			fmt.Println(name)
		}
		return nil
	case "cd":
		if len(args) < 1 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		return m.fileSystem.ChangeDirectory(args[0])
	case "changeuser":
		if len(args) < 2 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		err := m.fileSystem.ChangeUser(args[0], args[1])
		if err != nil {
			if errors.Is(err, errs.ErrIncorrectPassword) {
				fmt.Println("Неправильный пароль")
				return nil
			}
			if errors.Is(err, errs.ErrRecordNotFound) {
				fmt.Println("Пользователя с таким именем не существует")
				return nil
			}
			return err
		}
		return nil
	case "adduser":
		if len(args) < 2 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		return m.fileSystem.AddUser(args[0], args[1])
	case "deleteuser":
		if len(args) < 1 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		return m.fileSystem.DeleteUser(args[0])
	case "chmod":
		if len(args) < 1 {
			return fmt.Errorf("%w - %s", errs.ErrMissingArguments, command)
		}
		path := args[0]
		permissions, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		return m.fileSystem.ChangePermissions(path, permissions)
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
