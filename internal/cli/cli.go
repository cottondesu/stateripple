package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cottondesu/stateripple/internal/adapters"
	"github.com/cottondesu/stateripple/internal/fileio"
	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/rules/risk"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

const help = "StateRipple v0.1.0 - deterministic state propagation over any graph\nUsage:\n  stateripple simulate --input snapshot.json [--format json|text] [--out path]\n  stateripple analyze [--base revision] [--format json|text] [--out path]\n  stateripple version\nSimulate propagates seeded risk over the canonical snapshot graph. It needs no Git, Go, repository, or network.\nAnalyze is an optional Git+Go adapter: it compares base (default HEAD~1) to HEAD in a clean single-module Go repository and requires Git 2.45.0 or later.\nExit codes: 0 complete, 1 input/execution error, 2 incomplete result (generation limit or incomplete adapter evidence).\n"

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, help)
		return 1
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return writeHelp(stdout, stderr, help)
	}
	if args[0] == "version" {
		if len(args) != 1 {
			diagnose(stderr, "version accepts no arguments")
			return 1
		}
		return writeHelp(stdout, stderr, "stateripple v0.1.0\n")
	}
	if args[0] != "simulate" && args[0] != "analyze" {
		diagnose(stderr, "unknown command %q", args[0])
		fmt.Fprint(stderr, help)
		return 1
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	// The flag package prints a parse error verbatim, including the flag name
	// the caller chose, and ignores usage write errors. Run writes both itself.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	format := fs.String("format", "text", "output format: json or text")
	out := fs.String("out", "", "write report to this file instead of stdout")
	var input, base string
	if args[0] == "simulate" {
		fs.StringVar(&input, "input", "", "snapshot JSON file (required)")
	} else {
		fs.StringVar(&base, "base", "HEAD~1", "Git base revision")
	}
	if err := fs.Parse(args[1:]); err != nil {
		usage := help + flagDefaults(fs)
		if errors.Is(err, flag.ErrHelp) {
			if len(args) == 2 && (args[1] == "--help" || args[1] == "-h") {
				return writeHelp(stdout, stderr, usage)
			}
			return writeHelp(stderr, stderr, usage)
		}
		fmt.Fprintf(stderr, "%s\n%s", escapeDiagnostic(err.Error()), usage)
		return 1
	}
	if fs.NArg() != 0 {
		diagnose(stderr, "unexpected positional arguments")
		return 1
	}
	if *format != "json" && *format != "text" {
		diagnose(stderr, "--format must be json or text")
		return 1
	}
	var s snapshot.Snapshot
	var meta adapters.Metadata
	var inputRef *inputFileReference
	var err error
	if args[0] == "simulate" {
		if input == "" {
			diagnose(stderr, "--input is required")
			return 1
		}
		s, inputRef, err = readSnapshot(input)
	} else {
		var root string
		root, err = os.Getwd()
		if err == nil {
			s, meta, err = adapters.Analyze(ctx, root, base)
		}
	}
	if err != nil {
		diagnose(stderr, "%v", err)
		return 1
	}
	r, err := risk.Run(s)
	if err != nil {
		diagnose(stderr, "simulate: %v", err)
		return 1
	}
	r.Changes, r.UnknownChanges, r.BuildContext = meta.Changes, meta.UnknownChanges, meta.BuildContext
	r.Warnings = append(r.Warnings, meta.Warnings...)
	if meta.Incomplete {
		r.Status = "incomplete"
	}
	var data []byte
	if *format == "json" {
		data, err = report.JSON(r)
	} else {
		data = []byte(report.Text(r))
	}
	if err == nil {
		if *out != "" {
			err = writeReportFile(*out, data, inputRef)
		} else {
			var n int
			n, err = stdout.Write(data)
			if err == nil && n != len(data) {
				err = io.ErrShortWrite
			}
		}
	}
	if err != nil {
		diagnose(stderr, "write report: %v", err)
		return 1
	}
	if r.Status == "incomplete" {
		return 2
	}
	return 0
}

func readSnapshot(path string) (snapshot.Snapshot, *inputFileReference, error) {
	f, info, err := fileio.OpenRegular(path)
	if err != nil {
		return snapshot.Snapshot{}, nil, fmt.Errorf("open snapshot: %w", err)
	}
	s, readErr := snapshot.Read(f)
	closeErr := f.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return snapshot.Snapshot{}, nil, fmt.Errorf("read snapshot: %w", err)
	}
	return s, &inputFileReference{path: path, info: info}, nil
}
