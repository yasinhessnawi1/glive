package benchmark

import (
	"testing"

	"github.com/glive/domain/values"
)

func BenchmarkURLParsing(b *testing.B) {
	urls := []string{
		"https://github.com/user/repo",
		"https://github.com/organization/large-repo-name",
		"git@github.com:user/repo.git",
		"user/repo",
	}

	for _, url := range urls {
		b.Run(url, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = values.ParseRepoURL(url)
			}
		})
	}
}

func BenchmarkCommandValidation(b *testing.B) {
	validator := values.NewCommandValidator()
	commands := []string{
		"npm install",
		"pip install -r requirements.txt",
		"python -m venv venv",
		"go build ./...",
	}

	for _, cmd := range commands {
		b.Run(cmd, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = validator.Validate(cmd, "/workspace")
			}
		})
	}
}

func BenchmarkPathValidation(b *testing.B) {
	paths := []string{
		"/tmp/test",
		"/tmp/test/subdir/file.txt",
		"relative/path",
		"../../safe/path",
	}

	for _, path := range paths {
		b.Run(path, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = values.NewPath(path)
			}
		})
	}
}

func BenchmarkSafePathValidation(b *testing.B) {
	root := "/allowed/root"
	paths := []string{
		"/allowed/root/subdir/file.txt",
		"/allowed/root/nested/deep/path",
	}

	for _, path := range paths {
		b.Run(path, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = values.NewSafePath(path, root)
			}
		})
	}
}

func BenchmarkCommandParsing(b *testing.B) {
	commands := []string{
		"npm install",
		"npm install express lodash",
		"npm install \"express@latest\"",
		"python -m venv venv && source venv/bin/activate",
	}

	for _, cmd := range commands {
		b.Run(cmd, func(b *testing.B) {
			validator := values.NewCommandValidator()
			for i := 0; i < b.N; i++ {
				_, _ = validator.Validate(cmd, "/tmp")
			}
		})
	}
}


