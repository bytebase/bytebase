import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { useState } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { CollapsibleTilePicker, TileRadioItem } from "./CollapsibleTilePicker";

afterEach(cleanup);

function UncontrolledHarness({
  defaultCollapsed = true,
  disabled = false,
  onValueChange = () => undefined,
}: {
  defaultCollapsed?: boolean;
  disabled?: boolean;
  onValueChange?: (value: string) => void;
}) {
  const [value, setValue] = useState("generic");
  return (
    <CollapsibleTilePicker
      label="Hosted on"
      value={value}
      summary={<span>{value === "neon" ? "Neon" : "Any host or cloud"}</span>}
      defaultCollapsed={defaultCollapsed}
      disabled={disabled}
      onValueChange={(next) => {
        setValue(next);
        onValueChange(next);
      }}
    >
      <TileRadioItem value="generic">Any host or cloud</TileRadioItem>
      <TileRadioItem value="neon">Neon</TileRadioItem>
    </CollapsibleTilePicker>
  );
}

test("uncontrolled picker starts collapsed, expands on click, selects by radio or text, and collapses", () => {
  const onValueChange = vi.fn();
  render(<UncontrolledHarness onValueChange={onValueChange} />);

  const trigger = screen.getByRole("button", { name: /^Hosted on/ });
  expect(trigger).toHaveAttribute("aria-expanded", "false");
  expect(trigger).toHaveTextContent("Any host or cloud");
  expect(
    screen.queryByRole("radiogroup", { name: "Hosted on" })
  ).not.toBeInTheDocument();

  fireEvent.click(trigger);
  expect(trigger).toHaveAttribute("aria-expanded", "true");
  expect(
    screen.getByRole("radiogroup", { name: "Hosted on" })
  ).toBeVisible();

  // Clicking the tile text label selects the radio and collapses the panel.
  fireEvent.click(screen.getByText("Neon"));
  expect(onValueChange).toHaveBeenCalledWith("neon");
  expect(trigger).toHaveAttribute("aria-expanded", "false");
  expect(trigger).toHaveTextContent("Neon");

  // Re-open and click the already-selected radio directly.
  fireEvent.click(trigger);
  expect(trigger).toHaveAttribute("aria-expanded", "true");
  const neonRadio = screen.getByRole("radio", { name: "Neon" });
  expect(neonRadio).toHaveAttribute("aria-checked", "true");
  fireEvent.click(neonRadio);
  expect(trigger).toHaveAttribute("aria-expanded", "false");
  expect(onValueChange).toHaveBeenCalledTimes(1);
});

test("controlled mode delegates collapsed state to the parent", () => {
  function ControlledHarness() {
    const [value, setValue] = useState("mysql");
    const [collapsed, setCollapsed] = useState(false);
    return (
      <CollapsibleTilePicker
        label="Engine"
        value={value}
        summary={<span>{value}</span>}
        collapsed={collapsed}
        onCollapsedChange={setCollapsed}
        onValueChange={setValue}
      >
        <TileRadioItem value="mysql">MySQL</TileRadioItem>
        <TileRadioItem value="postgres">PostgreSQL</TileRadioItem>
      </CollapsibleTilePicker>
    );
  }

  render(<ControlledHarness />);
  const trigger = screen.getByRole("button", { name: /^Engine/ });
  expect(trigger).toHaveAttribute("aria-expanded", "true");

  fireEvent.click(screen.getByRole("radio", { name: "PostgreSQL" }));
  expect(trigger).toHaveAttribute("aria-expanded", "false");
  expect(trigger).toHaveTextContent("postgres");

  fireEvent.click(trigger);
  expect(trigger).toHaveAttribute("aria-expanded", "true");
});

test("pressing Enter or Escape inside the radiogroup collapses and restores focus to trigger", () => {
  render(<UncontrolledHarness defaultCollapsed={false} />);

  const trigger = screen.getByRole("button", { name: /^Hosted on/ });
  const genericRadio = screen.getByRole("radio", {
    name: "Any host or cloud",
  });

  act(() => {
    genericRadio.focus();
  });
  expect(genericRadio).toHaveFocus();
  fireEvent.keyDown(genericRadio, { key: "Enter" });
  expect(trigger).toHaveAttribute("aria-expanded", "false");
  expect(trigger).toHaveFocus();

  fireEvent.click(trigger);
  const neonRadio = screen.getByRole("radio", { name: "Neon" });
  act(() => {
    neonRadio.focus();
  });
  fireEvent.keyDown(neonRadio, { key: "Escape" });
  expect(trigger).toHaveAttribute("aria-expanded", "false");
  expect(trigger).toHaveFocus();
});

test("arrow-key navigation changes selection without collapsing", async () => {
  const onValueChange = vi.fn();
  render(
    <UncontrolledHarness
      defaultCollapsed={false}
      onValueChange={onValueChange}
    />
  );

  const trigger = screen.getByRole("button", { name: /^Hosted on/ });
  const genericRadio = screen.getByRole("radio", {
    name: "Any host or cloud",
  });
  act(() => {
    genericRadio.focus();
  });
  await act(async () => {
    fireEvent.keyDown(genericRadio, { key: "ArrowRight" });
  });
  expect(onValueChange).toHaveBeenCalledWith("neon");
  expect(trigger).toHaveAttribute("aria-expanded", "true");
});

test("respects disabled state on trigger and tiles", () => {
  const onValueChange = vi.fn();
  render(
    <UncontrolledHarness
      defaultCollapsed={false}
      disabled
      onValueChange={onValueChange}
    />
  );

  const trigger = screen.getByRole("button", { name: /^Hosted on/ });
  expect(trigger).toBeDisabled();

  const neonRadio = screen.getByRole("radio", { name: "Neon" });
  expect(neonRadio).toHaveAttribute("aria-disabled", "true");

  fireEvent.click(neonRadio);
  fireEvent.keyDown(neonRadio, { key: "Escape" });
  expect(onValueChange).not.toHaveBeenCalled();
  expect(trigger).toHaveAttribute("aria-expanded", "true");
});
