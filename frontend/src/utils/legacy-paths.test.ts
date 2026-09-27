import { describe, expect, test } from "vitest";
import { rewriteLegacyPath } from "./legacy-paths";

const setPath = (path: string) => {
  window.history.replaceState(null, "", path);
};

describe("rewriteLegacyPath", () => {
  test("rewrites a legacy sheet deep link, keeping query and hash", () => {
    setPath("/sql-editor/projects/p1/sheets/abc-123?schema=public#frag");
    rewriteLegacyPath();
    expect(
      window.location.pathname + window.location.search + window.location.hash
    ).toBe("/sql-editor/projects/p1/savedQueries/abc-123?schema=public#frag");
  });

  test("tolerates a trailing slash", () => {
    setPath("/sql-editor/projects/p1/sheets/abc/");
    rewriteLegacyPath();
    expect(window.location.pathname).toBe(
      "/sql-editor/projects/p1/savedQueries/abc"
    );
  });

  test("leaves current and unrelated paths alone", () => {
    for (const path of [
      "/sql-editor/projects/p1/savedQueries/abc",
      "/sql-editor/projects/p1/instances/i1/databases/db1",
      "/projects/p1/sheets/abc",
      "/sql-editor",
    ]) {
      setPath(path);
      rewriteLegacyPath();
      expect(window.location.pathname).toBe(path);
    }
  });
});
