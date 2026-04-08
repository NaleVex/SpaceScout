package main

import (
	"fmt"
	"testing"
)

func TestScanDir(t *testing.T) {
	result := ScanDir("G:/")
	for name, node := range result.Children {
		if node.IsDir {
			t.Logf("Directory: %s, Size: %.2f MB", name, float64(node.Size)/(1024*1024))
			fmt.Printf("Directory: %s, Size: %.2f MB \n", name, float64(node.Size)/(1024*1024))
		} else {
			t.Logf("File: %s, Size: %.2f MB", name, float64(node.Size)/(1024*1024))
			fmt.Printf("File: %s, Size: %.2f MB \n", name, float64(node.Size)/(1024*1024))
		}
	}
}
