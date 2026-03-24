package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type result struct {
	name string
	size int64
}

type NodeOld struct {
	Name     string
	Path     string
	Size     int64
	IsDir    bool
	Children map[string]*NodeOld
}

func printTree(node *NodeOld) {
	fmt.Printf("%s (%.2f MB)\n", node.Name, float64(node.Size)/(1024*1024))
	for _, child := range node.Children {
		fmt.Printf("  %-20.20s | (%.2f MB)\n", child.Name, float64(child.Size)/(1024*1024))
	}
}

func mainOld() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path>")
		return
	}
	rootPath := filepath.Clean(os.Args[1]) // Use your path here

	// 1. Setup the Root
	root := &NodeOld{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		IsDir:    true,
		Children: make(map[string]*NodeOld),
	}

	entries, _ := os.ReadDir(rootPath)

	results := make(chan *NodeOld)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	tStart := time.Now()
	// 2. Launch workers for each top-level entry
	for _, entry := range entries {
		if !entry.IsDir() {
			// Handle files in the root directly
			info, _ := entry.Info()
			root.Size += info.Size()
			root.Children[entry.Name()] = &NodeOld{Name: entry.Name(), Path: filepath.Join(rootPath, entry.Name()), Size: info.Size()}
			continue
		}

		wg.Add(1)
		go func(e fs.DirEntry) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			path := filepath.Join(rootPath, e.Name())
			// Each worker builds its own private sub-tree
			subTree := scanToTree(path)
			results <- subTree
		}(entry)
	}

	// 3. Closer goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// 4. Collector: Main thread safely attaches sub-trees
	for subTree := range results {
		root.Children[subTree.Name] = subTree
		root.Size += subTree.Size
		// fmt.Printf("Finished scanning: %s (%.2f MB)\n", subTree.Name, float64(subTree.Size)/1024/1024)
	}
	printTree(root)
	fmt.Printf("\nTOTAL SIZE: %.2f GB\n", float64(root.Size)/1024/1024/1024)
	fmt.Printf("Scan completed in %s\n", time.Since(tStart))
}

// scanToTree is the same logic as before, but it's "safe" because
// it only works on one directory branch at a time.
func scanToTree(rootPath string) *NodeOld {
	rootPath = filepath.Clean(rootPath)
	root := &NodeOld{
		Name:     filepath.Base(rootPath),
		Path:     rootPath,
		IsDir:    true,
		Children: make(map[string]*NodeOld),
	}

	nodes := make(map[string]*NodeOld)
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
		newNode := &NodeOld{
			Name:     d.Name(),
			Path:     path,
			Size:     0,
			IsDir:    d.IsDir(),
			Children: make(map[string]*NodeOld),
		}

		if !d.IsDir() {
			size := info.Size()
			newNode.Size = size
			// Bubble up size within THIS private sub-tree
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
