#!/usr/bin/env node
// Compare explicit file imports and local test definitions with Quint's ANTLR IR.
// Requires an installed @informalsystems/quint package; never executes specs.
const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');

function run(repo, command, args) {
  return execFileSync(command, args, { cwd: repo, encoding: 'utf8', maxBuffer: 32 * 1024 * 1024 });
}

function main() {
  const args = process.argv.slice(2);
  const repo = path.resolve(args.shift() || '.');
  const options = { '--scope': '.' };
  while (args.length) {
    const option = args.shift();
    if (!['--scope', '--clarity', '--quint-package'].includes(option) || !args.length) {
      throw new Error('Usage: node quint_import_diff.cjs REPO --scope DIR --clarity BINARY --quint-package PACKAGE_DIR');
    }
    options[option] = args.shift();
  }
  if (!options['--clarity'] || !options['--quint-package']) {
    throw new Error('--clarity and --quint-package are required');
  }
  const clarity = path.resolve(options['--clarity']);
  const quintRoot = path.resolve(options['--quint-package']);
  const scope = options['--scope'];
  const { parsePhase1fromText } = require(path.join(quintRoot, 'dist/src/parsing/quintParserFrontend.js'));
  const { newIdGenerator } = require(path.join(quintRoot, 'dist/src/idGenerator.js'));
  const referenceVersion = JSON.parse(fs.readFileSync(path.join(quintRoot, 'package.json'), 'utf8')).version;
  const files = run(repo, 'git', ['ls-files', '--cached', '--others', '--exclude-standard', '-z', '--', scope])
    .split('\0').filter(file => file.endsWith('.qnt'));
  if (!files.length) throw new Error('No supplied .qnt files found');
  const supplied = new Set(files.map(file => path.resolve(repo, file)));
  const expectedEdges = new Set();
  const expectedTests = new Set();
  const parseErrors = [];
  for (const relative of files) {
    const file = path.resolve(repo, relative);
    const result = parsePhase1fromText(newIdGenerator(), fs.readFileSync(file, 'utf8'), file);
    if (result.errors.length) {
      parseErrors.push({ file: relative, errors: result.errors.map(error => error.message) });
      continue;
    }
    const stem = path.basename(file, '.qnt');
    const normalized = relative.split(path.sep).join('/');
    if (/(?:Test|Tests|_test|_tests)$/.test(stem) || stem.startsWith('test_') || /(^|\/)(test|tests)\//.test(normalized)) {
      expectedTests.add(normalized);
    }
    for (const module of result.modules) {
      for (const declaration of module.declarations) {
        if (declaration.kind === 'def' && declaration.name.endsWith('Test')) expectedTests.add(normalized);
        if ((declaration.kind === 'import' || declaration.kind === 'instance') && declaration.fromSource) {
          const target = path.join(path.dirname(file), declaration.fromSource + '.qnt');
          if (target !== file && supplied.has(target)) {
            expectedEdges.add(JSON.stringify([normalized, path.relative(repo, target).split(path.sep).join('/')]));
          }
        }
      }
    }
  }
  const dot = run(repo, clarity, ['show', scope, '--include-ext', '.qnt', '--no-stats', '-f', 'dot']);
  const quoted = String.raw`"(?:[^"\\]|\\.)*"`;
  const edges = new Set([...dot.matchAll(new RegExp(`(${quoted}) -> (${quoted});`, 'g'))]
    .map(match => JSON.stringify([JSON.parse(match[1]), JSON.parse(match[2])])));
  const nodes = new Set([...dot.matchAll(new RegExp(`^  (${quoted}) \\[`, 'gm'))].map(match => JSON.parse(match[1])));
  const tests = new Set([...dot.matchAll(new RegExp(`^  (${quoted}) \\[.*fillcolor=lightgreen\\];$`, 'gm'))]
    .map(match => JSON.parse(match[1])));
  const difference = (a, b) => [...a].filter(value => !b.has(value)).sort();
  const suppliedNames = new Set(files.map(file => file.split(path.sep).join('/')));
  const report = {
    repo: path.basename(repo), scope, referenceVersion,
    revision: run(repo, 'git', ['rev-parse', 'HEAD']).trim(),
    files: files.length, edges: edges.size, tests: tests.size, parseErrors,
    missingNodes: difference(suppliedNames, nodes), extraNodes: difference(nodes, suppliedNames),
    missingEdges: difference(expectedEdges, edges), extraEdges: difference(edges, expectedEdges),
    missingTests: difference(expectedTests, tests), extraTests: difference(tests, expectedTests),
  };
  report.passed = Object.entries(report).every(([key, value]) =>
    !(key.startsWith('missing') || key.startsWith('extra') || key === 'parseErrors') || value.length === 0);
  console.log(JSON.stringify(report, null, 2));
  return report.passed ? 0 : 1;
}

process.exitCode = main();
