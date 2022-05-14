package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var nameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
var doneStyle = lipgloss.NewStyle().Margin(1, 2).Foreground(lipgloss.Color("201"))
var checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")

type model struct {
	doneq chan *string
	nTodo int
	nDone int
	done  bool

	width    int
	height   int
	spinner  spinner.Model
	progress progress.Model
}

type tickProgress struct {
	msg string
}

func newModel(doneq chan *string, nTodo int) model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	return model{
		doneq:    doneq,
		nTodo:    nTodo,
		nDone:    0,
		spinner:  s,
		progress: progress.New(progress.WithDefaultGradient(), progress.WithWidth(40)),
	}
}

func (m model) readProgress() tea.Msg {
	return tickProgress{*<-m.doneq}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.readProgress)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		if newModel, ok := newModel.(progress.Model); ok {
			m.progress = newModel
		}
		return m, cmd
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tickProgress:
		m.nDone++
		if m.nDone >= m.nTodo {
			m.done = true
			return m, tea.Sequence(tea.Quit)
		}
		return m, tea.Batch(
			m.progress.SetPercent(float64(m.nDone)/float64(m.nTodo)),
			tea.Printf("%s %s", checkMark, nameStyle.Render(msg.msg)),
			m.readProgress,
		)
	}
	return m, nil
}

func (m model) View() string {
	if m.done {
		return doneStyle.Render(fmt.Sprintf("Done! Processed %d images.\n", m.nDone))
	}

	spin := m.spinner.View() + " "
	prog := m.progress.View()
	cellsAvail := max(0, m.width-lipgloss.Width(spin+prog))
	info := lipgloss.NewStyle().MaxWidth(cellsAvail).Render("Processing ")
	cellsRemaining := max(0, m.width-lipgloss.Width(spin+info+prog))
	gap := strings.Repeat(" ", cellsRemaining)

	return spin + info + gap + prog
}
