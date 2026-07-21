import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const sectionDir = dirname(fileURLToPath(import.meta.url));

describe("UsersPage drawer layout", () => {
  test("keeps the password hint aligned in the compact drawer field", () => {
    const source = readFileSync(join(sectionDir, "UsersPage.tsx"), "utf8");

    const passwordTitleIndex = source.indexOf('id="user-form-password-title"');
    const passwordHintIndex = source.indexOf(
      't("settings.profile.password-hint")'
    );
    const hintRowIndex = source.indexOf("<span\n                className");
    const hintRowClassIndex = source.indexOf(
      "flex items-center gap-x-1 text-sm text-control-placeholder",
      hintRowIndex
    );
    const hintErrorIndex = source.indexOf('passwordHint ? "text-error" : ""');
    const passwordInputIndex = source.indexOf(
      'className="relative mt-1 flex w-full items-center"'
    );

    expect(passwordTitleIndex).toBeGreaterThan(-1);
    expect(passwordHintIndex).toBeGreaterThan(-1);
    expect(passwordTitleIndex).toBeLessThan(passwordHintIndex);
    expect(hintRowIndex).toBeGreaterThan(-1);
    expect(hintRowClassIndex).toBeGreaterThan(hintRowIndex);
    expect(hintRowClassIndex).toBeLessThan(passwordHintIndex);
    expect(hintErrorIndex).toBeGreaterThan(hintRowClassIndex);
    expect(hintErrorIndex).toBeLessThan(passwordHintIndex);
    expect(passwordInputIndex).toBeGreaterThan(passwordHintIndex);
    expect(source).not.toContain(
      'description={\n                <>\n                  {t("settings.profile.password-hint")}'
    );
    expect(source).toContain("FormTitle");
    expect(source).not.toContain("FormLabel");
    expect(source).not.toContain("textlabel");
    expect(source).not.toContain("textinfolabel");
  });
});
