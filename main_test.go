package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_showResult(t *testing.T) {
	type args struct {
		rows     [][]string
		benchmem bool
	}
	tests := []struct {
		name      string
		args      args
		wantTable string
	}{
		{
			name: "happy path",
			args: args{
				rows: [][]string{
					{"BenchmarkA", "HEAD", "123 ns/op"},
					{"BenchmarkA", "HEAD{@1}", "133 ns/op"},
					{"BenchmarkB", "HEAD", "5 ns/op"},
					{"BenchmarkB", "HEAD{@1}", "9 ns/op"},
				},
			},
			wantTable: `
Result
======

+------------+----------+-----------+
|    Name    |  Commit  |  NsPerOp  |
+------------+----------+-----------+
| BenchmarkA |   HEAD   | 123 ns/op |
+            +----------+-----------+
|            | HEAD{@1} | 133 ns/op |
+------------+----------+-----------+
| BenchmarkB |   HEAD   |  5 ns/op  |
+            +----------+-----------+
|            | HEAD{@1} |  9 ns/op  |
+------------+----------+-----------+
`,
		},
		{
			name: "with benchmem",
			args: args{
				rows: [][]string{
					{"BenchmarkA", "HEAD", "123 ns/op", "234 B/op"},
					{"BenchmarkA", "HEAD{@1}", "133 ns/op", "255 B/op"},
					{"BenchmarkB", "HEAD", "5 ns/op", "7 B/op"},
					{"BenchmarkB", "HEAD{@1}", "9 ns/op", "8 B/op"},
				},
				benchmem: true,
			},
			wantTable: `
Result
======

+------------+----------+-----------+-------------------+
|    Name    |  Commit  |  NsPerOp  | AllocedBytesPerOp |
+------------+----------+-----------+-------------------+
| BenchmarkA |   HEAD   | 123 ns/op |     234 B/op      |
+            +----------+-----------+-------------------+
|            | HEAD{@1} | 133 ns/op |     255 B/op      |
+------------+----------+-----------+-------------------+
| BenchmarkB |   HEAD   |  5 ns/op  |      7 B/op       |
+            +----------+-----------+-------------------+
|            | HEAD{@1} |  9 ns/op  |      8 B/op       |
+------------+----------+-----------+-------------------+
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			showResult(w, tt.args.rows, tt.args.benchmem)
			gotTable := w.String()
			assert.Equal(t, tt.wantTable, gotTable, tt.name)
		})
	}
}

func Test_showRatio(t *testing.T) {
	type args struct {
		results        []result
		benchmem       bool
		threshold      float64
		onlyDegression bool
	}
	tests := []struct {
		name      string
		args      args
		want      bool
		wantTable string
	}{
		{
			name: "happy path",
			args: args{
				results: []result{
					{
						Name:         "BenchmarkA",
						RatioNsPerOp: 0.01,
					},
				},
				benchmem:       false,
				threshold:      0.2,
				onlyDegression: false,
			},
			want: false,
			wantTable: fmt.Sprintf(`
Comparison
==========

+------------+---------+
|    Name    | NsPerOp |
+------------+---------+
| BenchmarkA |  %s  |
+------------+---------+

`, "\x1b[1;91m1.00%\x1b[0m"),
		},
		{
			name: "with benchmem",
			args: args{
				results: []result{
					{
						Name:                   "BenchmarkA",
						RatioNsPerOp:           0.01,
						RatioAllocedBytesPerOp: 0.5,
					},
				},
				benchmem:       true,
				threshold:      0.2,
				onlyDegression: false,
			},
			want: true,
			wantTable: fmt.Sprintf(`
Comparison
==========

+------------+---------+-------------------+
|    Name    | NsPerOp | AllocedBytesPerOp |
+------------+---------+-------------------+
| BenchmarkA |  %s  |      %s       |
+------------+---------+-------------------+

`, "\x1b[1;91m1.00%\x1b[0m", "\x1b[1;91m50.00%\x1b[0m"),
		},
		{
			name: "better and worse",
			args: args{
				results: []result{
					{
						Name:                   "BenchmarkA",
						RatioNsPerOp:           0.12345,
						RatioAllocedBytesPerOp: -0.9,
					},
					{
						Name:                   "BenchmarkB",
						RatioNsPerOp:           -0.03,
						RatioAllocedBytesPerOp: 0.5,
					},
				},
				benchmem:       true,
				threshold:      0.95,
				onlyDegression: false,
			},
			want: false,
			wantTable: fmt.Sprintf(`
Comparison
==========

+------------+---------+-------------------+
|    Name    | NsPerOp | AllocedBytesPerOp |
+------------+---------+-------------------+
| BenchmarkA | %s  |      %s       |
+------------+---------+-------------------+
| BenchmarkB |  %s  |      %s       |
+------------+---------+-------------------+

`, "\x1b[1;91m12.35%\x1b[0m", "\x1b[1;34m90.00%\x1b[0m", "\x1b[1;34m3.00%\x1b[0m", "\x1b[1;91m50.00%\x1b[0m"),
		},
		{
			name: "only degression",
			args: args{
				results: []result{
					{
						Name:                   "BenchmarkA",
						RatioNsPerOp:           0.12345,
						RatioAllocedBytesPerOp: -0.9,
					},
					{
						Name:                   "BenchmarkB",
						RatioNsPerOp:           -0.03,
						RatioAllocedBytesPerOp: 0.5,
					},
				},
				benchmem:       true,
				threshold:      0.4,
				onlyDegression: true,
			},
			want: true,
			wantTable: fmt.Sprintf(`
Comparison
==========

+------------+---------+-------------------+
|    Name    | NsPerOp | AllocedBytesPerOp |
+------------+---------+-------------------+
| BenchmarkB |  %s  |      %s       |
+------------+---------+-------------------+

`, "\x1b[1;34m3.00%\x1b[0m", "\x1b[1;91m50.00%\x1b[0m"),
		},
		{
			name: "no benchmark",
			args: args{
				results: []result{
					{
						Name:                   "BenchmarkA",
						RatioNsPerOp:           0.12345,
						RatioAllocedBytesPerOp: -0.9,
					},
					{
						Name:                   "BenchmarkB",
						RatioNsPerOp:           -0.03,
						RatioAllocedBytesPerOp: 0.5,
					},
				},
				benchmem:       false,
				threshold:      0.6,
				onlyDegression: true,
			},
			want:      false,
			wantTable: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			got := showRatio(w, tt.args.results, tt.args.benchmem, tt.args.threshold, tt.args.onlyDegression)
			gotTable := w.String()
			assert.Equal(t, tt.wantTable, gotTable, tt.name)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}

func Test_generateRatioItem(t *testing.T) {
	type args struct {
		ratio float64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "positive",
			args: args{
				ratio: 0.2,
			},
			want: "20.00%",
		},
		{
			name: "positive round",
			args: args{
				ratio: 0.234567,
			},
			want: "23.46%",
		},
		{
			name: "negative",
			args: args{
				ratio: -0.5,
			},
			want: "50.00%",
		},
		{
			name: "negative round",
			args: args{
				ratio: -0.56789,
			},
			want: "56.79%",
		},
		{
			name: "big number",
			args: args{
				ratio: 123.4567,
			},
			want: "12345.67%",
		},
		{
			name: "small positive number",
			args: args{
				ratio: 0.00001,
			},
			want: "0.00%",
		},
		{
			name: "small negative number",
			args: args{
				ratio: -0.00001,
			},
			want: "0.00%",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateRatioItem(tt.args.ratio)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
