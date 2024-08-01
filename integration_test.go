package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
)

func TestCOBIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cob-integration-test")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Build cob in the temporary directory
	cmd := exec.Command("go", "build", "-o", tempDir, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build cob: %v\nOutput: %s", err, output)
	}

	// Initialize git repository
	err = initGitRepo(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize git repository: %v", err)
	}

	// Create two mock benchmark files, the second one slower than the first
	err = createMockBenchmarks(tempDir)
	if err != nil {
		t.Fatalf("failed to create mock benchmarks: %v", err)
	}

	// Run cob in the temporary directory
	cmd = exec.Command(filepath.Join(tempDir, "cob"))
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()

	// Expect an error because the second benchmark is slower than the first
	if err == nil {
		t.Fatalf("expected an error, but got none")
	}

	assert.Contains(t, string(output), "Comparison")
	assert.Contains(t, string(output), "BenchmarkFoo")
	assert.Contains(t, string(output), "This commit makes benchmarks worse")
}

func initGitRepo(dir string) error {
	r, err := git.PlainInit(dir, false)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0644)
	if err != nil {
		return fmt.Errorf("failed to create go.mod: %w", err)
	}

	_, err = w.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add go.mod: %w", err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

func createMockBenchmarks(dir string) error {
	r, err := git.PlainOpen(dir)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	tmpl := `
package benchmark

import "testing"

func BenchmarkFoo(b *testing.B) {
    for i := 0; i < b.N; i++ {
        foo({{.Iterations}})
    }
}

func foo(n int) int {
    sum := 0
    for i := 0; i < n; i++ {
        sum += i
    }
    return sum
}
`

	// Create initial (good) benchmark
	if err := createBenchmarkFile(dir, "bench_test.go", tmpl, 100); err != nil {
		return fmt.Errorf("failed to create good benchmark: %w", err)
	}

	_, err = w.Add("bench_test.go")
	if err != nil {
		return fmt.Errorf("failed to add bench_test.go: %w", err)
	}

	_, err = w.Commit("Add initial good benchmark", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit initial benchmark: %w", err)
	}

	// Create worse benchmark (replacing the good one)
	if err := createBenchmarkFile(dir, "bench_test.go", tmpl, 1000); err != nil {
		return fmt.Errorf("failed to create worse benchmark: %w", err)
	}

	_, err = w.Add("bench_test.go")
	if err != nil {
		return fmt.Errorf("failed to add updated bench_test.go: %w", err)
	}

	_, err = w.Commit("Update with worse benchmark", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit worse benchmark: %w", err)
	}

	return nil
}

func createBenchmarkFile(dir, filename, tmpl string, iterations int) error {
	t, err := template.New("bench").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	data := struct {
		Iterations int
	}{
		Iterations: iterations,
	}

	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
