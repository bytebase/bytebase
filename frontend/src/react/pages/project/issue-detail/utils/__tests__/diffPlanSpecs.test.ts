import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  Plan_ChangeDatabaseConfigSchema,
  type Plan_Spec,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { diffPlanSpecs, type SpecDiffEntry } from "../diffPlanSpecs";

function cdcSpec(opts: {
  id: string;
  sheet?: string;
  targets?: string[];
  enablePriorBackup?: boolean;
}): Plan_Spec {
  return create(Plan_SpecSchema, {
    id: opts.id,
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, {
        sheet: opts.sheet ?? "",
        targets: opts.targets ?? [],
        enablePriorBackup: opts.enablePriorBackup ?? false,
      }),
    },
  });
}

describe("diffPlanSpecs", () => {
  it("empty/empty -> []", () => {
    expect(diffPlanSpecs([], [])).toEqual([]);
  });

  it("one added", () => {
    const entries = diffPlanSpecs([], [cdcSpec({ id: "a", sheet: "s1" })]);
    expect(entries).toHaveLength(1);
    expect(entries[0].kind).toBe("added");
    expect(
      (entries[0] as Extract<SpecDiffEntry, { kind: "added" }>).spec.id
    ).toBe("a");
  });

  it("one removed", () => {
    const entries = diffPlanSpecs([cdcSpec({ id: "a", sheet: "s1" })], []);
    expect(entries).toHaveLength(1);
    expect(entries[0].kind).toBe("removed");
    expect(
      (entries[0] as Extract<SpecDiffEntry, { kind: "removed" }>).spec.id
    ).toBe("a");
  });

  it("sheet changed -> updated with sheetChanged", () => {
    const entries = diffPlanSpecs(
      [cdcSpec({ id: "a", sheet: "projects/p/sheets/s1" })],
      [cdcSpec({ id: "a", sheet: "projects/p/sheets/s2" })]
    );
    expect(entries).toHaveLength(1);
    expect(entries[0].kind).toBe("updated");
    const u = entries[0] as Extract<SpecDiffEntry, { kind: "updated" }>;
    expect(u.sheetChanged).toBe(true);
    expect(u.targetsChanged).toBe(false);
    expect(u.priorBackupChanged).toBe(false);
    expect(u.otherChanged).toBe(false);
  });

  it("targets changed -> updated with targetsChanged", () => {
    const entries = diffPlanSpecs(
      [cdcSpec({ id: "a", targets: ["db1"] })],
      [cdcSpec({ id: "a", targets: ["db1", "db2"] })]
    );
    expect(entries).toHaveLength(1);
    const u = entries[0] as Extract<SpecDiffEntry, { kind: "updated" }>;
    expect(u.sheetChanged).toBe(false);
    expect(u.targetsChanged).toBe(true);
    expect(u.priorBackupChanged).toBe(false);
  });

  it("prior_backup flipped -> updated with priorBackupChanged", () => {
    const entries = diffPlanSpecs(
      [cdcSpec({ id: "a", enablePriorBackup: false })],
      [cdcSpec({ id: "a", enablePriorBackup: true })]
    );
    expect(entries).toHaveLength(1);
    const u = entries[0] as Extract<SpecDiffEntry, { kind: "updated" }>;
    expect(u.priorBackupChanged).toBe(true);
  });

  it("all three changed -> one updated entry with all flags set", () => {
    const entries = diffPlanSpecs(
      [
        cdcSpec({
          id: "a",
          sheet: "projects/p/sheets/s1",
          targets: ["db1"],
          enablePriorBackup: false,
        }),
      ],
      [
        cdcSpec({
          id: "a",
          sheet: "projects/p/sheets/s2",
          targets: ["db1", "db2"],
          enablePriorBackup: true,
        }),
      ]
    );
    expect(entries).toHaveLength(1);
    const u = entries[0] as Extract<SpecDiffEntry, { kind: "updated" }>;
    expect(u.sheetChanged).toBe(true);
    expect(u.targetsChanged).toBe(true);
    expect(u.priorBackupChanged).toBe(true);
  });

  it("reorder-only -> []", () => {
    const a = cdcSpec({ id: "a", sheet: "projects/p/sheets/s1" });
    const b = cdcSpec({ id: "b", sheet: "projects/p/sheets/s2" });
    expect(diffPlanSpecs([a, b], [b, a])).toEqual([]);
  });

  it("mixed add + remove + update -> three entries, ADDED, REMOVED, UPDATED order", () => {
    const entries = diffPlanSpecs(
      [
        cdcSpec({ id: "old", sheet: "projects/p/sheets/x" }),
        cdcSpec({ id: "keep", sheet: "projects/p/sheets/v1" }),
      ],
      [
        cdcSpec({ id: "keep", sheet: "projects/p/sheets/v2" }),
        cdcSpec({ id: "new", sheet: "projects/p/sheets/y" }),
      ]
    );
    expect(entries.map((e) => e.kind)).toEqual(["added", "removed", "updated"]);
  });

  it("unchanged content -> []", () => {
    const a = cdcSpec({
      id: "a",
      sheet: "projects/p/sheets/s1",
      targets: ["db1"],
      enablePriorBackup: false,
    });
    const b = cdcSpec({
      id: "a",
      sheet: "projects/p/sheets/s1",
      targets: ["db1"],
      enablePriorBackup: false,
    });
    expect(diffPlanSpecs([a], [b])).toEqual([]);
  });
});
