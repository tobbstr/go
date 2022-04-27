// Copyright 2022 tobbstr. All rights reserved.
// Use of this source code is governed by a MIT-
// license that can be found in the LICENSE file.
package imptree

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestBuilder_Build(t *testing.T) {
	nodeAA := &Node{
		PkgPath: "a/a",
	}

	nodeD := &Node{
		PkgPath: "d",
	}

	nodeC := &Node{
		PkgPath:  "c",
		Children: []*Node{nodeD, nodeAA},
	}
	nodeD.Parents = []*Node{nodeC}

	nodeA := &Node{
		PkgPath:  "a",
		Children: []*Node{nodeAA, nodeC},
	}
	nodeAA.Parents = []*Node{nodeA, nodeC}
	nodeC.Parents = []*Node{nodeA}

	type fields struct {
		nodes               map[string]*Node
		loadPkgs            func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error)
		printLoadPkgsErrors func(pkgs []*packages.Package) int
	}
	type args struct {
		importPath string
		matchPkg   MatchPkg
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Node
		wantErr bool
	}{
		{
			name: "should return error when loadPkgs fails",
			fields: fields{
				loadPkgs: func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
					return nil, fmt.Errorf("error")
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return error when printLoadPkgsErrors returns 1",
			fields: fields{
				loadPkgs: func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
					return nil, nil
				},
				printLoadPkgsErrors: func(pkgs []*packages.Package) int {
					return 1
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return error when len(pkgs) is 0",
			fields: fields{
				loadPkgs: func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
					return nil, nil
				},
				printLoadPkgsErrors: func(pkgs []*packages.Package) int {
					return 0
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return error when can't find tree root node",
			fields: fields{
				loadPkgs: func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
					return []*packages.Package{
						{ID: "id"},
					}, nil
				},
				printLoadPkgsErrors: func(pkgs []*packages.Package) int {
					return 0
				},
			},
			args: args{
				matchPkg: func(p *packages.Package) bool {
					return p.ID == "another id"
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should return tree for happy path",
			fields: fields{
				nodes: make(map[string]*Node),
				loadPkgs: func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
					return []*packages.Package{
						{
							PkgPath: "a",
							Imports: map[string]*packages.Package{
								"a/a": {
									PkgPath: "a/a",
								},
								"c": {
									PkgPath: "c",
									Imports: map[string]*packages.Package{
										"d": {
											PkgPath: "d",
										},
										"a/a": {
											PkgPath: "a/a",
										},
										"e": {
											PkgPath: "e",
										},
									},
								},
							},
						},
					}, nil
				},
				printLoadPkgsErrors: func(pkgs []*packages.Package) int {
					return 0
				},
			},
			args: args{
				matchPkg: func(p *packages.Package) bool {
					switch p.PkgPath {
					case "a", "a/a", "c", "d":
						return true
					}
					return false
				},
			},
			want:    nodeA,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			require := require.New(t)
			b := &Builder{
				nodes:               tt.fields.nodes,
				loadPkgs:            tt.fields.loadPkgs,
				printLoadPkgsErrors: tt.fields.printLoadPkgsErrors,
			}

			// When
			got, err := b.Build(tt.args.importPath, tt.args.matchPkg)

			// Then
			if tt.wantErr {
				require.Error(err)
				require.Nil(got)
				return
			}
			require.NoError(err)

			require.Equal(tt.want, got)
		})
	}
}

func TestNewBuilder(t *testing.T) {
	require := require.New(t)

	// When
	got := NewBuilder()

	// Then
	require.NotNil(got.loadPkgs)
	require.NotNil(got.printLoadPkgsErrors)
	require.NotNil(got.nodes)
	require.Empty(got.nodes)
}
