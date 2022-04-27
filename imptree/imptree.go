// Copyright 2022 tobbstr. All rights reserved.
// Use of this source code is governed by a MIT-
// license that can be found in the LICENSE file.
package imptree

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

// MatchPkg is a predicate function for selecting which Go packages to include in the tree
type MatchPkg func(*packages.Package) bool

// Node represents a Go source package
type Node struct {
	// Children are links to imported packages
	Children []*Node
	// Parents are links to packages that import this package
	Parents []*Node
	// PkgPath is the import path to this package
	PkgPath string
}

// Builder is a tree builder. A Builder should not be reused for different trees, instead a new Builder should
// be instantiated.
type Builder struct {
	// nodes maps import paths to Nodes
	nodes map[string]*Node
	// loadPkgs is a hook that allows for testing.
	// See https://pkg.go.dev/golang.org/x/tools/go/packages#Load for details regarding its actual
	// implementation.
	loadPkgs func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error)
	// printLoadPkgsErrors is a hook that allows for testing.
	// See https://pkg.go.dev/golang.org/x/tools/go/packages#PrintErrors for details regarding its actual
	// implementation.
	printLoadPkgsErrors func(pkgs []*packages.Package) int
}

// NewBuilder constructs an initialized Builder
func NewBuilder() *Builder {
	return &Builder{
		nodes:               make(map[string]*Node),
		loadPkgs:            packages.Load,
		printLoadPkgsErrors: packages.PrintErrors,
	}
}

// Build builds and returns a doubly-linked tree of import paths, so it's possible to see which packages are
// imported by a package (its children) and also which packages import a package (its parents). The tree's
// root package is given by importPath. Only packages matched by matchPkg are included in the tree.
//
// Ex.
//	builder.Build("github.com/johndoe/example/cmd/acme", func(pkg *package.Package) bool{
//		// includes only packages belonging to the same module
//		if strings.Contains(pkg.PkgPath, "github.com/johndoe/example") {
//			return true
//		}
//		return false
//	})
func (b *Builder) Build(importPath string, matchPkg MatchPkg) (*Node, error) {
	cfg := &packages.Config{}
	// Bypass default vendor mode, as we need a package not available in the
	// std module vendor folder.
	cfg.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	cfg.Mode = packages.NeedImports | packages.NeedName

	// pkgs, err := packages.Load(cfg, importPath)
	pkgs, err := b.loadPkgs(cfg, importPath)
	if err != nil {
		return nil, err
	}
	// if packages.PrintErrors(pkgs) > 0 || len(pkgs) != 1 {
	if b.printLoadPkgsErrors(pkgs) > 0 || len(pkgs) != 1 {
		return nil, fmt.Errorf("failed to load source package")
	}
	pkg := pkgs[0]

	// build tree
	b.buildTree(pkg, matchPkg)

	// find tree root node and return it
	for _, node := range b.nodes {
		for node.Parents != nil {
			node = node.Parents[0]
		}

		return node, nil
	}

	return nil, fmt.Errorf("could not find tree root node")
}

func (b *Builder) buildTree(pkg *packages.Package, matchPkg MatchPkg) {
	if !matchPkg(pkg) {
		return
	}

	var node *Node
	if n, ok := b.nodes[pkg.PkgPath]; ok {
		node = n
	} else {
		node = &Node{PkgPath: pkg.PkgPath}
		b.nodes[node.PkgPath] = node
	}

	for importPath, childPkg := range pkg.Imports {
		if !matchPkg(childPkg) {
			continue
		}

		var childNode *Node
		if cn, ok := b.nodes[importPath]; ok {
			childNode = cn
		} else {
			childNode = &Node{PkgPath: importPath}
			b.nodes[importPath] = childNode
		}

		if !containsNode(childNode.Parents, node) {
			childNode.Parents = append(childNode.Parents, node)
		}

		if !containsNode(node.Children, childNode) {
			node.Children = append(node.Children, childNode)
		}

		b.buildTree(childPkg, matchPkg)
	}
}

func containsNode(slc []*Node, node *Node) bool {
	if len(slc) == 0 || node == nil {
		return false
	}

	for i := 0; i < len(slc); i++ {
		if slc[i] == node {
			return true
		}
	}

	return false
}
