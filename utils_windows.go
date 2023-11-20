//go:build windows
// +build windows

package ishell

import (
	"github.com/liqianrain/readline"
)

func clearScreen(s *Shell) error {
	return readline.ClearScreen(s.writer)
}
