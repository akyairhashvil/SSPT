package tui

import "github.com/akyairhashvil/SSPT/internal/models"

// BuildHierarchy organizes a flat list of goals into a tree structure based on ParentID.
func BuildHierarchy(flatGoals []models.Goal) []models.Goal {
	goalMap := make(map[int64]*models.Goal)
	
	// Use a slice of pointers to track the nodes we are mutating
	nodes := make([]*models.Goal, len(flatGoals))
	
	// Initialize map. IMPORTANT: Use &flatGoals[i] to get stable pointers to the backing array.
	// Using a loop variable 'g' and taking '&g' is risky in older Go versions and less clear.
	for i := range flatGoals {
		nodes[i] = &flatGoals[i]
		goalMap[flatGoals[i].ID] = nodes[i]
	}

	var rootPtrs []*models.Goal

	// Build Tree
	for _, node := range nodes {
		if node.ParentID.Valid {
			if parent, ok := goalMap[node.ParentID.Int64]; ok {
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
	
	// Convert pointers back to values for the return slice
	// Note: The values in 'rootPtrs' have their 'Subtasks' fields populated because they point
	// to the same memory locations as 'nodes' which were mutated via 'goalMap'.
	var finalRoots []models.Goal
	for _, ptr := range rootPtrs {
		finalRoots = append(finalRoots, *ptr)
	}
	
	return finalRoots
}

// Flatten converts a hierarchical tree into a flat list for rendering, respecting expansion state.
func Flatten(goals []models.Goal, level int, expandedMap map[int64]bool) []models.Goal {
	var out []models.Goal
	for _, g := range goals {
		g.Level = level
		if expandedMap != nil {
			g.Expanded = expandedMap[g.ID]
		} else {
			g.Expanded = true // Default to expanded if no map provided (e.g. reports)
		}
		
		out = append(out, g)
		
		if g.Expanded && len(g.Subtasks) > 0 {
			out = append(out, Flatten(g.Subtasks, level+1, expandedMap)...)
		}
	}
	return out
}
