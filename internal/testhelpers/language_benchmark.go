package testhelpers

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/LegacyCodeHQ/clarity/depgraph/moduleapi"
)

// BenchmarkLanguageCorpus profiles a complete provider pass over a local corpus.
// The environment variable names the source directory; ordinary tests need no
// external repositories. Each iteration creates a fresh resolver and uses the
// filesystem reader, including source reads and configuration discovery.
func BenchmarkLanguageCorpus(b *testing.B, provider moduleapi.Provider, environment string) {
	b.Helper()
	root := os.Getenv(environment)
	if root == "" {
		b.Skip("set " + environment + " to a source checkout")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		b.Fatal(err)
	}
	extensions := make(map[string]bool)
	for _, extension := range provider.Extensions() {
		extensions[extension] = true
	}
	var files []string
	var size int64
	ctx := &moduleapi.Context{
		SuppliedFiles:    make(map[string]bool),
		DirToFiles:       make(map[string][]string),
		FilesByExtension: make(map[string][]string),
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", ".lake":
				return filepath.SkipDir
			}
			return nil
		}
		extension := filepath.Ext(path)
		if !extensions[extension] {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		files = append(files, path)
		ctx.SuppliedFiles[path] = true
		ctx.DirToFiles[filepath.Dir(path)] = append(ctx.DirToFiles[filepath.Dir(path)], path)
		ctx.FilesByExtension[extension] = append(ctx.FilesByExtension[extension], path)
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	if len(files) == 0 {
		b.Fatal("no supported source files in " + root)
	}
	var reads, edges, tests int64
	reader := func(path string) ([]byte, error) {
		reads++
		return os.ReadFile(path)
	}
	b.ReportAllocs()
	b.SetBytes(size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver := provider.NewResolver(ctx, reader)
		for _, file := range files {
			dependencies, err := resolver.ResolveProjectImports(file, file, filepath.Ext(file))
			if err != nil {
				b.Fatal(err)
			}
			edges += int64(len(dependencies))
			if provider.IsTestFile(file, reader) {
				tests++
			}
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(len(files)), "files/op")
	b.ReportMetric(float64(reads)/float64(b.N), "reads/op")
	b.ReportMetric(float64(edges)/float64(b.N), "edges/op")
	b.ReportMetric(float64(tests)/float64(b.N), "tests/op")
}
