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

  test("preserves an explicit content-sized body-text control contract", () => {
    const className = buttonVariants({
      size: "md",
      className: "h-auto gap-1 px-1.5 py-0.5 text-sm",
    });

    expect(className).toContain("h-auto");
    expect(className).toContain("gap-1");
    expect(className).toContain("px-1.5");
    expect(className).toContain("py-0.5");
    expect(className).toContain("text-sm");
    expect(className).not.toContain("h-9");
    expect(className).not.toContain("gap-1.5");
    expect(className).not.toContain("px-3");
  });
});
