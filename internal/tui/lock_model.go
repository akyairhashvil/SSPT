package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
)

type LockModel struct {
	Locked          bool
	Message         string
	PassphraseHash  string
	PassphraseInput textinput.Model
	LastInput       time.Time
	AutoLockAfter   time.Duration
	Attempts        int
	LockUntil       time.Time
}

func NewLockModel(autoLockAfter time.Duration, input textinput.Model) LockModel {
	return LockModel{
		PassphraseInput: input,
		AutoLockAfter:   autoLockAfter,
		LastInput:       time.Now(),
	}
}
