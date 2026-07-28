import { describe, expect, test } from "vitest";
import { buttonVariants } from "./button";

describe("buttonVariants", () => {
  test("keeps sizing overridable for class-helper consumers", () => {
    const className = buttonVariants({ className: "size-8 p-0" });

    expect(className).toContain("size-8");
    expect(className).toContain("p-0");
    expect(className).not.toContain("h-9");
    expect(className).not.toContain("px-3");
  });
});
