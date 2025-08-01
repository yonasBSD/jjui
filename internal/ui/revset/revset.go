package revset

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/common/autocompletion"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

type EditRevSetMsg struct {
	Clear bool
}

type Model struct {
	Editing         bool
	Value           string
	autoComplete    *autocompletion.AutoCompletionInput
	keymap          keymap
	History         []string
	historyIndex    int
	currentInput    string
	historyActive   bool
	MaxHistoryItems int
	context         *appContext.MainContext
	styles          styles
	width, height   int
}

func (m *Model) Width() int {
	return m.width
}

func (m *Model) Height() int {
	return m.height
}

func (m *Model) SetWidth(w int) {
	m.width = w
}

func (m *Model) SetHeight(h int) {
	m.height = h
}

type styles struct {
	promptStyle lipgloss.Style
	textStyle   lipgloss.Style
}

func (m *Model) IsFocused() bool {
	return m.Editing
}

type keymap struct{}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "complete")),
		key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next")),
		key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
	}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

func New(context *appContext.MainContext) *Model {
	styles := styles{
		promptStyle: common.DefaultPalette.Get("revset title"),
		textStyle:   common.DefaultPalette.Get("revset text"),
	}

	revsetAliases := context.JJConfig.RevsetAliases
	completionProvider := NewCompletionProvider(revsetAliases)
	autoComplete := autocompletion.New(completionProvider, autocompletion.WithStylePrefix("revset"))

	autoComplete.SetValue(context.DefaultRevset)
	autoComplete.Focus()

	return &Model{
		context:         context,
		Editing:         false,
		Value:           context.CurrentRevset,
		keymap:          keymap{},
		autoComplete:    autoComplete,
		History:         []string{},
		historyIndex:    -1,
		MaxHistoryItems: 50,
		styles:          styles,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) AddToHistory(input string) {
	if input == "" {
		return
	}

	for i, item := range m.History {
		if item == input {
			m.History = append(m.History[:i], m.History[i+1:]...)
			break
		}
	}

	m.History = append([]string{input}, m.History...)

	if len(m.History) > m.MaxHistoryItems && m.MaxHistoryItems > 0 {
		m.History = m.History[:m.MaxHistoryItems]
	}

	m.historyIndex = -1
	m.historyActive = false
}

func (m *Model) SetHistory(history []string) {
	m.History = history
	m.historyIndex = -1
	m.historyActive = false
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Editing {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Editing = false
			m.autoComplete.Blur()
			return m, nil
		case tea.KeyEnter:
			m.Editing = false
			m.autoComplete.Blur()
			m.Value = m.autoComplete.Value()
			m.AddToHistory(m.Value)
			if m.Value == "" {
				m.Value = m.context.DefaultRevset
			}
			return m, tea.Batch(common.Close, common.UpdateRevSet(m.Value))
		case tea.KeyUp:
			if len(m.History) > 0 {
				if !m.historyActive {
					m.currentInput = m.autoComplete.Value()
					m.historyActive = true
				}

				if m.historyIndex < len(m.History)-1 {
					m.historyIndex++
					m.autoComplete.SetValue(m.History[m.historyIndex])
					m.autoComplete.CursorEnd()
				}
				return m, nil
			}
		case tea.KeyDown:
			if m.historyActive {
				if m.historyIndex > 0 {
					m.historyIndex--
					m.autoComplete.SetValue(m.History[m.historyIndex])
				} else {
					m.historyIndex = -1
					m.historyActive = false
					m.autoComplete.SetValue(m.currentInput)
				}
				m.autoComplete.CursorEnd()
				return m, nil
			}
		}
	case common.UpdateRevSetMsg:
		m.Value = string(msg)
		if m.Editing {
			m.Editing = false
			m.AddToHistory(m.Value)
		}
	case EditRevSetMsg:
		m.Editing = true
		m.autoComplete.Focus()
		if msg.Clear {
			m.autoComplete.SetValue("")
		}
		m.historyActive = false
		m.historyIndex = -1
		return m, m.autoComplete.Init()
	}

	var cmd tea.Cmd
	m.autoComplete, cmd = m.autoComplete.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	var w strings.Builder
	w.WriteString(m.styles.promptStyle.PaddingRight(1).Render("revset:"))
	if m.Editing {
		w.WriteString(m.autoComplete.View())
	} else {
		revset := "(default)"
		if m.Value != "" {
			revset = m.Value
		}
		w.WriteString(m.styles.textStyle.Render(revset))
	}
	return lipgloss.Place(m.width, m.height, 0, 0, w.String(), lipgloss.WithWhitespaceBackground(m.styles.textStyle.GetBackground()))
}
