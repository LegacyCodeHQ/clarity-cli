#!/usr/bin/env python3
"""Compare Clarity with an independent static Lean import-header reader.

Requires Python 3.11+. Reads clean, flat Lake TOML packages without executing
Lean, Lake, or repository code. This checks source imports, not elaboration.
"""

import argparse
import json
from pathlib import Path
import re
import subprocess
import tomllib


IDENTIFIER = re.compile(r"(?:«[^»\n]+»|[\w'!?]+)(?:\.(?:«[^»\n]+»|[\w'!?]+))*")
DOT_STRING = r'"(?:[^"\\]|\\.)*"'


def run(repo, *args):
    return subprocess.check_output(args, cwd=repo, text=True)


def header_tokens(source):
    """Read identifiers, skipping whitespace, line comments and nested comments.

    Stop at punctuation; the caller stops at the first non-header command.
    Imports mentioned inside comments or declaration strings cannot be edges.
    """
    pos = 0
    while pos < len(source):
        if source[pos].isspace():
            pos += 1
        elif source.startswith("--", pos):
            end = source.find("\n", pos)
            pos = len(source) if end < 0 else end + 1
        elif source.startswith("/-", pos):
            pos += 2
            depth = 1
            while pos < len(source) and depth:
                if source.startswith("/-", pos):
                    depth += 1
                    pos += 2
                elif source.startswith("-/", pos):
                    depth -= 1
                    pos += 2
                else:
                    pos += 1
            if depth:
                raise ValueError("unterminated header comment")
        else:
            match = IDENTIFIER.match(source, pos)
            if not match:
                return
            yield match[0]
            pos = match.end()


def imports(source):
    tokens = iter(header_tokens(source))
    token = next(tokens, None)
    if token == "module":
        token = next(tokens, None)
    if token == "prelude":
        token = next(tokens, None)
    while token:
        if token == "public":
            token = next(tokens, None)
        if token == "meta":
            token = next(tokens, None)
        if token != "import":
            return
        token = next(tokens, None)
        if token == "all":
            token = next(tokens, None)
        if not token:
            raise ValueError("missing import name")
        yield token
        token = next(tokens, None)


def module_stem(name):
    parts = [quoted or plain for quoted, plain in re.findall(r"«([^»]+)»|([^.«»]+)", name)]
    return Path(*parts)


def check(repo, clarity, snapshot):
    repo = repo.resolve()
    if run(repo, "git", "status", "--porcelain").strip():
        raise ValueError(f"{repo}: validation requires a clean checkout")
    tracked = run(repo, "git", "ls-files", "-z").split("\0")
    files = {Path(path) for path in tracked if path.endswith(".lean")}
    if "lakefile.toml" not in tracked or "lakefile.lean" in tracked:
        raise ValueError(f"{repo}: only root lakefile.toml packages are supported")
    nested = [Path(path).parent for path in tracked if Path(path).name in {"lakefile.toml", "lakefile.lean"} and "/" in path]
    if any(source.is_relative_to(directory) for directory in nested for source in files):
        raise ValueError(f"{repo}: nested package configurations need a separate check")
    config = tomllib.loads((repo / "lakefile.toml").read_text())
    package_root = Path(config.get("srcDir", "."))
    roots = {package_root}
    for kind in ("lean_lib", "lean_exe"):
        roots.update(package_root / target.get("srcDir", ".") for target in config.get(kind, []))

    expected = set()
    for source in sorted(files):
        for name in imports((repo / source).read_text(encoding="utf-8-sig")):
            relative = Path(str(module_stem(name)) + ".lean")
            targets = {root / relative for root in roots} & files
            if len(targets) > 1:
                raise ValueError(f"{repo}: ambiguous module {name}: {targets}")
            expected.update((str(source), str(target)) for target in targets if source != target)

    # Check the library test-driver namespace independently of Clarity's parser.
    expected_tests = set()
    driver = config.get("testDriver")
    if driver:
        target = next((lib for lib in config.get("lean_lib", []) if lib["name"] == driver), None)
        if target is None:
            raise ValueError(f"{repo}: this harness supports library test drivers only")
        source_dir = package_root / target.get("srcDir", ".")
        for root in target.get("roots", [target["name"]]):
            stem = source_dir / module_stem(root)
            expected_tests.update(str(path) for path in files if str(path) == str(stem) + ".lean" or path.is_relative_to(stem))

    args = [clarity, "show", "--all", "--include-ext", ".lean", "--no-stats", "-f", "dot"]
    if snapshot:
        args.extend(["-c", "HEAD"])
    dot = run(repo, *args)
    actual = {(json.loads(a), json.loads(b)) for a, b in re.findall(rf"({DOT_STRING}) -> ({DOT_STRING});", dot)}
    nodes = {json.loads(node) for node in re.findall(rf"^  ({DOT_STRING}) \[", dot, re.M)}
    tests = {json.loads(node) for node in re.findall(rf"^  ({DOT_STRING}) \[.*fillcolor=lightgreen\];$", dot, re.M)}
    result = {
        "repo": repo.name,
        "revision": run(repo, "git", "rev-parse", "HEAD").strip(),
        "snapshot": "HEAD" if snapshot else "working_tree",
        "files": len(files), "edges": len(actual), "tests": len(tests),
        "missing_nodes": sorted({str(path) for path in files} - nodes),
        "extra_nodes": sorted(nodes - {str(path) for path in files}),
        "missing_edges": sorted(expected - actual),
        "extra_edges": sorted(actual - expected),
        "missing_tests": sorted(expected_tests - tests),
        "extra_tests": sorted(tests - expected_tests),
    }
    result["passed"] = not any(result[key] for key in result if key.startswith(("missing_", "extra_")))
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("repos", type=Path, nargs="+")
    parser.add_argument("--clarity", type=lambda path: str(Path(path).resolve()), required=True)
    parser.add_argument("--snapshot", action="store_true", help="also validate Clarity's Git HEAD reader")
    args = parser.parse_args()
    results = [check(repo, args.clarity, snapshot) for repo in args.repos for snapshot in ([False, True] if args.snapshot else [False])]
    print(json.dumps(results, indent=2))
    return 0 if all(result["passed"] for result in results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
