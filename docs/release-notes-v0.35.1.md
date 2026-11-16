# Clarity v0.35.1

Clarity fixes Rust paths inside inline modules and ships security updates for
the Go toolchain and the watch viewer's build tools.

## Rust dependency fix

Paths written inside an inline `mod tests { … }` block were resolved as if they
were written at file scope, so each `super` pointed one module too high:

- `super::decode()` in a test named the file's own function but produced an
  edge to the parent `mod.rs`, which `clarity cycles` reported as a false cycle.
- `super::super::profiler::run()` named a sibling file but produced no edge, so
  a real cycle through a test stayed hidden.

Clarity now tracks enclosing inline modules in both `use` statements and
qualified paths, rebases relative paths to file scope, and drops paths that
stay within the file. On Rust projects this can remove false edges to `lib.rs`
or `mod.rs` and reveal previously hidden edges, including cycles through test
modules. Expect `clarity cycles` results to change where those existed.

## Security

- Release binaries are built with **Go 1.26.11**, fixing GO-2026-6859
  (CVE-2026-97054), an HTTP/1 server crash in `net/http` that affects the
  `clarity watch` server. v0.35.0 was built with Go 1.26.10.
- The watch viewer's development dependencies were updated: Vite 8.3.6
  (GHSA-9jrq-w75r-8gcw, GHSA-rq7h-c2jc-7f22 and three launch-editor advisories),
  Vitest 4.1.12 (GHSA-qxg6-89mg-rqv8, GHSA-xp26-3454-mjgp) and fflate 0.8.3
  (GHSA-px8p-9vwx-vf98). These are build and test tools; none ship in the
  viewer bundle.
- deepmerge (GHSA-gcqx-w48m-w2h4, low) has no fixed release. It is a build-time
  dependency of the Svelte Vite plugin and only merges the project's own Vite
  configuration.

## Compatibility

No command, flag or output format changes. Binary size is unchanged
(darwin/arm64: 54.1 MiB).
