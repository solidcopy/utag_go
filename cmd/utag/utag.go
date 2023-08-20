package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/solidcopy/utag/internal/service"
)

func main() {

	var serviceArg string
	if len(os.Args) >= 2 {
		serviceArg = os.Args[1]
	}

	var dir string
	if len(os.Args) >= 3 {
		dir = os.Args[2]
	} else {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "カレントディレクトリが取得できません。: %s\n", err)
			os.Exit(1)
		}
		dir = wd
	}

	serviceList, err := selectServices(serviceArg, dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, service := range serviceList {
		service(dir)
	}
}

type ServiceFunc func(string)

func selectServices(args string, dir string) ([]ServiceFunc, error) {
	if args == "" {
		return selectServicesByFile(dir)
	} else {
		service, err := selectServicesByArgs(args)
		if err != nil {
			return nil, err
		}
		return []ServiceFunc{service}, nil
	}
}

func selectServicesByArgs(arg string) (ServiceFunc, error) {

	switch arg {
	case "e":
		return service.ExecuteExport, nil
	case "i":
		return service.ExecuteImport, nil
	case "r":
		return service.ExecuteRename, nil
	default:
		err := fmt.Errorf("サブコマンドが不正です。 \"%s\"", arg)
		return nil, err
	}
}

func selectServicesByFile(dir string) ([]ServiceFunc, error) {
	tagsFilePath := filepath.Join(dir, "tags")
	if _, err := os.Stat(tagsFilePath); err == nil {
		return []ServiceFunc{service.ExecuteImport, service.ExecuteRename}, nil
	} else {
		return []ServiceFunc{service.ExecuteExport}, nil
	}
}
