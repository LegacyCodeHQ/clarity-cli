# tree-sitter Lean 4

Grammar and scanner from [Julian/tree-sitter-lean](https://github.com/Julian/tree-sitter-lean),
pinned to commit `0d28e5d6e8c4755f5df7ef9bde3b6ec6f59fae94`.
The MIT license is preserved in `LICENSE`.

Files copied from upstream:

- `src/parser.c` -> `parser.c`
- `src/scanner.c` -> `scanner.c`
- `src/tree_sitter/*.h` -> `tree_sitter/*.h`

Upstream ships ABI 15. Before copying, regenerate at that revision with
tree-sitter CLI 0.27.1 using `tree-sitter generate --abi 14` to match Clarity's
`smacker/go-tree-sitter` runtime. Do not change the ABI version constant by hand.
No grammar or scanner changes are required. The generated parser is about 47 MiB.
Normal Go builds require CGO but do not need Lean, Lake, Node, the tree-sitter CLI,
or network access.

This is an experimental static grammar. Lean's user-defined syntax, notation,
and macros cannot all be parsed faithfully. Clarity reads explicit import nodes
from the file header and tolerates parse errors in the declaration body.
