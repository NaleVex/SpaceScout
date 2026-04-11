package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"
)

type model struct {
	currentMode   currentMode
	currentFolder string
	totalSize     int64
	resultNode    *Node
	currentDir    string
	directories   []string
	cursor        int
	settings      settings
	sortMode      sortMode
	styles        *styles
	nodesViewing  []*Node
	debug         string
}

type settings struct {
	sizeMin         int64
	spacePercentMin float64
	foldersOnly     bool
}

type sortMode int

const (
	sortNameAs sortMode = iota
	sortNameDes
	sortSizeAs
	sortSizeDes
)

type currentMode int

const (
	diskSelectMode currentMode = iota
	scanningMode
	settingsMode
)

type scanTick struct{}

func initialModel() model {
	return model{
		currentMode: diskSelectMode,
		currentDir:  "/mnt",
		directories: GetDisks(),
		cursor:      0,
		sortMode:    sortNameAs,
		styles:      newStyles(true),
		settings: settings{
			sizeMin:         1,
			spacePercentMin: .2,
			foldersOnly:     false,
		},
	}
}

func (m model) Init() tea.Cmd {
	return nil
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.styles = newStyles(msg.IsDark())
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case scanTick:
		if m.resultNode != nil && !m.resultNode.IsDone {
			cmds = append(cmds, doTick())
		}
	}
	switch m.currentMode {
	case diskSelectMode:
		{
			switch msg := msg.(type) {
			case tea.KeyPressMsg:
				switch msg.String() {
				case "up", "k":
					m.cursor--
					if m.cursor < 0 {
						m.cursor = len(m.directories) - 1
					}
				case "down", "j":
					m.cursor++
					m.cursor = m.cursor % len(m.directories)
				case "enter":
					selectedDir := m.directories[m.cursor]
					m.currentMode = scanningMode
					m.runScan(nil, selectedDir)
					m.cursor = 0
					return m, doTick()
				}
			}
			return m, nil
		}
	case scanningMode:
		{
			switch msg := msg.(type) {
			case tea.KeyPressMsg:
				switch msg.String() {
				case "up", "k":
					m.cursor--
				case "down", "j":
					m.cursor++
				case "o":
					if len(m.nodesViewing) == 0 {
						return m, nil
					} else {
						openFolder(m.nodesViewing[m.cursor].Path)
					}
					m.debug = "Opening: " + m.nodesViewing[m.cursor].Path
				}

				l := len(m.nodesViewing)
				if l == 0 {
					m.cursor = 0
				} else {
					m.cursor = min(max(m.cursor, 0), l-1)
				}
				m.debug = fmt.Sprintf("cursor: %d, len: %d", m.cursor, l)
				// return m, nil
			}
			keys := make([]string, 0, len(m.resultNode.Children))
			for k := range m.resultNode.Children {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			m.nodesViewing = m.nodesViewing[:0]
			totalSize := float64(m.resultNode.Size) / (1024 * 1024)
			for _, name := range keys {
				node := m.resultNode.Children[name]
				if m.settings.foldersOnly && !node.IsDir {
					continue
				}
				if node.Size < m.settings.sizeMin {
					continue
				}
				sizeMB := float64(node.Size) / (1024 * 1024)
				spacePercent := (sizeMB / totalSize) * 100.0
				if spacePercent < m.settings.spacePercentMin {
					continue
				}

				m.nodesViewing = append(m.nodesViewing, node)
			}
			return m, tea.Batch(cmds...)
		}
	case settingsMode:
		{
			return m, nil
		}
	}
	return m, nil

}

func (m model) View() tea.View {
	var s string
	switch m.currentMode {
	case diskSelectMode:
		s += diskSelectView(m)
	case scanningMode:
		s += scanningView(m)
	case settingsMode:
		s += settingsView(m)
	}
	return tea.NewView(s)
}

func diskSelectView(m model) string {
	if len(m.directories) == 0 {
		return "No disks found."
	}
	s := "Select a disk to scan:\n\n"
	for i, dir := range m.directories {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, dir)
	}
	s += "\nUp/down arrows to navigate, enter to select, q to quit."
	return s
}

func scanningView(m model) string {
	if m.resultNode == nil {
		return "Initializing scan..."
	}
	m.resultNode.mu.RLock()
	defer m.resultNode.mu.RUnlock()

	s := "\n"
	if m.resultNode.IsDone {
		s += "✅ Scan completed\n"
	} else {
		s += "Scanning (updates every second)...\n"
	}
	totalSize := float64(m.resultNode.Size) / (1024 * 1024)

	s += fmt.Sprintf("Total Size: %.2f MB\n", totalSize)
	s += fmt.Sprintf("Sort Mode: %s %s\n\n", func() string {
		switch m.sortMode {
		case sortNameAs:
			return "Name Ascending"
		case sortNameDes:
			return "Name Descending"
		case sortSizeAs:
			return "Size Ascending"
		case sortSizeDes:
			return "Size Descending"
		default:
			return "Unknown"
		}
	}(), m.styles.hintStyle.Render("tab"))
	s += fmt.Sprintf(" %-30s | %-12s | %10s\n", "Folder/File Name", "Size", "Space %")
	s += "----------------------------------------------\n"

	for i, node := range m.nodesViewing {
		sizeMB := float64(node.Size) / (1024 * 1024)
		spacePercent := (sizeMB / totalSize) * 100.0
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s%-30.30s | %12.2f MB | %3.2f%%\n", cursor, node.Name, sizeMB, spacePercent)
	}
	s += m.styles.hintStyle.Render("\nenter - drill in folder, o - open folder, p - parameters, r - start new scan, q - quit")
	if m.debug != "" {
		s += "\n\nDebug: " + m.debug
	}
	return s
}

func settingsView(m model) string {

	return "Settings... (not implemented yet)"
}

func main() {
	initialModel := initialModel()
	p := tea.NewProgram(initialModel)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func (m *model) runScan(p *tea.Program, path string) {
	m.resultNode = InitNode(path)
	go ScanDir(path, m.resultNode)
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return scanTick{}
	})
}
