import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import { MCP_CAPABILITY_CHOICES, MCP_MODE_PRESENTATION } from "./mcpPolicy";
import { MCPModeBadge } from "./MCPModeBadge";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

describe("MCPModeBadge", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  // The identity an admin picked in the selector is the identity shown in force
  // and on the consent screen. Without this the glyph can be deleted or wired to
  // the wrong mode with every gate still green.
  test("every mode carries its own glyph beside its name", () => {
    const glyphs = new Set<string>();
    for (const mode of MCP_CAPABILITY_CHOICES) {
      const { container, unmount } = renderIntoContainer(
        <MCPModeBadge mode={mode} />
      );
      const icon = container.querySelector("svg");
      expect(icon).not.toBeNull();
      glyphs.add(icon?.outerHTML ?? "");
      expect(container.textContent).toContain(
        `settings.mcp.policy.mode.${MCP_MODE_PRESENTATION[mode].key}.title`
      );
      unmount();
      document.body.innerHTML = "";
    }
    // Three modes, three distinct glyphs: a table wired to one icon for all
    // would still satisfy the per-mode assertions above.
    expect(glyphs.size).toBe(MCP_CAPABILITY_CHOICES.length);
  });

  test("without a description the badge is just the mode's name", () => {
    const { container, unmount } = renderIntoContainer(
      <MCPModeBadge mode={MCPSetting_Capability.READ_ONLY} />
    );
    expect(container.querySelector("span.sr-only")).toBeNull();
    expect(container.querySelector("[aria-hidden='true']")?.tagName).toBe(
      "svg"
    );
    unmount();
  });

  test("a described badge reads the sentence once, not the name twice", () => {
    const { container, unmount } = renderIntoContainer(
      <MCPModeBadge
        mode={MCPSetting_Capability.READ_ONLY}
        describedAs="Current policy: Read-only"
      />
    );
    expect(container.querySelector("span.sr-only")?.textContent).toBe(
      "Current policy: Read-only"
    );
    const visible = [...container.querySelectorAll("span")].find(
      (node) =>
        node.getAttribute("aria-hidden") === "true" && node.textContent !== ""
    );
    expect(visible?.textContent).toBe(
      "settings.mcp.policy.mode.read-only.title"
    );
    expect(container.querySelector("[aria-label]")).toBeNull();
    unmount();
  });
});
