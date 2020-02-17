package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/src-d/go-git.v4/plumbing"

	"golang.org/x/xerrors"

	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"golang.org/x/tools/benchmark/parse"
	"gopkg.in/src-d/go-git.v4"
)

type result struct {
	Name                   string
	RatioNsPerOp           float64
	RatioAllocedBytesPerOp float64
}

type comparedScore struct {
	nsPerOp           bool
	allocedBytesPerOp bool
}

func main() {
	app := &cli.App{
		Name:  "cob",
		Usage: "Continuous Benchmark for Go project",
		Action: func(c *cli.Context) error {
			return run(newConfig(c))
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "only-degression",
				Usage: "Show only benchmarks with worse score",
			},
			&cli.Float64Flag{
				Name:  "threshold",
				Usage: "The program fails if the benchmark gets worse than the threshold",
				Value: 0.2,
			},
			&cli.StringFlag{
				Name:  "base",
				Usage: "Specify a base commit compared with HEAD",
				Value: "HEAD~1",
			},
			&cli.StringFlag{
				Name:  "compare",
				Usage: "Which score to compare",
				Value: "ns/op,B/op",
			},
			&cli.StringFlag{
				Name:  "bench-cmd",
				Usage: "Specify a command to measure benchmarks",
				Value: "go",
			},
			&cli.StringFlag{
				Name:  "bench-args",
				Usage: "Specify arguments passed to -cmd",
				Value: "test -run '^$' -bench . -benchmem ./...",
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(c config) error {
	r, err := git.PlainOpen(".")
	if err != nil {
		return xerrors.Errorf("unable to open the git repository: %w", err)
	}

	head, err := r.Head()
	if err != nil {
		return xerrors.Errorf("unable to get the reference where HEAD is pointing to: %w", err)
	}

	prev, err := r.ResolveRevision(plumbing.Revision(c.base))
	if err != nil {
		return xerrors.Errorf("unable to resolves revision to corresponding hash: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return xerrors.Errorf("unable to get a worktree based on the given fs: %w", err)
	}

	s, err := w.Status()
	if err != nil {
		return xerrors.Errorf("unable to get the working tree status: %w", err)
	}

	if !s.IsClean() {
		return xerrors.New("the repository is dirty: commit all changes before running 'cob'")
	}

	err = w.Reset(&git.ResetOptions{Commit: *prev, Mode: git.HardReset})
	if err != nil {
		return xerrors.Errorf("failed to reset the worktree to a previous commit: %w", err)
	}

	defer func() {
		_ = w.Reset(&git.ResetOptions{Commit: head.Hash(), Mode: git.HardReset})
	}()

	log.Printf("Run Benchmark: %s %s", prev, c.base)
	prevSet, err := runPreviousBenchmark(c.benchCmd, c.benchArgs)
	if err != nil {
		return xerrors.Errorf("failed to run a benchmark: %w", err)
	}

	err = w.Reset(&git.ResetOptions{Commit: head.Hash(), Mode: git.HardReset})
	if err != nil {
		return xerrors.Errorf("failed to reset the worktree to HEAD: %w", err)
	}

	log.Printf("Run Benchmark: %s %s", head.Hash(), "HEAD")
	headSet, err := runBenchmark(c.benchCmd, c.benchArgs)
	if err != nil {
		return xerrors.Errorf("failed to run a benchmark: %w", err)
	}

	var ratios []result
	var rows [][]string
	for benchName, headBenchmarks := range headSet {
		var prevBench, headBench *parse.Benchmark

		if len(headBenchmarks) > 0 {
			headBench = headBenchmarks[0]
		}
		rows = append(rows, generateRow("HEAD", headBench))

		prevBenchmarks, ok := prevSet[benchName]
		if !ok {
			rows = append(rows, []string{benchName, c.base, "-", "-"})
			continue
		}

		if len(prevBenchmarks) > 0 {
			prevBench = prevBenchmarks[0]
		}
		rows = append(rows, generateRow(c.base, prevBench))

		var ratioNsPerOp float64
		if prevBench.NsPerOp != 0 {
			ratioNsPerOp = (headBench.NsPerOp - prevBench.NsPerOp) / prevBench.NsPerOp
		}

		var ratioAllocedBytesPerOp float64
		if prevBench.AllocedBytesPerOp != 0 {
			ratioAllocedBytesPerOp = (float64(headBench.AllocedBytesPerOp) - float64(prevBench.AllocedBytesPerOp)) / float64(prevBench.AllocedBytesPerOp)
		}

		ratios = append(ratios, result{
			Name:                   benchName,
			RatioNsPerOp:           ratioNsPerOp,
			RatioAllocedBytesPerOp: ratioAllocedBytesPerOp,
		})
	}

	if !c.onlyDegression {
		showResult(os.Stdout, rows)
	}

	degression := showRatio(os.Stdout, ratios, c.threshold, whichScoreToCompare(c.compare), c.onlyDegression)
	if degression {
		return xerrors.New("This commit makes benchmarks worse")
	}

	return nil
}

func runBenchmark(cmdStr string, args []string) (parse.Set, error) {
	var stderr bytes.Buffer
	cmd := exec.Command(cmdStr, args...)
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if strings.HasSuffix(strings.TrimSpace(stderr.String()), "no packages to test") {
			return parse.Set{}, nil
		}
		log.Println(string(out))
		log.Println(stderr.String())
		return nil, xerrors.Errorf("failed to run '%s' command: %w", cmd, err)
	}

	s, err := parseBenchmark(out)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse a result of benchmarks: %w", err)
	}
	return s, nil
}

func runPreviousBenchmark(cmdStr string, args []string) (parse.Set, error) {
	prevSet, err := runBenchmark(cmdStr, args)
	if err != nil && strings.Contains(err.Error(), "exit status ") {
		log.Printf("previous benchmark failed: %s\n", err.Error())
		return parse.Set{}, nil
	}
	return prevSet, err
}

func parseBenchmark(data []byte) (parse.Set, error) {
	r := bytes.NewReader(data)
	scan := bufio.NewScanner(r)
	ord := 0

	benchmarkName := ""
	bb := make(parse.Set)
	for scan.Scan() {
		t := scan.Text()

		if ok := parseLine(bb, t, ord); ok {
			ord++
			continue
		}

		containBenchmark := strings.HasPrefix(t, "Benchmark")
		if containBenchmark {
			benchmarkName = strings.Split(t, " ")[0]
			continue
		}

		if ok := parseLine(bb, fmt.Sprintf("%s %s", benchmarkName, t), ord); ok {
			ord++
		}
	}

	if err := scan.Err(); err != nil {
		return nil, err
	}

	return bb, nil
}

func parseLine(bb parse.Set, line string, ord int) bool {
	if b, err := parse.ParseLine(line); err == nil {
		b.Ord = ord
		bb[b.Name] = append(bb[b.Name], b)
		return true
	}
	return false
}

func generateRow(ref string, b *parse.Benchmark) []string {
	return []string{b.Name, ref, fmt.Sprintf(" %.2f ns/op", b.NsPerOp),
		fmt.Sprintf(" %d B/op", b.AllocedBytesPerOp)}
}

func showResult(w io.Writer, rows [][]string) {
	fmt.Fprintln(w, "\nResult")
	fmt.Fprintf(w, "%s\n\n", strings.Repeat("=", 6))

	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	headers := []string{"Name", "Commit", "NsPerOp", "AllocedBytesPerOp"}
	table.SetHeader(headers)
	table.SetAutoMergeCells(true)
	table.SetRowLine(true)
	table.AppendBulk(rows)
	table.Render()
}

func showRatio(w io.Writer, results []result, threshold float64, comparedScore comparedScore, onlyDegression bool) bool {
	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetRowLine(true)
	headers := []string{"Name", "NsPerOp", "AllocedBytesPerOp"}
	table.SetHeader(headers)

	var degression bool
	for _, result := range results {
		if comparedScore.nsPerOp && threshold < result.RatioNsPerOp {
			degression = true
		} else if comparedScore.allocedBytesPerOp && threshold < result.RatioAllocedBytesPerOp {
			degression = true
		} else {
			if onlyDegression {
				continue
			}
		}
		row := []string{result.Name, generateRatioItem(result.RatioNsPerOp), generateRatioItem(result.RatioAllocedBytesPerOp)}
		colors := []tablewriter.Colors{{}, generateColor(result.RatioNsPerOp), generateColor(result.RatioAllocedBytesPerOp)}
		if !comparedScore.nsPerOp {
			row[1] = "-"
			colors[1] = tablewriter.Colors{}
		}
		if !comparedScore.allocedBytesPerOp {
			row[2] = "-"
			colors[2] = tablewriter.Colors{}
		}
		table.Rich(row, colors)
	}
	if table.NumLines() > 0 {
		fmt.Fprintln(w, "\nComparison")
		fmt.Fprintf(w, "%s\n\n", strings.Repeat("=", 10))

		table.Render()
		fmt.Fprintln(w)
	}
	return degression
}

func generateRatioItem(ratio float64) string {
	if -0.0001 < ratio && ratio < 0.0001 {
		ratio = 0
	}
	if 0 <= ratio {
		return fmt.Sprintf("%.2f%%", 100*ratio)
	}
	return fmt.Sprintf("%.2f%%", -100*ratio)
}

func generateColor(ratio float64) tablewriter.Colors {
	if ratio > 0 {
		return tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor}
	}
	return tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlueColor}
}

func whichScoreToCompare(c []string) comparedScore {
	var comparedScore comparedScore
	for _, cc := range c {
		switch cc {
		case "ns/op":
			comparedScore.nsPerOp = true
		case "B/op":
			comparedScore.allocedBytesPerOp = true
		}
	}
	return comparedScore
}
