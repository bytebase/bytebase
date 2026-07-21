// Pure worksheet-folder path helpers, extracted from context.ts so the
// hierarchy logic is directly unit-testable.

/**
 * Whether `path` is a subfolder of `parent`. With `dig` set, any depth of
 * descendant matches; otherwise only direct children.
 *
 * Folder paths come back from localStorage with only an "array of strings"
 * shape check, so malformed entries (empty strings, paths missing the
 * `/{view}` root prefix) must be treated as inert — never as subfolders.
 */
export const isSubFolder = ({
  parent,
  path,
  dig,
}: {
  parent: string;
  path: string;
  dig: boolean;
}): boolean => {
  const parentPrefix = `${parent}/`;
  if (path === parentPrefix || !path.startsWith(parentPrefix)) {
    return false;
  }
  if (dig) {
    return true;
  }
  return !path.slice(parentPrefix.length).includes("/");
};
