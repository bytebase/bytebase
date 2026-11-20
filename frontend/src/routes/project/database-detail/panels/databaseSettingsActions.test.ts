import { toJson } from "@bufbuild/protobuf";
import { expect, test } from "vitest";
import { UpdateDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import {
  buildDatabaseEnvironmentUpdate,
  buildDatabaseLabelsUpdate,
  labelsChanged,
  labelsFingerprint,
  validRows,
} from "./databaseSettingsActions";

const name = "instances/prod/databases/app";
test("patches serialize only the selected mask and preserve an explicit clear", () => {
  const maps: Record<string, string>[] = [{}, { region: "west" }];
  for (const labels of maps) {
    const req = buildDatabaseLabelsUpdate(name, labels);
    expect(req.updateMask?.paths).toEqual(["labels"]);
    expect(toJson(UpdateDatabaseRequestSchema, req)).toEqual({
      database: Object.keys(labels).length ? { name, labels } : { name },
      updateMask: "labels",
    });
  }
  for (const environment of ["", "environments/prod"]) {
    expect(
      toJson(
        UpdateDatabaseRequestSchema,
        buildDatabaseEnvironmentUpdate(name, environment)
      )
    ).toEqual({
      database: { name, environment },
      updateMask: "environment",
    });
  }
});

test("saved labels compare by map semantics, without hiding invalid duplicate or blank rows", () => {
  const labels = { b: "2", a: "1" };
  expect(labelsFingerprint(labels)).toBe(labelsFingerprint({ a: "1", b: "2" }));
  const rows = Object.entries(labels).map(([key, value]) => ({ key, value }));
  expect(labelsChanged(labels, rows)).toBe(false);
  for (const invalid of [
    { key: "b", value: "2" },
    { key: "", value: "" },
    { key: "new", value: "x".repeat(64) },
  ]) {
    expect(labelsChanged(labels, [...rows, invalid])).toBe(true);
    expect(validRows([...rows, invalid])).toBe(false);
  }
});
