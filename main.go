package main

import (
	"bufio"
	"fmt"
	"gopkg.in/go-ini/ini.v1"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
)

type Todo struct {
	Prefix   string
	Suffix   string
	Id       *string
	Filename string
	Line     int
}

type GithubCredentials struct {
	PersonalToken string
}

func GithubCredentialsFromFile(filepath string) (GithubCredentials, error) {
	cfg, err := ini.Load(filepath)
	if err != nil {
		return GithubCredentials{}, err
	}

	return GithubCredentials {
		PersonalToken : cfg.Section("github").Key("personal_token").String(),
	}, nil
}

func (todo Todo) String() string {
	if todo.Id == nil {
		return fmt.Sprintf("%s:%d: %sTODO: %s\n",
			todo.Filename, todo.Line,
			todo.Prefix, todo.Suffix)
	} else {
		return fmt.Sprintf("%s:%d: %sTODO(%s): %s\n",
			todo.Filename, todo.Line,
			todo.Prefix, *todo.Id, todo.Suffix)
	}
}

func (todo Todo) Update() error {
	// TODO(#19): Todo.Update() is not implemented
	return nil
}

func ref_str(x string) *string {
	return &x
}

func LineAsUnreportedTodo(line string) *Todo {
	unreportedTodo := regexp.MustCompile("^(.*)TODO: (.*)$")
	groups := unreportedTodo.FindStringSubmatch(line)

	if groups != nil {
		return &Todo{
			Prefix:   groups[1],
			Suffix:   groups[2],
			Id:       nil,
			Filename: "",
			Line:     0,
		}
	}

	return nil
}

func LineAsReportedTodo(line string) *Todo {
	unreportedTodo := regexp.MustCompile("^(.*)TODO\\((.*)\\): (.*)$")
	groups := unreportedTodo.FindStringSubmatch(line)

	if groups != nil {
		return &Todo{
			Prefix:   groups[1],
			Suffix:   groups[3],
			Id:       &groups[2],
			Filename: "",
			Line:     0,
		}
	}

	return nil
}

func LineAsTodo(line string) *Todo {
	// TODO(#2): LineAsTodo has false positive result inside of string literals
	if todo := LineAsUnreportedTodo(line); todo != nil {
		return todo
	}

	if todo := LineAsReportedTodo(line); todo != nil {
		return todo
	}

	return nil
}

func WalkTodosOfFile(path string, visit func (Todo) error) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	line := 1
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		todo := LineAsTodo(scanner.Text())

		if todo != nil {
			todo.Filename = path
			todo.Line = line

			if err := visit(*todo); err != nil {
				return err
			}
		}

		line = line + 1
	}

	return scanner.Err()
}

func WalkTodosOfDir(dirpath string, visit func(todo Todo) error) error {
	return filepath.Walk(dirpath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			err := WalkTodosOfFile(path, visit)

			if err != nil {
				return err
			}
		}

		return nil
	})
}

func ListSubcommand() error {
	return WalkTodosOfDir(".", func(todo Todo) error {
		fmt.Printf("%v", todo)
		return nil
	})
}

func ReportTodo(todo Todo, creds GithubCredentials, repo string) (Todo, error) {
	// TODO(#20): ReportTodo is not implemented
	return Todo{}, nil
}

func ReportSubcommand(creds GithubCredentials, repo string) error {
	reportedTodos := []Todo{}

	err := WalkTodosOfDir(".", func(todo Todo) error {
		if todo.Id == nil {
			reportedTodo, err := ReportTodo(todo, creds, repo)

			if err != nil {
				return err
			}

			fmt.Printf("[REPORTED] %v\n", todo)

			reportedTodos = append(reportedTodos, reportedTodo)
		}

		return nil
	})

	if err != nil {
		return err
	}

	for _, todo := range reportedTodos {
		err := todo.Update()
		if err != nil {
			return err
		}
	}

	return nil
}

func usage() {
	// TODO(#9): implement a map for options instead of println'ing them all there
	fmt.Printf("snitch [opt]\n" +
		"\tlist: lists all todos of a dir recursively\n" +
		"\treport <owner/repo>: reports an issue to github\n")
}

func main() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	creds, err := GithubCredentialsFromFile(
		path.Join(usr.HomeDir, ".snitch/github.ini"))
	if err != nil {
		panic(err)
	}

	// TODO(#16): error results of subcommands are not handled
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			ListSubcommand()
		case "report":
			if len(os.Args) < 3 {
				usage()
				panic("Not enough arguments")
			}
			// TODO: GitHub repo is not automatically derived from the git repo
			ReportSubcommand(creds, os.Args[2])
		default:
			panic(fmt.Sprintf("`%s` unknown command", os.Args[1]))
		}
	} else {
		usage()
	}
}
