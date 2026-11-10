import { describe, expect, test } from "vitest";
import { buttonVariants } from "./button";

describe("buttonVariants", () => {
  test("uses an accent ghost appearance without a resting fill or border", () => {
    const classes = buttonVariants({ appearance: "ghost" }).split(" ");

    expect(classes).toContain("bg-transparent");
    expect(classes).toContain("text-accent");
    expect(classes).toContain("hover:bg-accent/10");
    expect(classes).not.toContain("border");
    expect(classes).not.toContain("bg-accent");
    expect(classes).not.toContain("text-accent-text");
    expect(classes).not.toContain("hover:underline");
  });

  test("uses error semantics for destructive ghost buttons", () => {
    const classes = buttonVariants({
      variant: "destructive",
      appearance: "ghost",
    }).split(" ");

    expect(classes).toContain("bg-transparent");
    expect(classes).toContain("text-error");
    expect(classes).toContain("hover:bg-error/10");
    expect(classes).not.toContain("text-accent");
    expect(classes).not.toContain("border");
  });

  test.each([
    ["default", "h-9", "px-3", "text-sm", "gap-1.5"],
    ["xs", "h-6", "px-1.5", "text-xs", "gap-1"],
    ["sm", "h-7", "px-2", "text-xs", "gap-1"],
    ["md", "h-9", "px-3", "text-sm", "gap-1.5"],
    ["lg", "h-10", "px-4", "text-sm", "gap-1.5"],
  ] as const)(
    "preserves the %s size and interaction contracts for ghost buttons",
    (size, height, padding, text, gap) => {
      const classes = buttonVariants({ appearance: "ghost", size }).split(" ");

      expect(classes).toEqual(
        expect.arrayContaining([
          height,
          padding,
          text,
          gap,
          "focus-visible:ring-2",
          "focus-visible:ring-accent",
          "focus-visible:ring-offset-2",
          "disabled:cursor-not-allowed",
          "disabled:opacity-50",
        ])
      );
    }
  );

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
