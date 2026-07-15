package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type isoNode struct {
	name      string
	isoID     string
	isDir     bool
	size      int64
	srcPath   string
	children  []*isoNode
	lba       uint32
	sectors   uint32
	pathNum   int
	parentNum int
}

func sanitizeName(name string, isDir bool) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	s := b.String()
	if s == "" {
		s = "_"
	}
	if isDir {
		if len(s) > 31 {
			s = s[:31]
		}
		return s
	}
	if !strings.Contains(s, ".") {
		s = s + "."
	}
	if len(s) > 28 {
		s = s[:28]
	}
	return s + ";1"
}

func buildTree(path string, name string) (*isoNode, error) {
	node := &isoNode{name: name, isDir: true, srcPath: path}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		full := filepath.Join(path, e.Name())
		if e.IsDir() {
			child, err := buildTree(full, e.Name())
			if err != nil {
				return nil, err
			}
			child.isoID = sanitizeName(e.Name(), true)
			node.children = append(node.children, child)
		} else if e.Type().IsRegular() {
			info, err := e.Info()
			if err != nil {
				return nil, err
			}
			child := &isoNode{name: e.Name(), isDir: false, srcPath: full, size: info.Size()}
			child.isoID = sanitizeName(e.Name(), false)
			node.children = append(node.children, child)
		}
	}
	sort.Slice(node.children, func(i, j int) bool {
		return node.children[i].isoID < node.children[j].isoID
	})
	return node, nil
}
