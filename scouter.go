package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type Node struct {
	Name     string
	Path     string
	Size     int64
	IsDir    bool
	Children map[string]*Node
	IsDone   bool
	mu       sync.RWMutex
}

func InitNode(rootPath string) *Node {
	root := &Node{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		IsDir:    true,
		Children: make(map[string]*Node),
	}
	return root
}

func ScanDir(dirPath string, root *Node) {
	rootPath := filepath.Clean(dirPath)
	results := make(chan *Node)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	entries, _ := os.ReadDir(rootPath)
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()

			// --- LOCK HERE ---
			root.mu.Lock()
			root.Size += info.Size()
			root.Children[entry.Name()] = &Node{
				Name: entry.Name(),
				Path: filepath.Join(rootPath, entry.Name()),
				Size: info.Size(),
			}
			root.mu.Unlock()
			// -----------------

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
		// root.mu.Unlock()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	for node := range results {
		// --- LOCK HERE ---
		// We lock every time a sub-folder finishes so the TUI
		// can safely read the new state.
		root.mu.Lock()
		root.Children[node.Name] = node
		root.Size += node.Size
		root.mu.Unlock()
		// -----------------
	}

	// --- LOCK HERE ---
	root.mu.Lock()
	root.IsDone = true
	root.mu.Unlock()
	// -----------------

}

func scanToTree(rootPath string) *Node {
	// fmt.Printf("Scanning %s \n", rootPath)
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
