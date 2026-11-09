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

  test("two versions that parse equal keep the stored order, reversed", () => {
    expect(
      versionsOf(groupReleaseChanges(filesOf(["1", "01", "2", "2.0", "2.00"])))
    ).toEqual(["2.00", "2.0", "2", "01", "1"]);
  });
});
