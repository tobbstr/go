// Copyright 2022 tobbstr. All rights reserved.
// Use of this source code is governed by a MIT-
// license that can be found in the LICENSE file.
package module

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// NameFrom returns the module name, otherwise it returns an error. The argument path has to be a valid
// module root path i.e. a path that contains a go.mod file.
func NameFrom(path string) (string, error) {
	goModFile, err := os.Open(fmt.Sprintf("%s/go.mod", path))
	if err != nil {
		log.Fatalf("could not open go.mod: %s", err)
	}

	goModScanner := bufio.NewScanner(goModFile)
	goModScanner.Split(bufio.ScanLines)

	for goModScanner.Scan() {
		line := goModScanner.Text()
		if strings.Contains(line, "module ") {
			return line[7:], err
		}
	}
	return "", fmt.Errorf("could not find go.mod file in path = %s", path)
}

// RootPathFromWorkingDir walks back up the file system until it finds a module root folder and
// returns its filepath, otherwise it returns an error.
func RootPathFromWorkingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	for len(wd) > 1 {
		if !IsRootPath(wd) {
			wd = parentDirTo(wd)
			continue
		}

		return wd, nil
	}

	return "", fmt.Errorf("could not find Go module root")
}

// IsRootPath returns true if path is a module root filepath, otherwise false.
func IsRootPath(path string) bool {
	f, err := os.Open(path + "/" + "go.mod")
	if err != nil {
		return false
	}
	defer f.Close()
	return true
}

func parentDirTo(path string) string {
	return filepath.Dir(path)
}

// IsMainPkg returns true if the filepath given by path is a main Go package, otherwise false.
func IsMainPkg(path string) bool {
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalf("could not read path = %s: %s", path, err)
	}

	isMainPkg := false
	for _, fi := range fis {
		if fi.IsDir() || filepath.Ext(fi.Name()) != ".go" {
			continue
		}

		filepath := fmt.Sprintf("%s/%s", path, fi.Name())
		fset := token.NewFileSet()
		tokenizedFile, err := parser.ParseFile(fset, filepath, nil, 0)
		if err != nil {
			log.Fatalf("could not parse Go file = %s in module.IsMainPkg: %s", filepath, err)
		}

		astutil.Apply(tokenizedFile, func(c *astutil.Cursor) bool {
			n := c.Node()
			if n == nil {
				return true
			}

			if file, ok := n.(*ast.File); ok {
				if file.Name.Name == "main" {
					isMainPkg = true
				}

				return false
			}

			return true
		}, nil)
	}

	return isMainPkg
}

// ImportPathFrom returns the import path for path given the moduleName and module root path (root).
//
// Ex.
//	path = /Users/john/repos/github.com/doe/example/a/b/c
//	root = /Users/john/repos/github.com/doe/example
//	moduleName = github.com/lolzor/example
//
//	Returns = github.com/lolzor/example/a/b/c
func ImportPathFrom(path, moduleName, root string) string {
	return fmt.Sprintf("%s%s", moduleName, path[len(root):])
}
