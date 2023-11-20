package ishell

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"
)

// Cmd is a shell command handler.
type (
	Cmd struct {
		// Command name.
		Name string
		// Command name aliases.
		Aliases []string
		// Function to execute for the command.
		Func func(c *Context)
		// One liner help message for the command.
		Help string
		// More descriptive help message for the command.
		LongHelp string

		Args []Arg

		// Completer is custom autocomplete for command.
		// It takes in command arguments and returns
		// autocomplete options.
		// By default all commands get autocomplete of
		// subcommands.
		// A non-nil Completer overrides the default behaviour.
		Completer func(args []string) []string

		// CompleterWithPrefix is custom autocomplete like
		// for Completer, but also provides the prefix
		// already so far to the completion function
		// If both Completer and CompleterWithPrefix are given,
		// CompleterWithPrefix takes precedence
		CompleterWithPrefix func(prefix string, args []string) []string

		// subcommands.
		//children map[string]*Cmd
		parent         *Cmd
		staticChildren map[string]*Cmd
		paramChild     *Cmd
		kind           kind
	}

	Arg struct {
		Name string

		Pair     bool
		Optional bool
		Help     string
	}

	kind uint8
)

const (
	StaticKind kind = iota
	ParamKind

	paramLabel = byte(':')
	spliter    = "/"
)

func addCmd(parent, child *Cmd) *Cmd {
	name := child.Name
	if name[0] == paramLabel {
		_cmd := parent.paramChild
		if _cmd == nil {
			child.Name = name[1:]
			child.kind = ParamKind

			_cmd = child
			child.parent = parent
			parent.paramChild = _cmd
		}

		return parent.paramChild
	}

	if parent.staticChildren == nil {
		parent.staticChildren = make(map[string]*Cmd)
	}

	if _, ok := parent.staticChildren[name]; !ok {
		child.kind = StaticKind
		child.parent = parent
		parent.staticChildren[name] = child
	}
	return parent.staticChildren[name]
}

// AddCmd adds cmd as a subcommand.
func (c *Cmd) AddCmd(cmd *Cmd) {
	if cmd.Name == "" {
		panic("cmd name should not be empty")
	}

	cmd.Name = strings.TrimSuffix(strings.TrimPrefix(cmd.Name, spliter), spliter)
	names := strings.Split(cmd.Name, spliter)

	last := c

	for _, name := range names[:len(names)-1] {
		if name[0] == paramLabel && len(name) < 2 {
			panic("wildcards must be named with a non-empty name '" + cmd.Name + "'")
		}

		last = addCmd(last, &Cmd{Name: name})
	}

	name := names[len(names)-1]
	cmd.Name = name
	addCmd(last, cmd)
}

// DeleteCmd deletes cmd from subcommands.
func (c *Cmd) DeleteCmd(name string) {
	if name[0] == paramLabel {
		if c.paramChild != nil && c.paramChild.Name == name[1:] {
			c.paramChild = nil
		}
		return
	}

	delete(c.staticChildren, name)
}

// Children returns the subcommands of c.
func (c *Cmd) Children() []*Cmd {
	var cmds []*Cmd
	for _, cmd := range c.staticChildren {
		cmds = append(cmds, cmd)
	}

	if c.paramChild != nil {
		cmds = append(cmds, c.paramChild)
	}

	sort.Sort(cmdSorter(cmds))
	return cmds
}

func (c *Cmd) hasSubcommand() bool {
	if len(c.staticChildren) > 1 || c.paramChild != nil {
		return true
	}
	if _, ok := c.staticChildren["help"]; !ok {
		return len(c.staticChildren) > 0
	}
	return false
}

// HelpText returns the computed help of the command and its subcommands.
func (c *Cmd) HelpText() string {
	var b bytes.Buffer
	p := func(s ...interface{}) {
		fmt.Fprintln(&b)
		if len(s) > 0 {
			fmt.Fprintln(&b, s...)
		}
	}
	if c.LongHelp != "" {
		p(c.LongHelp)
	} else if c.Help != "" {
		p(c.Help)
	} else if c.Name != "" {
		p(c.Name, "has no help")
	}
	if c.hasSubcommand() {
		p("Commands:")
		w := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
		for _, child := range c.Children() {
			fmt.Fprintf(w, "\t%s\t\t\t%s\n", child.Name, child.Help)
		}
		w.Flush()
		p()
	}
	return b.String()
}

// helpText returns the help of the command.
func (c *Cmd) helpText() string {
	if c.LongHelp != "" {
		return c.LongHelp
	}
	return c.Help
}

// findChildCmd returns the subcommand with matching name or alias.
func findChildCmd(c *Cmd, name string) *Cmd {
	// find perfect matches first
	if cmd, ok := c.staticChildren[name]; ok {
		return cmd
	}

	// find alias matching the name
	for _, cmd := range c.staticChildren {
		for _, alias := range cmd.Aliases {
			if alias == name {
				return cmd
			}
		}
	}

	// find param child
	return c.paramChild
}

// FindCmd finds the matching Cmd for args.
// It returns the Cmd and the remaining args.
func (c *Cmd) FindCmd(args []string, ctx *Context) (*Cmd, []string) {
	var cmd *Cmd
	_c := c

	if ctx == nil {
		ctx = &Context{}
	}

	for i, arg := range args {
		if cmd1 := findChildCmd(_c, arg); cmd1 != nil {
			cmd = cmd1
			_c = cmd

			if cmd1.kind == ParamKind {
				ctx.Params = append(ctx.Params, Param{
					Key:   cmd1.Name,
					Value: arg,
				})
			}

			continue
		}
		return cmd, args[i:]
	}
	return cmd, nil
}

type cmdSorter []*Cmd

func (c cmdSorter) Len() int           { return len(c) }
func (c cmdSorter) Less(i, j int) bool { return c[i].Name < c[j].Name }
func (c cmdSorter) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
