// The Changes list of a release (D1): every file by parsed version, newest
// first, with one grouping level derived from the version's shape.

export const CHANGES_ROW_LIMIT = 12;

export interface ReleaseChangeFile {
  version: string;
  path: string;
  sheet: string;
}

export interface ReleaseChangeGroup {
  // A day ("2026-10-05"), a month ("2026-10"), or "" for the one flat list.
  // A group names what its files share, never a row count.
  key: string;
  files: ReleaseChangeFile[];
}

// A first segment of 8 to 14 digits starting with a plausible year.
const TIMESTAMP_RE = /^((?:19|20)\d{2})(\d{2})(\d{2})\d{0,6}$/;

const timestampOf = (version: string) => {
  const match = TIMESTAMP_RE.exec(version.split(".")[0]);
  if (!match) return undefined;
  const [, year, month, day] = match;
  if (Number(month) < 1 || Number(month) > 12) return undefined;
  if (Number(day) < 1 || Number(day) > 31) return undefined;
  return { day: `${year}-${month}-${day}`, month: `${year}-${month}` };
};

export const groupReleaseChanges = (
  files: ReleaseChangeFile[]
): ReleaseChangeGroup[] => {
  // The store keeps the files in the executor's order, so newest first is
  // that list reversed; a client sort would reorder versions that parse equal.
  const sorted = [...files].reverse();
  if (sorted.length === 0) return [];

  const days = sorted.map((file) => timestampOf(file.version)?.day);
  // Groups follow the executor's order, so they need it to be date order.
  const dateOrdered = days.every(
    (day, i) => day !== undefined && (i === 0 || day <= (days[i - 1] ?? ""))
  );
  if (dateOrdered) {
    const newestDay = days[0];
    const groups: ReleaseChangeGroup[] = [];
    for (const file of sorted) {
      const stamp = timestampOf(file.version)!;
      const key = stamp.day === newestDay ? stamp.day : stamp.month;
      const last = groups.at(-1);
      if (last?.key === key) last.files.push(file);
      else groups.push({ key, files: [file] });
    }
    return groups;
  }

  return [{ key: "", files: sorted }];
};
