<img src="img/logo.png" width="350">

[![GitHub release](https://img.shields.io/github/release/knqyf263/cob.svg)](https://github.com/knqyf263/cob/releases/latest)
![](https://github.com/knqyf263/cob/workflows/Go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/knqyf263/cob)](https://goreportcard.com/report/github.com/knqyf263/cob)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](https://github.com/knqyf263/cob/blob/master/LICENSE)



# Abstract
`cob` compares benchmarks between the latest commit (HEAD) and the previous commit (HEAD{@1}). The program will fail if the change in score is worse than the threshold. This tools is suitable for CI/CD to detect a regression of a performance automatically.

<img src="img/usage.png" width="700">

`cob` runs `go test -bench` before and after commit internally, so it depends on `go` command.


# Table of Contents
<!-- TOC -->
- [Abstract](#abstract)
- [Continuous Integration (CI)](#continuous-integration-ci)
  - [GitHub Actions](#github-actions)
  - [Travis CI](#travis-ci)
  - [CircleCI](#circleci)
- [Example](#example)
  - [Print memory allocation statistics for benchmarks](#print-memory-allocation-statistics-for-benchmarks)
  - [Run only those benchmarks matching a regular expression](#run-only-those-benchmarks-matching-a-regular-expression)
  - [Show only benchmarks with worse score](#show-only-benchmarks-with-worse-score)
  - [Specify a threshold](#specify-a-threshold)
- [Usage](#usage)
- [Q&A](#qa)
  - [A result of benchmarks is unstable](#a-result-of-benchmarks-is-unstable)

# Continuous Integration (CI)

See [cob-example](https://github.com/knqyf263/cob-example) for details.

## GitHub Actions

```
name: Bench
on: [push, pull_request]
jobs:
  test:
    name: Bench
    runs-on: ubuntu-latest
    steps:

    - name: Set up Go 1.13
      uses: actions/setup-go@v1
      with:
        go-version: 1.13
      id: go

    - name: Check out code into the Go module directory
      uses: actions/checkout@v1

    - name: Install GolangCI-Lint
      run: curl -sfL https://raw.githubusercontent.com/knqyf263/cob/master/install.sh | sudo sh -s -- -b /usr/local/bin

    - name: Run Benchmark
      run: cob -benchmem ./...
```

## Travis CI

```
dist: bionic
language: go
go:
  - 1.13.x

before_script:
  - curl -sfL https://raw.githubusercontent.com/knqyf263/cob/master/install.sh | sudo sh -s -- -b /usr/local/bin

script:
  - cob -benchmem ./...
```

## CircleCI

```
version: 2
jobs:
  bench:
    docker:
      - image: circleci/golang:1.13
    steps:
      - checkout
      - run:
          name: Install cob
          command: curl -sfL https://raw.githubusercontent.com/knqyf263/cob/master/install.sh | sudo sh -s -- -b /usr/local/bin
      - run:
          name: Run cob
          command: cob -benchmem ./...
workflows:
  version: 2
  build-workflow:
    jobs:
      - bench
```


# Example
## Print memory allocation statistics for benchmarks

```
$ cob -benchmem ./...
```

<details>
<summary>Result</summary>

```
2020/01/12 17:31:16 Run Benchmark: 4363944cbed3da7a8245cbcdc8d8240b8976eb24 HEAD{@1}
2020/01/12 17:31:19 Run Benchmark: 599a5523729d4d99a331b9d3f71dde9e1e6daef0 HEAD

Result
======

+-----------------------------+----------+---------------+-------------------+
|            Name             |  Commit  |    NsPerOp    | AllocedBytesPerOp |
+-----------------------------+----------+---------------+-------------------+
| BenchmarkAppend_Allocate-16 |   HEAD   |  175.00 ns/op |      111 B/op     |
+                             +----------+---------------+-------------------+
|                             | HEAD@{1} |  108.00 ns/op |      23 B/op      |
+-----------------------------+----------+---------------+-------------------+
|      BenchmarkCall-16       |   HEAD   |   0.27 ns/op  |       0 B/op      |
+                             +----------+---------------+                   +
|                             | HEAD@{1} |   0.29 ns/op  |                   |
+-----------------------------+----------+---------------+-------------------+

Comparison
==========

+-----------------------------+---------+-------------------+
|            Name             | NsPerOp | AllocedBytesPerOp |
+-----------------------------+---------+-------------------+
| BenchmarkAppend_Allocate-16 | 62.04%  |      382.61%      |
+-----------------------------+---------+-------------------+
|      BenchmarkCall-16       |  7.53%  |       0.00%       |
+-----------------------------+---------+-------------------+

2020/01/12 17:31:21 This commit makes benchmarks worse
```

</details>


## Run only those benchmarks matching a regular expression

<details>
<summary>Result</summary>

```
$ cob -bench Append ./...
2020/01/12 17:32:30 Run Benchmark: 4363944cbed3da7a8245cbcdc8d8240b8976eb24 HEAD{@1}
2020/01/12 17:32:32 Run Benchmark: 599a5523729d4d99a331b9d3f71dde9e1e6daef0 HEAD

Result
======

+-----------------------------+----------+---------------+-------------------+
|            Name             |  Commit  |    NsPerOp    | AllocedBytesPerOp |
+-----------------------------+----------+---------------+-------------------+
| BenchmarkAppend_Allocate-16 |   HEAD   |  179.00 ns/op |      117 B/op     |
+                             +----------+---------------+-------------------+
|                             | HEAD@{1} |  115.00 ns/op |      23 B/op      |
+-----------------------------+----------+---------------+-------------------+

Comparison
==========

+-----------------------------+---------+-------------------+
|            Name             | NsPerOp | AllocedBytesPerOp |
+-----------------------------+---------+-------------------+
| BenchmarkAppend_Allocate-16 | 55.65%  |      408.70%      |
+-----------------------------+---------+-------------------+
```

</details>

## Show only benchmarks with worse score

```
$ cob -benchmem -only-degression
```

<details>
<summary>Result</summary>

```
2020/01/12 17:48:35 Run Benchmark: 4363944cbed3da7a8245cbcdc8d8240b8976eb24 HEAD{@1}
2020/01/12 17:48:38 Run Benchmark: 599a5523729d4d99a331b9d3f71dde9e1e6daef0 HEAD

Comparison
==========

+-----------------------------+---------+-------------------+
|            Name             | NsPerOp | AllocedBytesPerOp |
+-----------------------------+---------+-------------------+
| BenchmarkAppend_Allocate-16 | 52.34%  |      347.83%      |
+-----------------------------+---------+-------------------+

2020/01/12 17:48:39 This commit makes benchmarks worse

```

</details>

## Specify a threshold

The following option means the program fails if a benchmark score gets worse than 50%.

```
$ cob -threshold 0.5 ./...
```

# Usage

```
$ cob -h
NAME:
   cob - Continuous Benchmark for Go project

USAGE:
   cob [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --only-degression  Show only benchmarks with worse score (default: false)
   --threshold value  The program fails if the benchmark gets worse than the threshold (default: 0.1)
   --bench value      Run only those benchmarks matching a regular expression. (default: ".")
   --benchmem         Print memory allocation statistics for benchmarks. (default: false)
   --benchtime value  Run enough iterations of each benchmark to take t, specified as a time.Duration (for example, -benchtime 1h30s). (default: "1s")
   --help, -h         show help (default: false)

```

# Q&A

## Benchmarks with the same name

Specify a package name.

```
$ cob -benchmem ./foo
$ cob -benchmem ./bar
```

## A result of benchmarks is unstable

You can specify `--benchtime`.

```
$ cob -benchtime 10s ./...
```

# License

This repository is available under the [MIT](https://github.com/knqyf263/cob/blob/master/LICENSE)

# Author

[Teppei Fukuda](https://github.com/knqyf263) (knqyf263)
