import { describe, expect, test } from "vitest";
import { groupReleaseChanges } from "./releaseChanges";

const filesOf = (versions: string[]) =>
  versions.map((version) => ({
    version,
    path: `migrations/${version}.sql`,
    sheet: `projects/p1/sheets/${version}`,
  }));
const versionsOf = (groups: ReturnType<typeof groupReleaseChanges>) =>
  groups.flatMap((group) => group.files.map((file) => file.version));

describe("groupReleaseChanges", () => {
  test("lists the stored files newest first", () => {
    expect(versionsOf(groupReleaseChanges(filesOf(["1", "2", "10"])))).toEqual([
      "10",
      "2",
      "1",
    ]);
  });

  test("timestamps group by day for the newest day and by month for the rest", () => {
    const groups = groupReleaseChanges(
      filesOf(["202609011200", "202609151200", "202610050900", "202610051500"])
    );
    expect(groups.map((group) => group.key)).toEqual(["2026-10-05", "2026-09"]);
  });

  // Versions compare as numbers, so the 8-digit day runs before the 12-digit
  // September times.
  test("timestamps whose execution order is not date order are one flat list", () => {
    const groups = groupReleaseChanges(
      filesOf(["20261005", "202609011200", "202609301200"])
    );
    expect(groups.map((group) => group.key)).toEqual([""]);
    expect(versionsOf(groups)).toEqual([
      "202609301200",
      "202609011200",
      "20261005",
    ]);
  });

  test("two versions that parse equal keep the stored order, reversed", () => {
    expect(
      versionsOf(groupReleaseChanges(filesOf(["1", "01", "2", "2.0", "2.00"])))
    ).toEqual(["2.00", "2.0", "2", "01", "1"]);
  });
});
