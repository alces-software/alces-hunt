// Copyright (C) 2026 Alces Software Ltd.
//
// This program and the accompanying materials are made available under
// the terms of the Eclipse Public License 2.0 which is available at
// https://www.eclipse.org/legal/epl-2.0
//
// SPDX-License-Identifier: EPL-2.0

package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sierra-tango-echo/alces-hunt/internal/commands"
	"github.com/sierra-tango-echo/alces-hunt/internal/config"
	"github.com/sierra-tango-echo/alces-hunt/internal/hunt"
	"github.com/sierra-tango-echo/alces-hunt/internal/parse"
	"github.com/sierra-tango-echo/alces-hunt/internal/send"
	"github.com/sierra-tango-echo/alces-hunt/internal/version"
)

// ProgramName is the CLI binary name.
func ProgramName() string {
	if v := os.Getenv("ALCES_HUNT_PROGRAM_NAME"); v != "" {
		return v
	}
	return "alces-hunt"
}

// Run dispatches a subcommand.
func Run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "-h", "--help", "help":
		if len(rest) > 0 {
			return helpCommand(rest[0])
		}
		printUsage(os.Stdout)
		return nil
	case "-v", "--version", "version":
		fmt.Printf("v%s\n", version.Version)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch cmd {
	case "hunt":
		return runHunt(cfg, rest)
	case "send":
		return runSend(cfg, rest)
	case "autorun":
		return runAutorun(cfg, rest)
	case "list":
		return runList(cfg, rest)
	case "show":
		return runShow(cfg, rest)
	case "remove-node":
		return runRemove(cfg, rest)
	case "modify-groups":
		return runModifyGroups(cfg, rest)
	case "modify-label":
		return runModifyLabel(cfg, rest)
	case "rename-group":
		return runRenameGroup(cfg, rest)
	case "parse":
		return runParse(cfg, rest)
	case "dump-buffer":
		return runDump(cfg, rest)
	default:
		return fmt.Errorf("unknown command %q\n\nRun '%s --help' for usage", cmd, ProgramName())
	}
}

func runHunt(cfg *config.Config, args []string) error {
	fs := newFlagSet("hunt")
	var opt hunt.Options
	fs.StringVar(&opt.Port, "port", "", "Override port")
	fs.BoolVar(&opt.AllowExisting, "allow-existing", false, "Allow replacement of existing entries")
	fs.BoolVar(&opt.IncludeSelf, "include-self", false, "Immediately try to send payload to self")
	fs.StringVar(&opt.Auth, "auth", "", "Override default authentication key")
	fs.StringVar(&opt.AutoParse, "auto-parse", "", "Automatically parse nodes matching this regex")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	return hunt.Run(cfg, opt)
}

func runSend(cfg *config.Config, args []string) error {
	fs := newFlagSet("send")
	var opt send.Options
	var groups string
	fs.StringVar(&opt.Command, "c", "", "Command to use to generate sent content")
	fs.StringVar(&opt.Command, "command", "", "Command to use to generate sent content")
	fs.StringVar(&opt.Port, "p", "", "Override server port")
	fs.StringVar(&opt.Port, "port", "", "Override server port")
	fs.StringVar(&opt.Server, "s", "", "Override server hostname")
	fs.StringVar(&opt.Server, "server", "", "Override server hostname")
	fs.StringVar(&opt.Auth, "auth", "", "Override default authentication key")
	fs.BoolVar(&opt.Broadcast, "broadcast", false, "Send identity to all nodes on a given subnet")
	fs.StringVar(&opt.BroadcastAddress, "broadcast-address", "", "Specify a broadcast address to use if broadcasting")
	fs.StringVar(&groups, "groups", "", "Comma-separated list of groups for this node")
	fs.StringVar(&opt.Label, "label", "", "Specify a label to use for this node")
	fs.StringVar(&opt.Prefix, "prefix", "", "Specify a prefix to use for this node")
	fs.StringVar(&opt.RetryInterval, "retry-interval", "", "Retry send every N seconds until successful")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	opt.LabelSet = flagWasSet(fs, "label")
	if groups != "" {
		for _, g := range strings.Split(groups, ",") {
			g = strings.TrimSpace(g)
			if g != "" {
				opt.Groups = append(opt.Groups, g)
			}
		}
	}
	return send.Run(cfg, opt)
}

func runAutorun(cfg *config.Config, args []string) error {
	switch cfg.AutorunMode {
	case "hunt":
		return runHunt(cfg, args)
	case "send":
		return runSend(cfg, args)
	default:
		return fmt.Errorf("Autorun mode provided is invalid.")
	}
}

func runList(cfg *config.Config, args []string) error {
	fs := newFlagSet("list")
	var opt commands.ListOptions
	fs.BoolVar(&opt.Plain, "plain", false, "Print in machine-readable manner")
	fs.BoolVar(&opt.ByGroup, "by-group", false, "Group nodes by group")
	fs.BoolVar(&opt.Buffer, "buffer", false, "Use node buffer list instead of parsed")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	return commands.List(cfg, opt)
}

func runShow(cfg *config.Config, args []string) error {
	fs := newFlagSet("show")
	var opt commands.ShowOptions
	fs.BoolVar(&opt.Buffer, "buffer", false, "Use node buffer list instead of parsed (use ID instead of label here)")
	fs.BoolVar(&opt.Plain, "plain", false, "Print in machine-readable format")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	arg := ""
	if len(pos) > 0 {
		arg = pos[0]
	}
	return commands.Show(cfg, arg, opt)
}

func runRemove(cfg *config.Config, args []string) error {
	fs := newFlagSet("remove-node")
	var opt commands.SelectOptions
	fs.BoolVar(&opt.Buffer, "buffer", false, "Use node buffer list instead of parsed (use ID instead of label here)")
	fs.BoolVar(&opt.Regex, "regex", false, "Match all nodes with regex NODE")
	fs.BoolVar(&opt.MatchHostname, "match-hostname", false, "Match against hostname instead of label (or ID)")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	arg := ""
	if len(pos) > 0 {
		arg = pos[0]
	}
	return commands.RemoveNode(cfg, arg, opt)
}

func runModifyGroups(cfg *config.Config, args []string) error {
	fs := newFlagSet("modify-groups")
	var opt commands.ModifyGroupsOptions
	fs.StringVar(&opt.Add, "add", "", "Comma separated list of groups to add")
	fs.StringVar(&opt.Remove, "remove", "", "Comma separated list of groups to remove")
	fs.BoolVar(&opt.Buffer, "buffer", false, "Use node buffer list instead of parsed (use ID instead of label here)")
	fs.BoolVar(&opt.Regex, "regex", false, "Match all hostnames with regex NODE")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	arg := ""
	if len(pos) > 0 {
		arg = pos[0]
	}
	return commands.ModifyGroups(cfg, arg, opt)
}

func runModifyLabel(cfg *config.Config, args []string) error {
	fs := newFlagSet("modify-label")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	oldL, newL := "", ""
	if len(pos) > 0 {
		oldL = pos[0]
	}
	if len(pos) > 1 {
		newL = pos[1]
	}
	return commands.ModifyLabel(cfg, oldL, newL)
}

func runRenameGroup(cfg *config.Config, args []string) error {
	fs := newFlagSet("rename-group")
	var buffer bool
	fs.BoolVar(&buffer, "buffer", false, "Use node buffer list instead of parsed")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	oldG, newG := "", ""
	if len(pos) > 0 {
		oldG = pos[0]
	}
	if len(pos) > 1 {
		newG = pos[1]
	}
	return commands.RenameGroup(cfg, oldG, newG, buffer)
}

func runParse(cfg *config.Config, args []string) error {
	args = normalizeBareBool(args, "skip-used-index")
	fs := newFlagSet("parse")
	var opt parse.Options
	var skip string
	fs.StringVar(&opt.Prefix, "prefix", "", "Prefix for the generated labels")
	fs.StringVar(&opt.Start, "start", "", "Start value for the numeric portion of the labels")
	fs.BoolVar(&opt.Auto, "auto", false, "Automatically process everything in buffer list")
	fs.BoolVar(&opt.AllowExisting, "allow-existing", false, "Allow replacement of existing entries")
	fs.StringVar(&skip, "skip-used-index", "", "Ignore errors if a label index is already in use")
	fs.BoolVar(&opt.DryRun, "dry-run", false, "Print generated node labels without parsing nodes")
	fs.StringVar(&opt.DefaultLabel, "default-label", "", "Set the way that hostnames are processed in node labels. Must be 'short', 'long' or 'blank'")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	if flagWasSet(fs, "skip-used-index") {
		opt.SkipSet = true
		opt.SkipUsedIndex = strings.EqualFold(skip, "true") || skip == "" || skip == "1"
	}
	return parse.Run(cfg, opt)
}

func runDump(cfg *config.Config, args []string) error {
	fs := newFlagSet("dump-buffer")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	return commands.DumpBuffer(cfg)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s %s [options]\n", ProgramName(), name)
		fs.PrintDefaults()
	}
	return fs
}

// parseArgs accepts flags before or after positional arguments.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	needsValue := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) {
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			needsValue["-"+f.Name] = false
			needsValue["--"+f.Name] = false
			return
		}
		needsValue["-"+f.Name] = true
		needsValue["--"+f.Name] = true
	})
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		name := a
		if eq := strings.IndexByte(a, '='); eq >= 0 {
			name = a[:eq]
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && needsValue[name] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	if err := fs.Parse(flags); err != nil {
		return nil, err
	}
	return pos, nil
}

func normalizeBareBool(args []string, name string) []string {
	long := "--" + name
	var out []string
	for i := 0; i < len(args); i++ {
		if args[i] == long {
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") &&
				(strings.EqualFold(args[i+1], "true") || strings.EqualFold(args[i+1], "false")) {
				out = append(out, args[i], args[i+1])
				i++
				continue
			}
			out = append(out, long+"=true")
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

func helpCommand(name string) error {
	// Re-parse with -h by constructing a dummy flag set via each runner
	// is heavy; print the same usage the flag set would.
	args := []string{"-h"}
	cfg := &config.Config{}
	switch name {
	case "hunt":
		return runHunt(cfg, args)
	case "send":
		return runSend(cfg, args)
	case "list":
		return runList(cfg, args)
	case "show":
		return runShow(cfg, args)
	case "remove-node":
		return runRemove(cfg, args)
	case "modify-groups":
		return runModifyGroups(cfg, args)
	case "modify-label":
		return runModifyLabel(cfg, args)
	case "rename-group":
		return runRenameGroup(cfg, args)
	case "parse":
		return runParse(cfg, args)
	case "dump-buffer":
		return runDump(cfg, args)
	case "autorun":
		fmt.Printf("Usage: %s autorun\nInterpret running mode from config or environment (hunt or send).\n", ProgramName())
		return nil
	default:
		return fmt.Errorf("unknown command %q", name)
	}
}

func printUsage(w io.Writer) {
	p := ProgramName()
	fmt.Fprintf(w, `%s v%s — node discovery and inventory for bare-metal clusters

Usage:
  %s <command> [options]

Commands:
  hunt            Listen for broadcasting clients (TCP + UDP)
  send            Push this node's identity to a hunt server
  autorun         Dispatch to hunt or send from autorun_mode
  list            Show nodes in the parsed (or buffer) list
  show            Show details of one node
  remove-node     Remove nodes by label (or ID with --buffer)
  modify-groups   Add or remove groups on nodes
  modify-label    Change a parsed node's label
  rename-group    Rename a group across a list
  parse           Move nodes from buffer to parsed list
  dump-buffer     Drop all nodes in the buffer list
  help            Show this help
  version         Show version

Configuration is loaded from $ALCES_HUNT_ROOT/etc/config.yml and
$XDG_CONFIG_HOME/alces-hunt/config.yml. Every scalar key may be
overridden by ALCES_HUNT_<key>.

Install as a service with the bundled install.sh (server or send mode).
`, p, version.Version, p)
}
