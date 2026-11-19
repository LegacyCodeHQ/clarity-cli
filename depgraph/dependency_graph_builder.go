package depgraph

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	graphlib "github.com/dominikbraun/graph"

	"github.com/LegacyCodeHQ/clarity/vcs"
)

// BuildDependencyGraph analyzes a list of files and builds a dependency graph
// containing only project imports (excluding package:/dart: imports for Dart,
// and standard library/external imports for Go).
// Only dependencies that are in the supplied file list are included in the graph.
// The contentReader function is used to read file contents (from filesystem, git commit, etc.)
func BuildDependencyGraph(filePaths []string, contentReader vcs.ContentReader) (DependencyGraph, error) {
	ctx, err := buildDependencyGraphContext(filePaths, contentReader)
	if err != nil {
		return nil, err
	}

	return BuildDependencyGraphWithResolver(filePaths, NewDefaultDependencyResolver(ctx, contentReader))
}

// BuildDependencyGraphInUniverse resolves the files in scope against a larger,
// authoritative file universe, then returns only the induced scope graph.
// Dependency meaning must not change merely because a renderer shows fewer
// files: in particular, Rust crate-root re-exports need the complete crate in
// the resolver context even when only the working set is visible.
func BuildDependencyGraphInUniverse(
	filePaths []string,
	universeFilePaths []string,
	contentReader vcs.ContentReader,
) (DependencyGraph, error) {
	ctx, err := buildDependencyGraphContext(universeFilePaths, contentReader)
	if err != nil {
		return nil, err
	}

	graph, err := BuildDependencyGraphWithResolver(
		filePaths,
		NewDefaultDependencyResolver(ctx, contentReader),
	)
	if err != nil {
		return nil, err
	}

	adjacency, err := AdjacencyList(graph)
	if err != nil {
		return nil, err
	}
	keep := make(map[string]bool, len(filePaths))
	for _, path := range filePaths {
		absolute, absErr := filepath.Abs(path)
		if absErr != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", path, absErr)
		}
		keep[absolute] = true
	}

	return extractSubgraph(adjacency, keep), nil
}

// BuildDependencyGraphWithResolver builds a graph using the provided DependencyResolver implementation.
func BuildDependencyGraphWithResolver(
	filePaths []string,
	dependencyResolver DependencyResolver,
) (DependencyGraph, error) {
	graph := NewDependencyGraph()

	if dependencyResolver == nil {
		return nil, fmt.Errorf("dependency resolver is required")
	}

	type resolveResult struct {
		absPath        string
		projectImports []string
		supported      bool
		err            error
	}

	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(filePaths) {
		workerCount = len(filePaths)
	}
	if workerCount < 1 {
		workerCount = 1
	}

	results := make([]resolveResult, len(filePaths))
	jobs := make(chan int)
	var wg sync.WaitGroup

	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				filePath := filePaths[idx]
				absPath, err := filepath.Abs(filePath)
				if err != nil {
					results[idx] = resolveResult{
						err: fmt.Errorf("failed to resolve path %s: %w", filePath, err),
					}
					continue
				}

				ext := filepath.Ext(absPath)
				if !dependencyResolver.SupportsFileExtension(ext) {
					results[idx] = resolveResult{
						absPath:   absPath,
						supported: false,
					}
					continue
				}

				projectImports, err := dependencyResolver.ResolveProjectImports(absPath, filePath, ext)
				if err != nil {
					results[idx] = resolveResult{err: err}
					continue
				}

				if len(projectImports) > 0 {
					projectImports = deduplicatePaths(projectImports)
				}
				results[idx] = resolveResult{
					absPath:        absPath,
					projectImports: projectImports,
					supported:      true,
				}
			}
		}()
	}

	for i := range filePaths {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	for _, result := range results {
		if result.err != nil {
			return nil, result.err
		}

		if err := graph.AddVertex(result.absPath); err != nil && !errors.Is(err, graphlib.ErrVertexAlreadyExists) {
			return nil, fmt.Errorf("failed to add graph vertex %s: %w", result.absPath, err)
		}

		if !result.supported {
			continue
		}

		for _, dep := range result.projectImports {
			if err := graph.AddVertex(dep); err != nil && !errors.Is(err, graphlib.ErrVertexAlreadyExists) {
				return nil, fmt.Errorf("failed to add graph dependency vertex %s: %w", dep, err)
			}
			if err := graph.AddEdge(result.absPath, dep); err != nil && !errors.Is(err, graphlib.ErrEdgeAlreadyExists) {
				return nil, fmt.Errorf("failed to add graph edge %s -> %s: %w", result.absPath, dep, err)
			}
		}
	}

	// Third pass: add intra-package dependencies for languages that need it.
	if err := dependencyResolver.FinalizeGraph(graph); err != nil {
		return graph, fmt.Errorf("failed to add intra-package dependencies: %w", err)
	}

	return graph, nil
}

// deduplicatePaths removes duplicate entries while preserving insertion order
func deduplicatePaths(paths []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}
