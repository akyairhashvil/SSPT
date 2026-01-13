package tui

import (
	"log"

	"github.com/akyairhashvil/SSPT/internal/models"
)

const (
	goalTreeWarnDepth       = 20
	goalTreeMaxDepthDefault = 100
)

// BuildHierarchy organizes a flat list of goals into a tree structure based on ParentID.
func BuildHierarchy(flatGoals []models.Goal) []GoalView {
	goalMap := make(map[int64]*GoalView)
	nodes := make([]GoalView, len(flatGoals))

	for i := range flatGoals {
		nodes[i] = GoalView{Goal: flatGoals[i]}
		goalMap[nodes[i].ID] = &nodes[i]
	}

	var rootPtrs []*GoalView

	// Build Tree
	for i := range nodes {
		node := &nodes[i]
		if node.ParentID != nil {
			if parent, ok := goalMap[*node.ParentID]; ok {
				parent.Subtasks = append(parent.Subtasks, *node)
			} else {
				// Parent not found in this context (e.g. parent is in another sprint or completed)
				// Treat as root for this view.
				rootPtrs = append(rootPtrs, node)
			}
		} else {
			rootPtrs = append(rootPtrs, node)
		}
	}

	var finalRoots []GoalView
	for _, ptr := range rootPtrs {
		finalRoots = append(finalRoots, *ptr)
	}

	return finalRoots
}

// Flatten converts a hierarchical tree into a flat list for rendering, respecting expansion state.
func Flatten(goals []GoalView, level int, expandedMap map[int64]bool, maxDepth int) []GoalView {
	if maxDepth <= 0 {
		maxDepth = goalTreeMaxDepthDefault
	}
	warned := false
	return flatten(goals, level, expandedMap, maxDepth, &warned)
}

func flatten(goals []GoalView, level int, expandedMap map[int64]bool, maxDepth int, warned *bool) []GoalView {
	var out []GoalView
	for _, g := range goals {
		if level >= maxDepth {
			if !*warned {
				log.Printf("goal tree depth exceeds %d; truncating deeper nodes", goalTreeWarnDepth)
				*warned = true
			}
			break
		}
		if level >= goalTreeWarnDepth && !*warned {
			log.Printf("goal tree depth exceeds %d; truncating deeper nodes", goalTreeWarnDepth)
			*warned = true
		}
		g.Level = level
		if expandedMap != nil {
			g.Expanded = expandedMap[g.ID]
		} else {
			g.Expanded = true // Default to expanded if no map provided (e.g. reports)
		}
		out = append(out, g)
		if g.Expanded && len(g.Subtasks) > 0 {
			out = append(out, flatten(g.Subtasks, level+1, expandedMap, maxDepth, warned)...)
		}
	}
	return out
}
