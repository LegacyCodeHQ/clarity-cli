# Language Parsing

The project uses [go-tree-sitter](https://github.com/smacker/go-tree-sitter) to integrate [tree-sitter](https://tree-sitter.github.io/tree-sitter/) for parsing and extracting dependency information from source files.

## Available Languages

**go-tree-sitter** provides built-in support for parsing various programming languages, with each language binding in its own directory within the [project's source](https://github.com/smacker/go-tree-sitter).

## Unsupported Languages

### Dart

Dart support is not available natively in go-tree-sitter. To add support for Dart:

1. The [tree-sitter-dart](https://github.com/UserNobody14/tree-sitter-dart) repository was cloned and the parser was built using the [tree-sitter-cli](https://github.com/tree-sitter/tree-sitter/blob/master/crates/cli/README.md) tool.
2. The parser files (`parser.c`, `parser.h`, `scanner.c`) were copied into the `internal/tree_sitter/dart/` directory.
3. Go bindings were created inside the same directory to interface with the C-based tree-sitter parser using CGo.

### Zig

Zig support uses generated parser files from [tree-sitter-grammars/tree-sitter-zig](https://github.com/tree-sitter-grammars/tree-sitter-zig), stored under `internal/tree_sitter/zig/`.

The copied source is pinned to commit `976140ed1fc828c8fb2ff2bcfbe6853f1ae9f183`. See `internal/tree_sitter/zig/README.md` for the copied files and license details.

### Lean 4

Lean 4 (`.lean`) support uses the MIT-licensed experimental
[Julian/tree-sitter-lean](https://github.com/Julian/tree-sitter-lean) grammar.
Sources are pinned and vendored under `internal/tree_sitter/lean/`, regenerated
for ABI 14 to match the Go runtime. See that directory's README for reproduction.

The language provider extracts explicit imports, including `public`, `meta`, and
`all` modifiers, and supports Unicode and `«quoted»` module-name components.
It maps module names to `.lean` paths under the nearest Lake package's source
roots, honoring package and target `srcDir` in `lakefile.toml`. This prevents a
sibling file such as `AesopTest/Aesop.lean` from shadowing the module `Aesop`,
and prevents external `Lean.Expr` from matching `Batteries/Lean/Expr.lean`.
Nested Lake packages form boundaries; standalone sources without a Lake package
use an ancestor-directory heuristic, nearest first. External imports and implicit
prelude imports do not create edges.

Executable `lakefile.lean` configuration, `LEAN_PATH`, compiled `.olean` files,
and dependency package search paths are not evaluated. An uninterpreted Lake
configuration still establishes a package boundary and uses its directory as
the default source root. If both config formats exist, `lakefile.lean` takes
precedence, matching Lake; the dormant TOML file supplies neither source roots
nor test-driver classification. This is source import analysis, not Lean
elaboration or declaration-level proof dependency analysis. Syntax errors in
declarations do not discard successfully parsed imports from the header.
Test detection uses `test`/`tests` directories (case insensitive), `_test.lean`,
and `test_` prefixes; theorem and example declarations are ordinary code.
It also reads the nearest `lakefile.toml` through the source content reader:
the configured `testDriver` library's root modules and submodules are tests,
as is an executable driver's root module. Library `roots`, executable `root`,
and package/target `srcDir` overrides are honored. Nested Lake packages form
boundaries. This classification does not evaluate `lakefile.lean`, custom
library `globs`, scripts, or test drivers in other packages, and does not mark
production modules merely because the test driver imports them.

### Quint

Quint (`.qnt`) uses [LegacyCodeHQ/tree-sitter-quint](https://github.com/LegacyCodeHQ/tree-sitter-quint)
0.33.0. The Apache-2.0 parser, scanner, recognition tables, and attribution
notices are pinned under `internal/tree_sitter/quint/`. The parser is regenerated
for ABI 14 to match Clarity's Go runtime; normal builds need no Quint or Node
installation. See the vendored README for reproduction details.

File dependencies come from explicit `from` clauses in named, wildcard, aliased,
and instantiated imports. Paths resolve relative to the importing file, with
`.qnt` appended unconditionally, matching Quint 0.33. String contents are literal;
backslashes are not unescaped. Only supplied project files produce edges.
Imports without `from` and exports refer to already loaded modules; Clarity
does not guess sibling filenames or infer additional edges for module references
through a transitive import context. Multiple modules in one file stay one node.

Test detection uses `Test.qnt`, `Tests.qnt`, `_test.qnt`, `_tests.qnt`, `test_`
prefixes, and `test`/`tests` directories. It also recognizes top-level definitions
whose names end in `Test`, matching the default `quint test` name filter.
Comments, strings, local definitions, and `run` witnesses without a `Test` suffix
do not by themselves mark a file as a test. Quint support starts at Basic Tests.

### Profiling Lean and Quint

Both providers have opt-in corpus benchmarks. Each operation performs a complete
dependency-resolution and test-classification pass, creates a fresh resolver,
and reads source/configuration files through the filesystem reader. Setup and
corpus discovery are outside the measured region. Ordinary test runs do not
require external repositories.

CPU and allocation profiles identified repeated Lake directory/configuration
lookups and tree-sitter node wrappers as hot paths. Lean now bounds tree-sitter
input to the import header, accounting for nested comments and quoted names;
the grammar still validates the header and extracts its identifier nodes.
Lake lookup results, including absent configurations, are cached only within
one resolver/reader snapshot and shared safely by its concurrent workers.
Sources without imports do not trigger configuration discovery.

Quint uses separate queries for explicit file imports and local test definitions.
A missing literal `from` or `Test` token rules out the corresponding analysis
without parsing; positive matches still go through tree-sitter. Queries retain
the original direct-module declaration scope, so comments, strings, and nested
local definitions do not become dependencies or tests.

Measured on 2026-11-06 with Go 1.27.2, darwin/arm64, Apple M5 Max, compared with
`eded143`. Times and Go allocations are medians of three unprofiled 2-second
runs, per complete corpus pass:

| Corpus | Before | After | Go bytes before | Go bytes after |
| --- | ---: | ---: | ---: | ---: |
| Batteries, 275 Lean files | 158.23 ms | 8.03 ms | 5,994,688 | 3,407,217 |
| Official Quint examples, 77 files | 255.79 ms | 109.62 ms | 3,191,699 | 1,575,808 |

This is 19.7x and 2.3x faster, with 43% and 51% less Go allocation traffic.
Go allocation counts fell from 57,603 to 29,556 and 24,865 to 7,482 per pass.
Lean filesystem read attempts fell from 3,659 to 2,045 per pass. The workloads
retain exactly 453/24 file edges and 76/25 test files, respectively.

Go heap profiles do not include all tree-sitter C allocations. Separate
100-pass standalone-process measurements with macOS `/usr/bin/time -l` recorded
peak RSS of 31,408,128 -> 22,167,552 bytes for Lean and
58,228,736 -> 50,511,872 bytes for Quint (29% and 13% lower). Peak RSS includes
runtime and native memory and can vary between runs. These corpus results do
not imply the same speedup for whole CLI startup, rendering, or Git I/O.

To reproduce, build standalone test binaries so process memory excludes the Go
compiler. Use a source checkout for each environment variable:

```sh
go test -c -o /tmp/clarity-lean-bench ./depgraph/languages/lean
go test -c -o /tmp/clarity-quint-bench ./depgraph/languages/quint

CLARITY_LEAN_BENCH_REPO=/path/to/batteries /tmp/clarity-lean-bench \
  -test.run '^$' -test.bench BenchmarkLeanCorpus -test.benchtime=2s -test.count=3 -test.benchmem
CLARITY_QUINT_BENCH_REPO=/path/to/quint/examples /tmp/clarity-quint-bench \
  -test.run '^$' -test.bench BenchmarkQuintCorpus -test.benchtime=2s -test.count=3 -test.benchmem

CLARITY_LEAN_BENCH_REPO=/path/to/batteries /tmp/clarity-lean-bench \
  -test.run '^$' -test.bench BenchmarkLeanCorpus -test.benchtime=3s \
  -test.cpuprofile=/tmp/lean.cpu -test.memprofile=/tmp/lean.heap
go tool pprof -top /tmp/clarity-lean-bench /tmp/lean.cpu
go tool pprof -top -sample_index=alloc_space /tmp/clarity-lean-bench /tmp/lean.heap

CLARITY_LEAN_BENCH_REPO=/path/to/batteries /usr/bin/time -l /tmp/clarity-lean-bench \
  -test.run '^$' -test.bench BenchmarkLeanCorpus -test.benchtime=100x
```

Use the equivalent Quint benchmark binary, environment variable, and benchmark
name for its CPU/heap/RSS profiles. Keep profiled runs separate from timing
comparisons. Re-run the reference-parser checks in `scripts/oracle-diff/README.md`
when changing either integration, and the race tests for snapshot-cache changes.

### Supporting Additional Languages

1. Create a new language directory under `internal/tree_sitter/` and build and copy the appropriate tree-sitter files.
2. Check the `internal/tree_sitter/dart` directory for a reference implementation of the Go bindings.

## Resources

- [go-tree-sitter](https://github.com/smacker/go-tree-sitter)
- [tree-sitter-dart](https://github.com/UserNobody14/tree-sitter-dart)
- [tree-sitter-zig](https://github.com/tree-sitter-grammars/tree-sitter-zig)
