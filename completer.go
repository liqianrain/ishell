package ishell

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flynn-archive/go-shlex"
)

type (
	iCompleter struct {
		shell    *Shell
		cmd      *Cmd
		disabled func() bool
	}
	Suggestion struct {
		Word     string
		Param    bool   // param in command path or param for command
		Optional bool   // optional argument
		Help     string // help msg
	}
)

type suggestionSorter []Suggestion

func (s suggestionSorter) Len() int {
	return len(s)
}

func (s suggestionSorter) Less(i, j int) bool {
	return s[i].Word < s[j].Word
}

func (s suggestionSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (ic iCompleter) Do(line []rune, pos int) (newLine [][]rune, length int, offset int) {
	if ic.disabled != nil && ic.disabled() {
		return nil, 0, len(line)
	}
	var words []string
	if w, err := shlex.Split(string(line)); err == nil {
		words = w
	} else {
		// fall back
		words = strings.Fields(string(line))
	}

	var cWords []Suggestion

	prefix := ""
	if len(words) > 0 && pos > 0 && line[pos-1] != ' ' {
		prefix = words[len(words)-1]
		cWords = ic.getWords(prefix, words[:len(words)-1])
	} else {
		cWords = ic.getWords(prefix, words)
	}

	var suggestions [][]rune

	var tips []string

	hasParam := false
	for _, w := range cWords {
		if w.Param {
			hasParam = true
		}

		leftBracket := ""
		righeBracket := ""
		if w.Optional && !w.Param {
			leftBracket = "["
			righeBracket = "]"
		}
		leftAngle := ""
		rightAngel := ""
		if w.Param {
			leftAngle = "<"
			rightAngel = ">"
		}

		tip := fmt.Sprintf("%s%s%s%s%s",
			leftBracket, leftAngle, w.Word, rightAngel, righeBracket)

		tips = append(tips, fmt.Sprintf("%-15s %s", tip, w.Help))

		if !w.Param && strings.HasPrefix(w.Word, prefix) {
			suggestions = append(suggestions, []rune(strings.TrimPrefix(w.Word, prefix)))
		}
	}
	if len(suggestions) == 1 && prefix != "" && string(suggestions[0]) == "" {
		suggestions = [][]rune{[]rune(" ")}
		hasParam = false
	}

	length = len(suggestions)
	if hasParam {
		length += 1
	}

	if length > 1 || hasParam {
		for i, tip := range tips {
			if i == 0 {
				ic.shell.Println()
			}
			ic.shell.Println(tip)
		}
	}

	return suggestions, length, len(prefix)
}

func (ic iCompleter) getWords(prefix string, w []string) (s []Suggestion) {
	ctx := &Context{}
	cmd, args := ic.cmd.FindCmd(w, ctx)
	if cmd == nil {
		cmd, _ = ic.cmd, w
	}
	//if cmd.CompleterWithPrefix != nil {
	//	return cmd.CompleterWithPrefix(prefix, args)
	//}
	//if cmd.Completer != nil {
	//	return cmd.Completer(args)
	//}

	for k, child := range cmd.staticChildren {
		if !strings.HasPrefix(k, prefix) {
			continue
		}

		s = append(s, Suggestion{
			Word:     k,
			Param:    false,
			Optional: false,
			Help:     child.helpText(),
		})

	}

	if cmd.paramChild != nil {
		s = append(s, Suggestion{
			Word:     cmd.paramChild.Name,
			Param:    true,
			Optional: false,
			Help:     cmd.paramChild.helpText(),
		})
	}

	defer func() {
		sort.Sort(suggestionSorter(s))
	}()

	argMap := make(map[string]struct{})
	for _, arg := range args {
		argMap[arg] = struct{}{}
	}

	if len(args) > 0 {
		last := args[len(args)-1]
		for _, arg := range cmd.Args {
			if arg.Name == last && arg.Pair {
				s = append(s, Suggestion{
					Word:     arg.Name,
					Param:    true,
					Optional: arg.Optional,
					Help:     arg.Help,
				})
				return
			}
		}
	}

	for _, arg := range cmd.Args {
		if _, ok := argMap[arg.Name]; ok {
			continue
		}
		s = append(s, Suggestion{
			Word:     arg.Name,
			Param:    false,
			Optional: arg.Optional,
			Help:     arg.Help,
		})
	}

	return
}
