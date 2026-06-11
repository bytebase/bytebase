import { describe, expect, test } from "vitest";
import { isSubFolder } from "./folder";

describe("isSubFolder", () => {
  test("direct child matches with dig=false", () => {
    expect(isSubFolder({ parent: "/my", path: "/my/a", dig: false })).toBe(
      true
    );
  });

  test("deeper descendant does not match with dig=false", () => {
    expect(isSubFolder({ parent: "/my", path: "/my/a/b", dig: false })).toBe(
      false
    );
  });

  test("any descendant matches with dig=true", () => {
    expect(isSubFolder({ parent: "/my", path: "/my/a", dig: true })).toBe(true);
    expect(isSubFolder({ parent: "/my", path: "/my/a/b", dig: true })).toBe(
      true
    );
  });

  test("a path outside the parent never matches", () => {
    expect(isSubFolder({ parent: "/my", path: "/shared/a", dig: false })).toBe(
      false
    );
    expect(isSubFolder({ parent: "/my", path: "/shared/a", dig: true })).toBe(
      false
    );
  });

  test("the parent itself (with or without trailing slash) is not its own subfolder", () => {
    expect(isSubFolder({ parent: "/my", path: "/my", dig: false })).toBe(false);
    expect(isSubFolder({ parent: "/my", path: "/my/", dig: false })).toBe(
      false
    );
  });

  // Malformed entries can reach us from persisted localStorage (the read
  // path only validates "array of strings"). They must be inert — the old
  // implementation treated them as subfolders of EVERY parent, including
  // themselves, sending the sheet-tree builder into infinite recursion.
  test("malformed persisted entries are never subfolders of anything", () => {
    for (const dig of [false, true]) {
      expect(isSubFolder({ parent: "/my", path: "", dig })).toBe(false);
      expect(isSubFolder({ parent: "/my", path: "foo", dig })).toBe(false);
      expect(isSubFolder({ parent: "", path: "", dig })).toBe(false);
      expect(isSubFolder({ parent: "foo", path: "foo", dig })).toBe(false);
    }
  });

  test("a parent that appears as an inner segment elsewhere does not match", () => {
    // The old `.replace(parentPrefix, "")` stripped the FIRST occurrence
    // anywhere in the string, not just the prefix.
    expect(
      isSubFolder({ parent: "/my", path: "/shared/my/a", dig: false })
    ).toBe(false);
  });
});
