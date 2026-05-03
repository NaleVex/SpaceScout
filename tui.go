package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type model struct {
	currentMode     currentMode
	currentFolder   string
	totalSize       int64
	resultNode      *Node
	currentDir      string
	directories     []string
	cursor          int
	settings        settings
	sortMode        sortMode
	styles          *styles
	nodesViewing    []*Node
	viewingNodePath []*Node
	debug           string
	textInput       textinput.Model
	isEditing       bool
	isScanning      bool
	scanBreak       context.CancelFunc
	width           int
	height          int
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
	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.CharLimit = 10
	// ti.Width = 10
	return model{
		textInput:   ti,
		isEditing:   false,
		currentMode: diskSelectMode,
		currentDir:  "/mnt",
		directories: GetDisks(),
		cursor:      0,
		sortMode:    sortNameAs,
		styles:      newStyles(true),
		isScanning:  false,
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.BackgroundColorMsg:
		m.styles = newStyles(msg.IsDark())
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case scanTick:
		if m.resultNode != nil {
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
					m.runScan(selectedDir)
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
				case "p":
					m.cursor = 0
					m.currentMode = settingsMode
				case "enter", " ":
					if len(m.nodesViewing) == 0 {
						return m, nil
					} else {
						m.viewingNodePath = append(m.viewingNodePath, m.resultNode)
						selectedNode := m.nodesViewing[m.cursor]
						if selectedNode.IsDir {
							m.resultNode = selectedNode
							m.cursor = 0
						}
					}
				case "backspace":
					if len(m.viewingNodePath) == 0 {
						return m, nil
					} else {
						m.resultNode = m.viewingNodePath[len(m.viewingNodePath)-1]
						m.viewingNodePath = m.viewingNodePath[:len(m.viewingNodePath)-1]
						m.cursor = 0

					}
				case "tab":
					m.sortMode = (m.sortMode + 1) % 4

				case "s":
					if m.scanBreak != nil && m.isScanning {
						m.scanBreak()
						m.isScanning = false
						m.debug = "Scan stopped by user"
					}

					// case "r":
					// 	//reload application
					// 	e, _ := os.Executable()
					// 	exec.Command(e).Start()
					// 	os.Exit(0)
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
			switch m.sortMode {
			case sortNameAs:
				sort.Slice(keys, func(i, j int) bool {
					return keys[i] < keys[j]
				})
			case sortNameDes:
				sort.Slice(keys, func(i, j int) bool {
					return keys[i] > keys[j]
				})
			case sortSizeAs:
				sort.Slice(keys, func(i, j int) bool {
					return m.resultNode.Children[keys[i]].Size < m.resultNode.Children[keys[j]].Size
				})
			case sortSizeDes:
				sort.Slice(keys, func(i, j int) bool {
					return m.resultNode.Children[keys[i]].Size > m.resultNode.Children[keys[j]].Size
				})
			}
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
		if m.isEditing {
			switch msg := msg.(type) {
			case tea.KeyPressMsg:
				switch msg.String() {
				case "enter":
					val := m.textInput.Value()
					switch m.cursor {
					case 0: // sizeMin (int64)
						if i, err := strconv.ParseInt(val, 10, 64); err == nil {
							m.settings.sizeMin = i
						}
					case 1: // spacePercentMin (float64)
						if f, err := strconv.ParseFloat(val, 64); err == nil {
							m.settings.spacePercentMin = f
						}
					}
					m.isEditing = false
					m.textInput.Blur()
					return m, nil

				case "esc":
					m.isEditing = false
					m.textInput.Blur()
					return m, nil
				}
			}

			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < 2 {
					m.cursor++
				}
			case "enter", " ":
				if m.cursor == 2 {
					m.settings.foldersOnly = !m.settings.foldersOnly
				} else {
					m.isEditing = true
					m.textInput.Focus()
					if m.cursor == 0 {
						m.textInput.SetValue(fmt.Sprintf("%d", m.settings.sizeMin))
					} else {
						m.textInput.SetValue(fmt.Sprintf("%.2f", m.settings.spacePercentMin))
					}
				}
			case "p", "esc":
				m.cursor = 0
				m.currentMode = scanningMode
			}
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
	s += m.styles.hintStyle.Render("\n↑/↓ to navigate, enter to select, q to quit.")
	return s
}

func scanningView(m model) string {
	if m.resultNode == nil {
		return "Initializing scan..."
	}
	m.resultNode.mu.RLock()
	defer m.resultNode.mu.RUnlock()

	s := "\n"
	if m.isScanning {
		s += "Scanning "
		s += m.styles.hintStyle.Render("s to stop\n")
	}
	totalSize := float64(m.resultNode.Size) / (1024 * 1024)
	s += fmt.Sprintf("Current Folder %s:%s\n", m.currentFolder, m.resultNode.Path)
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
	t := ""
	if len(m.viewingNodePath) > 0 {
		t = " backspace - go back,"
	}
	s += m.styles.hintStyle.Render(fmt.Sprintf("\nenter - drill in folder,%s o - open folder, p - parameters, q - quit", t))
	// if m.debug != "" {
	// 	s += "\n\nDebug: " + m.debug
	// }
	return s
}

func settingsView(m model) string {
	s := " Settings\n\n"

	settingsItems := []string{
		"Minimum File Size (MB)",
		"Minimum Space (%)",
		"Folders Only",
	}

	for i, label := range settingsItems {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}
		valueStr := ""
		if m.cursor == i && m.isEditing {
			valueStr = m.textInput.View()
		} else {
			switch i {
			case 0:
				valueStr = fmt.Sprintf("%d MB", m.settings.sizeMin)
			case 1:
				valueStr = fmt.Sprintf("%.2f%%", m.settings.spacePercentMin)
			case 2:
				valueStr = fmt.Sprintf("%t", m.settings.foldersOnly)
			}
		}

		s += fmt.Sprintf("%s %-25s: %s\n", cursor, label, valueStr)
	}
	if m.isEditing {
		s += m.styles.hintStyle.Render("\n Enter to Save, Esc to Cancel")
	} else {
		s += m.styles.hintStyle.Render("\n Enter to Edit, P to Go Back")
	}

	return s
}

func main() {
	initialModel := initialModel()
	p := tea.NewProgram(initialModel)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func (m *model) runScan(path string) {
	m.resultNode = InitNode(path)
	m.isScanning = true
	ctx, cancel := context.WithCancel(context.Background())
	m.scanBreak = cancel
	go ScanDir(path, m, ctx)
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return scanTick{}
	})
}
