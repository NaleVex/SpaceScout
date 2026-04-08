package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

type ScanStatus struct {
	CurrentFolder string
	TotalSize     int64
	ResultNode    *Node
}

type rootSelectModel struct {
	currentDir  string
	directories []string
	cursor      int
}

func initializeRootSelectModel() rootSelectModel {
	return rootSelectModel{
		currentDir:  "/mnt",
		directories: GetDisks(),
		cursor:      0,
	}
}

func getAvailableDirectories(path string) []string {
	entries, _ := os.ReadDir(path)
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs
}

func (m rootSelectModel) Init() tea.Cmd {
	return nil
}

func (m rootSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.directories) - 1
			}
		case "down", "j":
			m.cursor++
			m.cursor = m.cursor % len(m.directories)
		}
	}
	return m, nil
}

func (m rootSelectModel) View() tea.View {
	s := "Select a directory\n\n"
	for i, dir := range m.directories {
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor
		}
		s += fmt.Sprintf("%s %s\n", cursor, dir)
	}
	return tea.NewView(s)
}

func main() {
	p := tea.NewProgram(initializeRootSelectModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
