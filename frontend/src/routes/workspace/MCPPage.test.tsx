import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => ({
    externalUrl: "https://bb.example.com",
    needConfigureExternalUrl: false,
  }),
}));

vi.mock("@/components/ExternalUrlAlert", () => ({ ExternalUrlAlert: () => null }));
vi.mock("./mcp/MCPAccessPolicySection", () => ({
  MCPAccessPolicySection: () => null,
}));
vi.mock("@/utils", () => ({ isDev: () => false, cn: (...a: unknown[]) => a.join(" ") }));

const { MCPPage } = await import("./MCPPage");

const render = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => root.render(element));
  return {
    container,
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

// Base UI renders only the active panel, so each tab has to be visited.
const allTabValues = (container: HTMLElement) => {
  const triggers = [...container.querySelectorAll('[role="tab"]')];
  const values: string[] = [];
  for (const trigger of triggers) {
    act(() => (trigger as HTMLElement).click());
    for (const input of container.querySelectorAll("input[readonly]")) {
      values.push((input as HTMLInputElement).value);
    }
    for (const area of container.querySelectorAll("textarea[readonly]")) {
      values.push((area as HTMLTextAreaElement).value);
    }
  }
  return values;
};

describe("MCPPage connect panel", () => {
  // Codex, #21237: three of the six commands were invalid. Each was checked
  // against its own source — the codex CLI's own AddArgs (name is positional,
  // --url is the flag, there is no --transport), the Copilot CLI docs
  // (`copilot mcp add --transport http NAME URL`, and the binary is `copilot`,
  // not `gh copilot`), and the Gemini CLI docs. These assertions are the guard
  // against the wrong forms coming back; the syntax itself can only be
  // re-checked against those sources.
  test("no command uses a flag its CLI does not have", () => {
    const { container, unmount } = render(<MCPPage />);
    const values = allTabValues(container);
    expect(values.length).toBeGreaterThan(6);
    for (const value of values) {
      expect(value).not.toContain("--name ");
      expect(value).not.toMatch(/^gh /);
      // codex takes --url; the others take the URL positionally.
      if (value.startsWith("codex ")) {
        expect(value).toContain("--url ");
      } else if (value.includes(" mcp add")) {
        expect(value).not.toContain("--url ");
      }
    }
    unmount();
  });

  // Codex, #21237: claude_desktop_config.json reaches LOCAL servers only, so
  // the tab handed users a config that cannot connect to a remote Bytebase.
  // A remote server is added as a custom connector on claude.ai instead, so
  // what this tab owes the user is the URL.
  test("the Claude Desktop tab offers the endpoint URL, not a config object", () => {
    const { container, unmount } = render(<MCPPage />);
    const desktop = [...container.querySelectorAll('[role="tab"]')].find((t) =>
      t.textContent?.includes("Claude Desktop")
    );
    act(() => (desktop as HTMLElement).click());

    const shown = [
      ...container.querySelectorAll("input[readonly], textarea[readonly]"),
    ].map((n) => (n as HTMLInputElement | HTMLTextAreaElement).value);

    expect(shown).toContain("https://bb.example.com/mcp");
    // The JSON tab still offers the config object; this one must not.
    for (const value of shown) {
      expect(value).not.toContain("mcpServers");
    }
    unmount();
  });
});
