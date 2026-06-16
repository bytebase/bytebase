import { render, screen } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import { PRESET_BY_ID } from "./presets";
import { SQLEditorThemeScope, useSQLEditorTheme } from "./SQLEditorThemeScope";

function Probe() {
  const theme = useSQLEditorTheme();
  return <span data-testid="id">{theme.id}</span>;
}

describe("SQLEditorThemeScope", () => {
  test("writes the theme tokens as inline CSS vars on its container", () => {
    const { container } = render(
      <SQLEditorThemeScope theme={PRESET_BY_ID.dark}>
        <div>child</div>
      </SQLEditorThemeScope>
    );
    const el = container.firstChild as HTMLElement;
    expect(el.style.getPropertyValue("--color-background")).toBe(
      PRESET_BY_ID.dark.tokens["--color-background"]
    );
  });

  test("provides the theme via context", () => {
    render(
      <SQLEditorThemeScope theme={PRESET_BY_ID.dark}>
        <Probe />
      </SQLEditorThemeScope>
    );
    expect(screen.getByTestId("id").textContent).toBe("dark");
  });

  test("nested scope overrides the parent for context consumers", () => {
    render(
      <SQLEditorThemeScope theme={PRESET_BY_ID.light}>
        <SQLEditorThemeScope theme={PRESET_BY_ID.dark}>
          <Probe />
        </SQLEditorThemeScope>
      </SQLEditorThemeScope>
    );
    expect(screen.getByTestId("id").textContent).toBe("dark");
  });

  test("default context value is the light theme", () => {
    render(<Probe />);
    expect(screen.getByTestId("id").textContent).toBe("light");
  });
});
