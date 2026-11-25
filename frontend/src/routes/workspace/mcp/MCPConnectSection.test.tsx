import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
  Trans: ({ i18nKey }: { i18nKey: string }) => i18nKey,
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

const serverState = vi.hoisted(() => ({
  externalUrl: "https://bb.example.com",
  needConfigureExternalUrl: false,
}));

vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => serverState,
}));

const { MCP_SETUP_GUIDE_URL, MCPConnectSection } = await import(
  "./MCPConnectSection"
);

const PLACEHOLDER = "{https://your-bytebase-url.com}";

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

const tabs = (container: HTMLElement) =>
  [...container.querySelectorAll('[role="tab"]')] as HTMLElement[];

const openTab = (container: HTMLElement, title: string) => {
  const tab = tabs(container).find((t) => t.textContent === title);
  if (!tab) throw new Error(`no tab ${title}`);
  act(() => tab.click());
};

const stepOne = (container: HTMLElement) =>
  container.querySelector('[role="tabpanel"]') as HTMLElement;

// Base UI renders only the active panel, so each tab has to be visited.
const eachToolText = (container: HTMLElement) =>
  tabs(container).map((tab) => {
    act(() => tab.click());
    return stepOne(container).querySelector("pre")?.textContent ?? "";
  });

// Tools set up by copying text. VS Code is set up through a link instead.
const copyTabs = (container: HTMLElement) =>
  tabs(container).filter((tab) => tab.textContent !== "VS Code");

const copyButton = (root: HTMLElement) =>
  root.querySelector("button[aria-label]") as HTMLButtonElement;

beforeEach(() => {
  serverState.externalUrl = "https://bb.example.com";
  serverState.needConfigureExternalUrl = false;
});

describe("MCPConnectSection", () => {
  test("scrolls tool tabs horizontally without vertical overflow", () => {
    const { container, unmount } = render(<MCPConnectSection />);
    const tabList = container.querySelector('[role="tablist"]');

    expect(tabList?.classList.contains("overflow-x-auto")).toBe(true);
    expect(tabList?.classList.contains("overflow-y-hidden")).toBe(true);
    unmount();
  });

  // Each command was checked against its own CLI: codex takes the name
  // positionally and the address as --url with no --transport, and the Copilot
  // binary is `copilot`, not `gh copilot`.
  test("no command uses a flag its CLI does not have", () => {
    const { container, unmount } = render(<MCPConnectSection />);
    const values = eachToolText(container).filter((value) => value !== "");
    expect(values).toHaveLength(6);
    for (const value of values) {
      expect(value).not.toContain("--name ");
      expect(value).not.toMatch(/^gh /);
      if (value.startsWith("codex ")) {
        expect(value).toContain("--url ");
      } else if (value.includes(" mcp add")) {
        expect(value).not.toContain("--url ");
      }
    }
    unmount();
  });

  // claude_desktop_config.json skips an entry that has a URL and no command,
  // so the entry has to start a local process that forwards to this instance.
  test("the Claude Desktop entry starts the npm package with this instance's address", () => {
    const { container, unmount } = render(<MCPConnectSection />);
    openTab(container, "Claude Desktop");
    const panel = stepOne(container);
    const entry = JSON.parse(`{${panel.querySelector("pre")?.textContent}}`);

    expect(entry.mcpServers.bytebase).toEqual({
      command: "npx",
      args: ["-y", "@bytebase/bytebase-mcp"],
      env: { BYTEBASE_URL: "https://bb.example.com" },
    });
    expect(panel.textContent).toContain(
      "settings.mcp.connect.desktop.open-folder"
    );
    unmount();
  });

  // The page cannot tell whether VS Code is installed, so the tab offers the
  // install link and points to the guide for the other way.
  test("the VS Code tab offers an install link and its setup guide", () => {
    const { container, unmount } = render(<MCPConnectSection />);
    openTab(container, "VS Code");
    const panel = stepOne(container);
    const [install, guide] = [...panel.querySelectorAll("a")];
    const [scheme, query] = install.getAttribute("href")?.split("?") ?? [];

    expect(scheme).toBe("vscode:mcp/install");
    expect(JSON.parse(decodeURIComponent(query))).toEqual({
      name: "bytebase",
      type: "http",
      url: "https://bb.example.com/mcp",
    });
    expect(guide.href).toBe(`${MCP_SETUP_GUIDE_URL}#vs-code`);
    expect(panel.querySelector("pre")).toBeNull();
    unmount();
  });

  test("the sign-in step follows the selected tool", () => {
    const { container, unmount } = render(<MCPConnectSection />);
    expect(container.textContent).toContain(
      "settings.mcp.connect.sign-in.claude-code"
    );

    openTab(container, "Claude Desktop");
    expect(container.textContent).toContain(
      "settings.mcp.connect.sign-in.claude-desktop"
    );
    expect(container.textContent).not.toContain(
      "settings.mcp.connect.sign-in.claude-code"
    );

    openTab(container, "Codex");
    expect(container.textContent).toContain(
      "settings.mcp.connect.sign-in.default"
    );
    unmount();
  });

  test("links to the setup guide", () => {
    const { container, unmount } = render(<MCPConnectSection />);
    const hrefs = [...container.querySelectorAll("a")].map((a) => a.href);
    expect(hrefs).toContain(MCP_SETUP_GUIDE_URL);
    unmount();
  });

  describe("with an External URL", () => {
    test("every tool's text carries it and can be copied", () => {
      const { container, unmount } = render(<MCPConnectSection />);
      for (const tab of copyTabs(container)) {
        act(() => tab.click());
        const panel = stepOne(container);
        expect(panel.querySelector("pre")?.textContent).toContain(
          "https://bb.example.com"
        );
        expect(copyButton(panel).disabled).toBe(false);
      }
      expect(container.textContent).not.toContain(
        "settings.mcp.connect.copy-off"
      );
      openTab(container, "VS Code");
      expect(container.textContent).not.toContain(
        "settings.mcp.connect.vscode.off"
      );
      unmount();
    });
  });

  describe("without an External URL", () => {
    beforeEach(() => {
      serverState.externalUrl = "";
      serverState.needConfigureExternalUrl = true;
    });

    test("every tool's text shows the placeholder and cannot be copied, with the reason shown", () => {
      const { container, unmount } = render(<MCPConnectSection />);
      for (const tab of copyTabs(container)) {
        act(() => tab.click());
        const panel = stepOne(container);
        expect(panel.querySelector("pre")?.textContent).toContain(PLACEHOLDER);
        expect(copyButton(panel).disabled).toBe(true);
        expect(panel.textContent).toContain("settings.mcp.connect.copy-off");
      }
      unmount();
    });

    test("no link carries the placeholder, and install is a disabled button with the reason shown", () => {
      const { container, unmount } = render(<MCPConnectSection />);
      openTab(container, "VS Code");
      const panel = stepOne(container);

      for (const link of container.querySelectorAll("a")) {
        expect(decodeURIComponent(link.getAttribute("href") ?? "")).not.toContain(
          PLACEHOLDER
        );
      }
      const install = [...panel.querySelectorAll("button")].find((button) =>
        button.textContent?.includes("settings.mcp.connect.vscode.install")
      );
      expect(install?.disabled).toBe(true);
      expect(panel.textContent).toContain("settings.mcp.connect.vscode.off");
      unmount();
    });

    test("the first prompt holds no address and stays copyable", () => {
      const { container, unmount } = render(<MCPConnectSection />);
      const prompt = [...container.querySelectorAll("pre")].find(
        (pre) => pre.textContent === "settings.mcp.first-prompt.example"
      );
      const block = prompt?.parentElement as HTMLElement;
      expect(copyButton(block).disabled).toBe(false);
      unmount();
    });
  });
});
