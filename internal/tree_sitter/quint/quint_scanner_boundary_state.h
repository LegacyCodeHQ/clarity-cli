#ifndef QUINT_SCANNER_BOUNDARY_STATE_H
#define QUINT_SCANNER_BOUNDARY_STATE_H
#include "quint_boundaries.h"

#ifndef Q_INDEX_STEP
#define Q_INDEX_STEP() ((void)0)
#endif
#ifndef Q_SUFFIX_SEARCH_STEP
#define Q_SUFFIX_SEARCH_STEP(scanner) ((void)0)
#endif

// Intern persistent stacks: scanner serialization stores a small stable ID,
// rather than imposing a nesting limit through Tree-sitter's 1024-byte buffer.
typedef struct { uint32_t parent, start; uint8_t kind; } QDefinitionStack;
typedef struct {
  QDocument document;
  char *source;
  uint32_t *suffixes, *suffix_ranks, *positions;
  size_t position_count;
  uint32_t position;
  bool recovering;
  uint64_t end_bytes[4];
  uint32_t end_token;
  unsigned end_prefix;
  bool end_cached;
  uint64_t begin_bytes[2][4];
  bool begin_cached;
  size_t length;
  uint64_t fingerprint;
  bool recognized;
  QBoundary *boundary_index;
  uint32_t *stack_slots;
  size_t slot_capacity;
  QDefinitionStack *stacks;
  size_t stack_count, stack_capacity;
  uint32_t stack;
} QScanner;

static size_t q_utf8(int32_t c, char bytes[4]) {
  if (c < 0x80) { bytes[0] = (char)c; return 1; }
  if (c < 0x800) { bytes[0] = (char)(0xc0 | (c >> 6)); bytes[1] = (char)(0x80 | (c & 0x3f)); return 2; }
  if (c < 0x10000) { bytes[0] = (char)(0xe0 | (c >> 12)); bytes[1] = (char)(0x80 | ((c >> 6) & 0x3f)); bytes[2] = (char)(0x80 | (c & 0x3f)); return 3; }
  bytes[0] = (char)(0xf0 | (c >> 18)); bytes[1] = (char)(0x80 | ((c >> 12) & 0x3f)); bytes[2] = (char)(0x80 | ((c >> 6) & 0x3f)); bytes[3] = (char)(0x80 | (c & 0x3f)); return 4;
}
// Prefix-doubling suffix sorting lets a zero-width boundary marker locate its
// source position after reading only a distinguishing prefix, instead of
// rescanning the entire remaining file for every definition.
static void q_suffix_sort(const uint32_t *input, uint32_t *output, size_t count,
                          const uint32_t *rank, size_t distance,
                          size_t max_rank, uint32_t *buckets) {
  memset(buckets, 0, (max_rank + 1) * sizeof(uint32_t));
  for (size_t i = 0; i < count; i++) {
    size_t at = (size_t)input[i] + distance;
    buckets[at < count ? rank[at] : 0]++;
  }
  uint32_t offset = 0;
  for (size_t i = 0; i <= max_rank; i++) {
    uint32_t width = buckets[i]; buckets[i] = offset; offset += width;
  }
  for (size_t i = 0; i < count; i++) {
    size_t at = (size_t)input[i] + distance;
    output[buckets[at < count ? rank[at] : 0]++] = input[i];
  }
}
static bool q_build_suffixes(QScanner *scanner) {
  size_t count = scanner->length + 1;
  if (count > UINT32_MAX || count > SIZE_MAX / sizeof(uint32_t)) return false;
  uint32_t *suffixes = malloc(count * sizeof(uint32_t));
  uint32_t *rank = malloc(count * sizeof(uint32_t));
  uint32_t *next_rank = malloc(count * sizeof(uint32_t));
  uint32_t *work = malloc(count * sizeof(uint32_t));
  size_t bucket_count = count > 257 ? count : 257;
  uint32_t *buckets = calloc(bucket_count, sizeof(uint32_t));
  if (!suffixes || !rank || !next_rank || !work || !buckets) {
    free(suffixes); free(rank); free(next_rank); free(work); free(buckets);
    return false;
  }
  for (size_t i = 0; i < count; i++) {
    suffixes[i] = (uint32_t)i;
    rank[i] = i < scanner->length ? (unsigned char)scanner->source[i] + 1u : 0;
  }
  size_t max_rank = 256;
  for (size_t distance = 1; distance < count; distance *= 2) {
    q_suffix_sort(suffixes, work, count, rank, distance, max_rank, buckets);
    q_suffix_sort(work, suffixes, count, rank, 0, max_rank, buckets);
    uint32_t classes = 0;
    next_rank[suffixes[0]] = 0;
    for (size_t i = 1; i < count; i++) {
      size_t a = suffixes[i - 1], b = suffixes[i];
      uint32_t ar = a + distance < count ? rank[a + distance] : 0;
      uint32_t br = b + distance < count ? rank[b + distance] : 0;
      if (rank[a] != rank[b] || ar != br) classes++;
      next_rank[b] = classes;
    }
    uint32_t *swap = rank; rank = next_rank; next_rank = swap;
    max_rank = classes;
    if ((size_t)classes + 1 == count) break;
  }
  free(next_rank); free(work); free(buckets);
  free(scanner->suffixes); scanner->suffixes = suffixes;
  free(scanner->suffix_ranks); scanner->suffix_ranks = rank;
  return true;
}
static int q_suffix_byte(const QScanner *scanner, size_t index, size_t consumed) {
  size_t at = (size_t)scanner->suffixes[index] + consumed;
  return at < scanner->length ? (unsigned char)scanner->source[at] : -1;
}
// Find a unique suffix inside the reference-boundary window. The inverse
// suffix ranks make this independent of identical text elsewhere in the file.
static uint32_t q_window_position(const QScanner *scanner, size_t low, size_t high,
                                  size_t start, size_t end) {
  uint32_t found = UINT32_MAX;
  if (high - low < end - start + 1) {
    for (size_t i = low; i < high; i++) {
      uint32_t at = scanner->suffixes[i];
      if (at < start || at > end) continue;
      if (found != UINT32_MAX) return UINT32_MAX;
      found = at;
    }
  } else {
    for (size_t at = start; at <= end; at++) {
      uint32_t rank = scanner->suffix_ranks[at];
      if (rank < low || rank >= high) continue;
      if (found != UINT32_MAX) return UINT32_MAX;
      found = (uint32_t)at;
    }
  }
  return found;
}
static uint32_t q_locate_source(QScanner *scanner, TSLexer *lexer, size_t start, size_t end) {
  if (lexer->eof(lexer)) return (uint32_t)scanner->length;
  size_t low = 0, high = scanner->length + 1, consumed = 0;
  while (low < high) {
    uint32_t unique = q_window_position(scanner, low, high, start, end);
    if (unique != UINT32_MAX) return unique;
    char bytes[4];
    size_t width = lexer->eof(lexer) ? 1 : q_utf8(lexer->lookahead, bytes);
    for (size_t i = 0; i < width; i++) {
      int key = lexer->eof(lexer) ? -1 : (unsigned char)bytes[i];
      // Every suffix in this interval shares the consumed prefix. If both
      // lexicographic endpoints have the next byte, every interior suffix does
      // too: the interval and its position ambiguity cannot change. Keep reading
      // without repeating two binary searches and the same uniqueness check.
      if (q_suffix_byte(scanner, low, consumed) == key &&
          q_suffix_byte(scanner, high - 1, consumed) == key) {
        consumed++; continue;
      }
      size_t left = low, right = high;
      while (left < right) {
        Q_SUFFIX_SEARCH_STEP(scanner);
        size_t mid = left + (right - left) / 2;
        if (q_suffix_byte(scanner, mid, consumed) < key) left = mid + 1;
        else right = mid;
      }
      low = left; right = high;
      while (left < right) {
        Q_SUFFIX_SEARCH_STEP(scanner);
        size_t mid = left + (right - left) / 2;
        if (q_suffix_byte(scanner, mid, consumed) <= key) left = mid + 1;
        else right = mid;
      }
      high = left; consumed++;
      if (low == high) return UINT32_MAX;
      unique = q_window_position(scanner, low, high, start, end);
      if (unique != UINT32_MAX) return unique;
    }
    if (lexer->eof(lexer)) return UINT32_MAX;
    lexer->advance(lexer, false);
  }
  return UINT32_MAX;
}

// Zed's Wasm runtime has no qsort, so sort in place with a heap sort. The comparators below are
// total orders, so the unstable order does not matter.
static void q_swap_bytes(unsigned char *left, unsigned char *right, size_t size) {
  while (size--) {
    unsigned char byte = *left;
    *left++ = *right;
    *right++ = byte;
  }
}
static void q_sift_down(unsigned char *base, size_t size, size_t root, size_t count,
                        int (*compare)(const void *, const void *)) {
  for (;;) {
    size_t child = 2 * root + 1;
    if (child >= count) return;
    if (child + 1 < count && compare(base + child * size, base + (child + 1) * size) < 0) child++;
    if (compare(base + root * size, base + child * size) >= 0) return;
    q_swap_bytes(base + root * size, base + child * size, size);
    root = child;
  }
}
static void q_sort(void *items, size_t count, size_t size,
                   int (*compare)(const void *, const void *)) {
  unsigned char *base = items;
  if (count < 2) return;
  for (size_t i = count / 2; i-- > 0;) q_sift_down(base, size, i, count, compare);
  for (size_t end = count - 1; end > 0; end--) {
    q_swap_bytes(base, base + end * size, size);
    q_sift_down(base, size, 0, end, compare);
  }
}

static int q_position_compare(const void *left, const void *right) {
  uint32_t a = *(const uint32_t *)left, b = *(const uint32_t *)right;
  return (a > b) - (a < b);
}
static bool q_build_positions(QScanner *scanner) {
  size_t count = scanner->document.boundary_count;
  if (count > SIZE_MAX / (2 * sizeof(uint32_t))) return false;
  uint32_t *positions = count ? malloc(count * 2 * sizeof(uint32_t)) : NULL;
  if (count && !positions) return false;
  for (size_t i = 0; i < count; i++) {
    positions[2*i] = scanner->document.boundaries[i].start;
    positions[2*i+1] = scanner->document.boundaries[i].end;
  }
  if (count) q_sort(positions, count * 2, sizeof(uint32_t), q_position_compare);
  free(scanner->positions); scanner->positions = positions;
  scanner->position_count = count * 2;
  return true;
}
// Sort a copy: reference traversal order remains untouched.
static int q_boundary_compare(const void *left, const void *right) {
  const QBoundary *a = left, *b = right;
  if (a->start != b->start) return (a->start > b->start) - (a->start < b->start);
  if (a->kind != b->kind) return (a->kind > b->kind) - (a->kind < b->kind);
  return (a->end > b->end) - (a->end < b->end);
}
static bool q_build_boundary_index(QScanner *scanner) {
  size_t count = scanner->document.boundary_count;
  if (count > SIZE_MAX / sizeof(QBoundary)) return false;
  QBoundary *index = count ? malloc(count * sizeof(QBoundary)) : NULL;
  if (count && !index) return false;
  if (count) {
    memcpy(index, scanner->document.boundaries, count * sizeof(QBoundary));
    q_sort(index, count, sizeof(QBoundary), q_boundary_compare);
  }
  free(scanner->boundary_index); scanner->boundary_index = index;
  return true;
}
static bool q_has_boundary(const QScanner *scanner, uint32_t start, uint8_t kind,
                           uint32_t end, bool exact) {
  size_t low = 0, high = scanner->document.boundary_count;
  QBoundary key = {.start = start, .end = exact ? end : 0, .kind = kind};
  while (low < high) {
    Q_INDEX_STEP();
    size_t mid = low + (high - low) / 2;
    if (q_boundary_compare(&scanner->boundary_index[mid], &key) < 0) low = mid + 1;
    else high = mid;
  }
  if (low == scanner->document.boundary_count) return false;
  QBoundary b = scanner->boundary_index[low];
  return b.start == start && b.kind == kind && (!exact || b.end == end);
}
// Negative filter only: a possible character still requires the original exact
// position lookup. Include the whole token and its leading skipped text, since
// recovery can enter either at an interior byte or before a skipped comment.
static void q_marker_bytes(const QScanner *scanner, uint32_t index, unsigned prefix, uint64_t bytes[4]) {
  size_t start = index ? scanner->document.tokens[index - 1].end : 0;
  QToken token = scanner->document.tokens[index];
  size_t end = token.kind == -1 ? scanner->length : token.end;
  end += prefix <= scanner->length - end ? prefix : scanner->length - end;
  for (size_t at = start; at < end; at++) {
    unsigned c = (unsigned char)scanner->source[at];
    bytes[c / 64] |= UINT64_C(1) << (c % 64);
  }
}
static bool q_begin_may_match(QScanner *scanner, TSLexer *lexer, bool definition, bool declaration) {
  if (!definition && !declaration) return false;
  if (lexer->eof(lexer)) return true;
  if (!scanner->begin_cached) {
    memset(scanner->begin_bytes, 0, sizeof(scanner->begin_bytes));
    for (size_t i = 0; i < scanner->document.boundary_count; i++) {
      QBoundary b = scanner->document.boundaries[i];
      if ((b.kind == 3 || b.kind == 4) && b.start < scanner->document.token_count)
        q_marker_bytes(scanner, b.start, 0, scanner->begin_bytes[b.kind - 3]);
    }
    scanner->begin_cached = true;
  }
  char bytes[4]; q_utf8(lexer->lookahead, bytes);
  unsigned byte = (unsigned char)bytes[0];
  uint64_t mask = (declaration ? scanner->begin_bytes[0][byte / 64] : 0)
    | (definition ? scanner->begin_bytes[1][byte / 64] : 0);
  return (mask >> (byte % 64)) & 1;
}
static bool q_end_may_match(QScanner *scanner, TSLexer *lexer, unsigned prefix,
                            bool definition, bool declaration) {
  if (!scanner->stack || scanner->stack > scanner->stack_count) return false;
  QDefinitionStack top = scanner->stacks[scanner->stack - 1];
  if (!((top.kind == 4 && definition) || (top.kind == 3 && declaration))) return false;
  if (lexer->eof(lexer)) return true;
  char bytes[4]; q_utf8(lexer->lookahead, bytes);
  unsigned byte = (unsigned char)bytes[0];
  size_t low = 0, high = scanner->document.boundary_count;
  QBoundary key = {.start = top.start, .kind = top.kind};
  while (low < high) {
    size_t mid = low + (high - low) / 2;
    if (q_boundary_compare(&scanner->boundary_index[mid], &key) < 0) low = mid + 1;
    else high = mid;
  }
  for (; low < scanner->document.boundary_count; low++) {
    QBoundary b = scanner->boundary_index[low];
    if (b.start != top.start || b.kind != top.kind) break;
    if (b.end >= scanner->document.token_count) continue;
    if (!scanner->end_cached || scanner->end_token != b.end || scanner->end_prefix != prefix) {
      memset(scanner->end_bytes, 0, sizeof(scanner->end_bytes));
      q_marker_bytes(scanner, b.end, prefix, scanner->end_bytes);
      scanner->end_token = b.end; scanner->end_prefix = prefix; scanner->end_cached = true;
    }
    if ((scanner->end_bytes[byte / 64] >> (byte % 64)) & 1) return true;
  }
  return false;
}
static size_t q_stack_hash(uint32_t parent, uint32_t start, uint8_t kind) {
  uint64_t x = ((uint64_t)parent << 32) | start;
  x ^= (uint64_t)kind * UINT64_C(0x9e3779b97f4a7c15);
  x = (x ^ (x >> 30)) * UINT64_C(0xbf58476d1ce4e5b9);
  x = (x ^ (x >> 27)) * UINT64_C(0x94d049bb133111eb);
  return (size_t)(x ^ (x >> 31));
}
static bool q_reserve_stack_slots(QScanner *scanner) {
  if (scanner->stack_count < scanner->slot_capacity / 2) return true;
  size_t capacity = scanner->slot_capacity ? scanner->slot_capacity * 2 : 16;
  if (capacity < scanner->slot_capacity || capacity > SIZE_MAX / sizeof(uint32_t)) return false;
  uint32_t *slots = calloc(capacity, sizeof(uint32_t));
  if (!slots) return false;
  for (size_t i = 0; i < scanner->stack_count; i++) {
    QDefinitionStack s = scanner->stacks[i];
    size_t at = q_stack_hash(s.parent, s.start, s.kind) & (capacity - 1);
    while (slots[at]) { Q_INDEX_STEP(); at = (at + 1) & (capacity - 1); }
    slots[at] = (uint32_t)i + 1;
  }
  free(scanner->stack_slots); scanner->stack_slots = slots;
  scanner->slot_capacity = capacity;
  return true;
}
static bool q_scan_document(QScanner *scanner, TSLexer *lexer) {
  char *source = NULL;
  size_t length = 0, capacity = 0;
  uint64_t fingerprint = UINT64_C(14695981039346656037);
  // The zero-width initial token looks through EOF. An edit anywhere in the
  // file invalidates it; the fingerprint also prevents reuse of scanner states
  // from a different document. Boundary decisions therefore never use old text.
  lexer->mark_end(lexer);
  while (!lexer->eof(lexer)) {
    char bytes[4]; size_t width = q_utf8(lexer->lookahead, bytes);
    if (!q_grow((void **)&source, &capacity, length + width, 1)) { free(source); return false; }
    memcpy(source + length, bytes, width); length += width;
    for (size_t i = 0; i < width; i++) fingerprint = (fingerprint ^ (unsigned char)bytes[i]) * UINT64_C(1099511628211);
    lexer->advance(lexer, false);
  }
  if (scanner->length != length || scanner->fingerprint != fingerprint || (length && memcmp(scanner->source, source, length))) {
    free(scanner->source); scanner->source = source; scanner->length = length;
    scanner->fingerprint = fingerprint;
    scanner->recognized = q_parse_document_cached(&scanner->document, source, length);
    scanner->end_cached = false;
    scanner->begin_cached = false;
    scanner->stack_count = 0;
    if (scanner->stack_slots) memset(scanner->stack_slots, 0, scanner->slot_capacity * sizeof(uint32_t));
    free(scanner->boundary_index); scanner->boundary_index = NULL;
    free(scanner->suffixes); scanner->suffixes = NULL;
    free(scanner->suffix_ranks); scanner->suffix_ranks = NULL;
    free(scanner->positions); scanner->positions = NULL; scanner->position_count = 0;
  } else free(source);
  // A failed allocation must be retried, never leave an index for old text.
  if (!scanner->suffixes && !q_build_suffixes(scanner)) return false;
  if (scanner->document.boundary_count && !scanner->positions && !q_build_positions(scanner)) return false;
  if (scanner->document.boundary_count && !scanner->boundary_index && !q_build_boundary_index(scanner)) return false;
  scanner->stack = 0;
  scanner->position = 0;
  scanner->recovering = false;
  return true;
}
static uint32_t q_token_position(QScanner *scanner, TSLexer *lexer, unsigned prefix) {
  if (!prefix) lexer->mark_end(lexer);
  if (!scanner->suffixes) return UINT32_MAX;
  size_t start = 0, end = scanner->length;
  if (scanner->recognized && !scanner->recovering && scanner->position < scanner->document.token_count) {
    // No successful parse can cross the next reference marker without emitting
    // it. Serialized lower bounds follow GLR backtracking; recovery uses the
    // unrestricted index because skipped tokens break that guarantee.
    uint32_t first = scanner->position;
    if (scanner->stack && scanner->stacks[scanner->stack - 1].start == first &&
        first + 1 < scanner->document.token_count) first++;
    start = scanner->document.tokens[first].start;
    size_t low = 0, high = scanner->position_count;
    while (low < high) {
      size_t mid = low + (high - low) / 2;
      if (scanner->positions[mid] <= scanner->position) low = mid + 1;
      else high = mid;
    }
    if (low < scanner->position_count) {
      uint32_t next = scanner->positions[low];
      if (next < scanner->document.token_count) end = scanner->document.tokens[next].start;
    }
    if (!scanner->stack && q_has_boundary(scanner, scanner->position, 3, 0, false)) end = start;
    if (prefix <= scanner->length - end) end += prefix;
  }
  uint32_t at = q_locate_source(scanner, lexer, start, end);
  if (at == UINT32_MAX || at < prefix) return UINT32_MAX;
  size_t offset = at - prefix;
  QDocument *doc = &scanner->document;
  // Find the first token whose end follows this byte, including skipped text.
  size_t low = 0, high = doc->token_count;
  while (low < high) {
    size_t mid = low + (high - low) / 2;
    if (doc->tokens[mid].end <= offset && doc->tokens[mid].kind != -1) low = mid + 1;
    else high = mid;
  }
  return low < doc->token_count ? (uint32_t)low : UINT32_MAX;
}
static bool q_begin_definition(QScanner *scanner, uint32_t start, uint8_t kind) {
  if (scanner->recognized && !q_has_boundary(scanner, start, kind, 0, false)) return false;
  size_t at = scanner->slot_capacity ? q_stack_hash(scanner->stack, start, kind) & (scanner->slot_capacity - 1) : 0;
  while (scanner->slot_capacity && scanner->stack_slots[at]) {
    Q_INDEX_STEP();
    uint32_t id = scanner->stack_slots[at];
    QDefinitionStack s = scanner->stacks[id - 1];
    if (s.parent == scanner->stack && s.start == start && s.kind == kind) {
      scanner->stack = id; scanner->position = start; return true;
    }
    at = (at + 1) & (scanner->slot_capacity - 1);
  }
  if (scanner->stack_count >= UINT32_MAX || !q_reserve_stack_slots(scanner)) return false;
  at = q_stack_hash(scanner->stack, start, kind) & (scanner->slot_capacity - 1);
  while (scanner->stack_slots[at]) { Q_INDEX_STEP(); at = (at + 1) & (scanner->slot_capacity - 1); }
  if (!q_grow((void **)&scanner->stacks, &scanner->stack_capacity, scanner->stack_count + 1, sizeof(QDefinitionStack))) return false;
  scanner->stacks[scanner->stack_count++] = (QDefinitionStack){scanner->stack, start, kind};
  scanner->stack = (uint32_t)scanner->stack_count;
  scanner->stack_slots[at] = scanner->stack;
  scanner->position = start;
  return true;
}
static bool q_end_definition(QScanner *scanner, uint32_t end, uint8_t kind) {
  if (!scanner->recognized) {
    while (scanner->stack && scanner->stack <= scanner->stack_count) {
      QDefinitionStack top = scanner->stacks[scanner->stack - 1];
      scanner->stack = top.parent;
      if (top.kind == kind) break;
    }
    return true;
  }
  if (!scanner->stack || scanner->stack > scanner->stack_count) return !scanner->recognized;
  QDefinitionStack top = scanner->stacks[scanner->stack - 1];
  if (top.kind != kind) return false;
  if (!q_has_boundary(scanner, top.start, kind, end, true)) return false;
  scanner->stack = top.parent;
  scanner->position = end;
  return true;
}
#endif
