package main

import (
	"path/filepath"
	"sync"
)

type Node struct {
	Name     string
	Path     string
	Size     int64
	IsDir    bool
	Children map[string]*Node
}

func ScanDir(dirPath string) {
	rootPath := filepath.Clean(dirPath)
	root := &Node{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		IsDir:    true,
		Children: make(map[string]*Node),
	}

	results := make(chan *Node)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
}

func scanPath(path string, node *Node, wg *sync.WaitGroup, sem chan struct{}, results chan *Node) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()
}
