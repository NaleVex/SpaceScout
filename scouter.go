package main

import (
	"context"
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

func ScanDir(dirPath string, m *model, ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	rootPath := filepath.Clean(dirPath)
	results := make(chan *Node)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	entries, _ := os.ReadDir(rootPath)
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()

			m.resultNode.mu.Lock()
			m.resultNode.Size += info.Size()
			m.resultNode.Children[entry.Name()] = &Node{
				Name: entry.Name(),
				Path: filepath.Join(rootPath, entry.Name()),
				Size: info.Size(),
			}
			m.resultNode.mu.Unlock()

			continue
		}

		wg.Add(1)
		go func(entry os.DirEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			path := filepath.Join(rootPath, entry.Name())
			node := scanToTree(path, ctx)
			results <- node
		}(entry)
		// root.mu.Unlock()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	for node := range results {
		m.resultNode.mu.Lock()
		m.resultNode.Children[node.Name] = node
		m.resultNode.Size += node.Size
		m.resultNode.mu.Unlock()
	}

	m.resultNode.mu.Lock()
	m.isScanning = false
	m.isScanCompleted = true
	m.resultNode.mu.Unlock()

}

func scanToTree(rootPath string, ctx context.Context) *Node {
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
		select {
		case <-ctx.Done(): // Non-blocking check for cancellation
			return filepath.SkipAll
		default:
			// Continue working
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
