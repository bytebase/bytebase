import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, test } from "vitest";

const componentDir = dirname(fileURLToPath(import.meta.url));

describe("UsersPage onboarding intro", () => {
  test("binds the shared Create User intro to the existing create button", () => {
    const source = readFileSync(join(componentDir, "UsersPage.tsx"), "utf8");

    expect(source).toContain("useProductIntro({");
    expect(source).toContain("id: CREATE_USER_PRODUCT_INTRO");
    expect(source).toContain("disabled: !canCreateUser");
    expect(source).toContain(
      "data-product-intro-target={CREATE_USER_PRODUCT_INTRO}"
    );
  });
});
