package database

import (
	"fmt"
	"strings"
)

type GoalQuery struct {
	columns string
	filters []string
	args    []interface{}
	orderBy string
	limit   int
}

func NewGoalQuery() *GoalQuery {
	return &GoalQuery{columns: goalColumnsWithSprint}
}

func (q *GoalQuery) Where(filter string, args ...interface{}) *GoalQuery {
	q.filters = append(q.filters, filter)
	q.args = append(q.args, args...)
	return q
}

func (q *GoalQuery) WhereBacklog() *GoalQuery {
	return q.Where("sprint_id IS NULL")
}

func (q *GoalQuery) WhereSprint(sprintID int64) *GoalQuery {
	return q.Where("sprint_id = ?", sprintID)
}

func (q *GoalQuery) WhereWorkspace(workspaceID int64) *GoalQuery {
	return q.Where("workspace_id = ?", workspaceID)
}

func (q *GoalQuery) OrderBy(orderBy string) *GoalQuery {
	q.orderBy = orderBy
	return q
}

func (q *GoalQuery) Limit(limit int) *GoalQuery {
	q.limit = limit
	return q
}

func (q *GoalQuery) Build() (string, []interface{}) {
	query := fmt.Sprintf("SELECT %s FROM goals", q.columns)
	if len(q.filters) > 0 {
		query += " WHERE " + strings.Join(q.filters, " AND ")
	}
	if q.orderBy != "" {
		query += " ORDER BY " + q.orderBy
	}
	if q.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", q.limit)
	}
	return query, q.args
}
