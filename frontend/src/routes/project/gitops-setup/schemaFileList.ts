// How the schema files preview lists the export: one row per folder, with its
// file count. A schema can have thousands of files, so a folder shows at most
// SCHEMA_FILE_CAP of them until the owner asks for all.
export const SCHEMA_FILE_CAP = 50;
// A schema with this many files or fewer starts with its folders open.
export const SCHEMA_OPEN_BY_DEFAULT = 12;
// Above this many files the preview has a filter.
export const SCHEMA_FILTER_FROM = 20;

export type SchemaListEntry =
  | { kind: "folder"; dir: string; paths: string[] }
  | { kind: "file"; path: string };

const dirOf = (path: string) => path.slice(0, path.lastIndexOf("/") + 1);

// One entry per folder, in path order. A folder that holds one file is listed
// as that file, so showing it takes no extra click.
export function groupSchemaFiles(paths: string[]): SchemaListEntry[] {
  const byDir = new Map<string, string[]>();
  for (const path of [...paths].sort((a, b) => a.localeCompare(b))) {
    const dir = dirOf(path);
    const list = byDir.get(dir) ?? [];
    list.push(path);
    byDir.set(dir, list);
  }
  return [...byDir.entries()].map(([dir, list]) =>
    list.length === 1
      ? { kind: "file", path: list[0] }
      : { kind: "folder", dir, paths: list }
  );
}

// The paths that contain the query, ignoring case.
export function filterSchemaFiles(paths: string[], query: string): string[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return paths;
  return paths.filter((path) => path.toLowerCase().includes(needle));
}
