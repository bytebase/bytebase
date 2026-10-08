// A bounded Myers line diff whose hunk output is pinned by tests. Placement
// depends on exactly which lines a diff keeps, so this stays our own
// implementation rather than a library whose alignment may change on upgrade.

// `sLen` saved lines starting at saved line `sStart` (one-based) are replaced
// by `cLen` current lines. Hunks hold edits only, no context. For a pure
// insertion `sLen` is 0 and `sStart` is the saved line the new lines go in
// front of, in `1..N+1`.
export interface Hunk {
  readonly sStart: number;
  readonly sLen: number;
  readonly cLen: number;
}

export type LineDiffResult =
  | { readonly complete: true; readonly hunks: Hunk[]; readonly work: number }
  | { readonly complete: false; readonly work: number };

type Op = 0 | 1 | 2; // equal, delete, insert
const EQUAL: Op = 0;
const DELETE: Op = 1;
const INSERT: Op = 2;

// Classic forward Myers (An O(ND) Difference Algorithm, 1986) with a trace of
// each round's frontier for backtracking. Tie-breaking is the paper's: when
// both neighbors reach as far, extend from the k+1 diagonal, so an insertion
// is preferred over a deletion at equal cost, and a deletion is chosen only
// when it reaches strictly further. `work` counts frontier steps and token
// comparisons; when it would exceed `budget` the diff stops and reports
// `complete: false` with no hunks.
export function diffLines(
  saved: readonly string[],
  current: readonly string[],
  budget: number
): LineDiffResult {
  const n = saved.length;
  const m = current.length;
  const max = n + m;
  // Diagonal k maps to index k + offset; the extra slot on each side lets the
  // k == -d / k == d reads stay in range without branching.
  const offset = max + 1;
  const v = new Int32Array(2 * max + 3);
  const trace: Int32Array[] = [];
  let work = 0;

  let found = false;
  for (let d = 0; d <= max && !found; d++) {
    // Snapshot the diagonals round d reads from: [-d-1, d+1].
    trace.push(v.slice(offset - d - 1, offset + d + 2));
    for (let k = -d; k <= d; k += 2) {
      if (++work > budget) return { complete: false, work };
      let x: number;
      if (k === -d || (k !== d && v[k - 1 + offset] < v[k + 1 + offset])) {
        x = v[k + 1 + offset];
      } else {
        x = v[k - 1 + offset] + 1;
      }
      let y = x - k;
      while (x < n && y < m) {
        if (++work > budget) return { complete: false, work };
        if (saved[x] !== current[y]) break;
        x++;
        y++;
      }
      v[k + offset] = x;
      if (x >= n && y >= m) {
        found = true;
        break;
      }
    }
  }

  return { complete: true, hunks: backtrack(trace, n, m), work };
}

function backtrack(trace: Int32Array[], n: number, m: number): Hunk[] {
  const script: Op[] = [];
  let x = n;
  let y = m;
  for (let d = trace.length - 1; d > 0; d--) {
    const prev = trace[d];
    const at = (k: number) => prev[k + d + 1];
    const k = x - y;
    const prevK =
      k === -d || (k !== d && at(k - 1) < at(k + 1)) ? k + 1 : k - 1;
    const prevX = at(prevK);
    const prevY = prevX - prevK;
    while (x > prevX && y > prevY) {
      script.push(EQUAL);
      x--;
      y--;
    }
    script.push(x === prevX ? INSERT : DELETE);
    x = prevX;
    y = prevY;
  }
  // What remains is the leading snake on diagonal 0.
  while (x > 0) {
    script.push(EQUAL);
    x--;
  }
  script.reverse();

  const hunks: Hunk[] = [];
  let open: { sStart: number; sLen: number; cLen: number } | undefined;
  let sPos = 0; // saved lines consumed so far
  for (const op of script) {
    if (op === EQUAL) {
      if (open) {
        hunks.push(open);
        open = undefined;
      }
      sPos++;
      continue;
    }
    open ??= { sStart: sPos + 1, sLen: 0, cLen: 0 };
    if (op === DELETE) {
      open.sLen++;
      sPos++;
    } else {
      open.cLen++;
    }
  }
  if (open) hunks.push(open);
  return hunks;
}
