import { render, screen } from "@testing-library/react";
import { expect, test } from "vitest";
import { Switch } from "./switch";

test("uses disabled state selectors for a locked checked switch", () => {
  render(
    <Switch
      checked
      disabled
      aria-label="Sync all databases"
      onCheckedChange={() => undefined}
    />
  );

  const control = screen.getByRole("switch", { name: "Sync all databases" });
  expect(control).toHaveAttribute("data-disabled");
  expect(control).toHaveClass("data-disabled:cursor-not-allowed");
  expect(control).toHaveClass("data-disabled:data-[checked]:bg-accent/50");
});
