package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Node struct {
	Name     string
	Path     string
	Size     int64
	IsDir    bool
	Children map[string]*Node
}

func ScanDir(dirPath string) Node {
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
	tStart := time.Now()
	entries, _ := os.ReadDir(rootPath)
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			root.Size += info.Size()
			root.Children[entry.Name()] = &Node{Name: entry.Name(), Path: filepath.Join(rootPath, entry.Name()), Size: info.Size()}
			continue
		}

		wg.Add(1)
		go func(entry os.DirEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			path := filepath.Join(rootPath, entry.Name())
			node := scanToTree(path)
			results <- node
		}(entry)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	for node := range results {
		root.Children[node.Name] = node
		root.Size += node.Size
	}
	fmt.Printf("Scan completed in %s", time.Since(tStart))
	return *root
}

func scanToTree(rootPath string) *Node {
	fmt.Printf("Scanning %s \n", rootPath)
	rootPath = filepath.Clean(rootPath)
	root := &Node{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		IsDir:    true,
		Children: make(map[string]*Node),
	}
	nodes := make(map[string]*Node)
	nodes[rootPath] = root

	_ = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		path = filepath.Clean(path)
		if path == rootPath {
			return nil
		}
		parentPath := filepath.Dir(path)
		parentNode, exists := nodes[parentPath]
		if !exists || parentNode == nil {
			return nil
		}
		info, _ := d.Info()
		newNode := &Node{
			Name:     d.Name(),
			Path:     path,
			Size:     0,
			IsDir:    d.IsDir(),
			Children: make(map[string]*Node),
		}
		if !d.IsDir() {
			size := info.Size()
			newNode.Size = size
			currPath := parentPath
			for {
				if n, ok := nodes[currPath]; ok {
					n.Size += size
				}
				if currPath == rootPath {
					break
				}
				currPath = filepath.Dir(currPath)
			}
		} else {
			nodes[path] = newNode
		}
		parentNode.Children[d.Name()] = newNode
		return nil
	})
	return root
}

func StartScan() {
	result := ScanDir("G:/")
	for name, node := range result.Children {
		if node.IsDir {
			fmt.Printf("Directory: %s, Size: %.2f MB\n", name, float64(node.Size)/(1024*1024))
		} else {
			fmt.Printf("File: %s, Size: %.2f MB\n", name, float64(node.Size)/(1024*1024))
		}
	}
}
