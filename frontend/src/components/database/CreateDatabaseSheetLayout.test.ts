import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const componentDir = dirname(fileURLToPath(import.meta.url));

describe("CreateDatabaseSheet layout", () => {
  test("uses shared form primitives in the create database drawer", () => {
    const source = readFileSync(
      join(componentDir, "CreateDatabaseSheet.tsx"),
      "utf8"
    );

    expect(source).toContain("<FormFieldGroup>");
    expect(source).toContain("<FormField");
    expect(source).toContain("FormTitle");
    expect(source).not.toContain("FormLabel");
    expect(source).toContain('id="create-database-name"');
    expect(source).toContain('id="create-database-title"');
    expect(source).toContain('t("create-db.issue-title")');
    expect(source).not.toContain('t("common.title")');
    expect(source).not.toContain('className="block text-sm font-medium mb-1"');
    expect(source).not.toContain('className="flex flex-col gap-y-2"');
  });
});
