# Clarity v0.35.0

Clarity adds Lean 4 and Quint dependency graphs, speeds up their analysis, and
refreshes the live watch viewer.

## New language support

- **Lean 4 (`.lean`)**: explicit module imports, Unicode and quoted module names,
  Lake TOML source roots, and test-driver classification. Package boundaries
  prevent sibling filenames from shadowing imported modules.
- **Quint (`.qnt`)**: explicit `from` imports, including instantiated modules,
  and test detection from filenames and top-level `…Test` definitions. Uses
  LegacyCodeHQ's Quint 0.33 tree-sitter grammar.
- Both languages start at **Basic Tests** maturity. Lean support was checked
  against Batteries, Aesop, Quote4, and DXN. Quint support was compared with its
  ANTLR reference parser on the official examples, Tendermint, and Mysticeti.

## Faster language analysis

CPU and memory profiling guided header-only Lean parsing, per-analysis Lake
lookup caching, and targeted Quint import/test queries.

| Complete provider pass | Before | After | Go bytes allocated |
| --- | ---: | ---: | ---: |
| Batteries, 275 Lean files | 158 ms | 8 ms | 43% lower |
| Official Quint examples, 77 files | 256 ms | 110 ms | 51% lower |

Separate 100-pass runs measured peak process memory reductions of 29% for Lean
and 13% for Quint. These are corpus-provider measurements, not whole-CLI startup,
rendering, or Git I/O guarantees. Benchmarks, profiling commands, and methodology
are documented in [Language Parsing](https://github.com/LegacyCodeHQ/clarity-cli/blob/v0.35.0/docs/development/language-parsing.md).

## Watch viewer

- A unified top bar combines the source selector and worktree tabs.
- Worktree tabs show changed-file counts and unseen-change indicators.
- Overflow menus preserve access to worktrees that do not fit in the tab bar.
- Tab sizing, selection feedback, badges, typography, and connection indicators
  have been refined.

## Packaging and fixes

- Windows release archives use ZIP, matching the npm installer's download URL.
- Binary archives and the npm package include vendored grammar licenses/notices.
- Lean import resolution honors Lake configuration-file precedence and preserves
  literal opening guillemets inside quoted module-name components.
- Release binaries are built with Go 1.26.10.

## Current scope

Analysis is at file granularity. Lean does not evaluate executable
`lakefile.lean` configuration or elaborate proofs/custom syntax. Quint imports
without `from` refer to loaded modules and do not produce guessed sibling-file
dependencies.

## Upgrade

```sh
npm install -g @legacycodehq/clarity@0.35.0
# or
brew upgrade LegacyCodeHQ/tap/clarity
```

**Full changelog:** https://github.com/LegacyCodeHQ/clarity-cli/compare/v0.34.2...v0.35.0
