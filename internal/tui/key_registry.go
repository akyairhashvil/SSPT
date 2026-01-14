package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type KeyBinding struct {
	Key         string
	Handler     KeyHandler
	Description string
	ViewModes   []int
	Priority    int
}

func (b KeyBinding) AppliesToView(mode int) bool {
	if len(b.ViewModes) == 0 {
		return true
	}
	for _, v := range b.ViewModes {
		if v == mode {
			return true
		}
	}
	return false
}

type HandlerRegistry struct {
	bindings []KeyBinding
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{}
}

func (r *HandlerRegistry) Register(b KeyBinding) {
	r.bindings = append(r.bindings, b)
	sort.SliceStable(r.bindings, func(i, j int) bool {
		return r.bindings[i].Priority > r.bindings[j].Priority
	})
}

func (r *HandlerRegistry) Handle(m DashboardModel, key string) (DashboardModel, tea.Cmd, bool) {
	for _, b := range r.bindings {
		if b.Key == key && b.AppliesToView(m.viewMode) {
			next, cmd, handled := b.Handler(m, key)
			if handled {
				return next, cmd, true
			}
		}
	}
	return m, nil, false
}

func (r *HandlerRegistry) GetBindingsForView(mode int) []KeyBinding {
	var out []KeyBinding
	for _, b := range r.bindings {
		if b.AppliesToView(mode) {
			out = append(out, b)
		}
	}
	return out
}

func (r *HandlerRegistry) HelpForView(mode int) string {
	bindings := r.GetBindingsForView(mode)
	seen := make(map[string]bool)
	var parts []string
	for _, b := range bindings {
		if b.Description == "" {
			continue
		}
		if seen[b.Key] {
			continue
		}
		seen[b.Key] = true
		parts = append(parts, "["+b.Key+"]"+b.Description)
	}
	return strings.Join(parts, "|")
}
