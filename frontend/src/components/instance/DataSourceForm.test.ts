import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";

const source = () =>
  readFileSync(
    join(process.cwd(), "src/components/instance/DataSourceForm.tsx"),
    "utf-8"
  );

describe("DataSourceForm password sources", () => {
  test("keeps a draft per instance data source and secret provider", () => {
    const form = source();

    expect(form).toContain("sourceDraftStateRef");
    expect(form).toContain("invalidateSourceDrafts");
    expect(form).toContain("dataSourceResetEvent");
    expect(form).toContain("sourceDraftsRef.current.get(dataSource.id)");
    expect(form).toContain("drafts.set(");
    expect(form).toContain("const existingDraft = drafts.get(secretType)");
    expect(form).toContain("ds.externalSecret = existingDraft");
  });

  test("exports Redis Sentinel fields for the connection mode panel", () => {
    expect(source()).toContain("export function RedisSentinelFields");
  });

  test("does not initialize a provider from another provider's identifiers", () => {
    const form = source();

    expect(form).not.toContain("ds.externalSecret?.secretName ??");
    expect(form).not.toContain("ds.externalSecret?.passwordKeyName ??");
    expect(form).toContain('secretName: ""');
    expect(form).toContain('passwordKeyName: ""');
  });

  test("keeps the password source selector beside the Bytebase password input", () => {
    const form = source();
    const sourceIndex = form.indexOf("{passwordSourceControl}");
    const inputIndex = form.indexOf('type="password"', sourceIndex);
    const storedHintIndex = form.indexOf("stored-in-bytebase");

    expect(sourceIndex).toBeGreaterThan(0);
    expect(inputIndex).toBeGreaterThan(sourceIndex);
    expect(storedHintIndex).toBeGreaterThan(sourceIndex);
  });
});
