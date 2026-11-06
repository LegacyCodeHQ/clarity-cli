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

### Supporting Additional Languages

1. Create a new language directory under `internal/tree_sitter/` and build and copy the appropriate tree-sitter files.
2. Check the `internal/tree_sitter/dart` directory for a reference implementation of the Go bindings.

## Resources

- [go-tree-sitter](https://github.com/smacker/go-tree-sitter)
- [tree-sitter-dart](https://github.com/UserNobody14/tree-sitter-dart)
- [tree-sitter-zig](https://github.com/tree-sitter-grammars/tree-sitter-zig)
