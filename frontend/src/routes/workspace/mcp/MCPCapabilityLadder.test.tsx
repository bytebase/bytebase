import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { MCP_CAPABILITY_ROWS } from "@/components/mcp/mcpCapabilityRows";
import { MCPSetting_Capability } from "@/types/proto-es/v1/setting_service_pb";
import { MCPCapabilityLadder } from "./MCPCapabilityLadder";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, string>) =>
      vars ? `${key}(${Object.values(vars).join(",")})` : key,
  }),
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
    rerender: (next: ReactElement) => act(() => root.render(next)),
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

const ladder = (props: Partial<Parameters<typeof MCPCapabilityLadder>[0]>) => (
  <MCPCapabilityLadder
    mode={MCPSetting_Capability.READ_ONLY}
    expanded={true}
    details={false}
    onExpandedChange={() => undefined}
    onDetailsChange={() => undefined}
    {...props}
  />
);

const rowItems = (container: HTMLElement) => [
  ...container.querySelectorAll("li:not([role='presentation'])"),
];

const servedIds = (container: HTMLElement) =>
  [...container.querySelectorAll("li")]
    .filter((item) => item.textContent?.includes("settings.mcp.ladder.tier."))
    .map((item) =>
      MCP_CAPABILITY_ROWS.find((row) =>
        item.textContent?.includes(`row.${row.id}.title`)
      )
    )
    .map((row) => row?.id);

describe("MCPCapabilityLadder", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  // Comparing modes must never need a second surface, so an unserved row stays
  // on screen — muted, with no tier tag — rather than disappearing.
  test("every row is listed under every mode, served or not", () => {
    for (const mode of [
      MCPSetting_Capability.READ_ONLY,
      MCPSetting_Capability.READ_WRITE,
    ] as const) {
      const { container, unmount } = renderIntoContainer(ladder({ mode }));
      // One list, so a row's position is announced as "n of 8". The dividers
      // live in it too but are presentational, so they do not inflate that
      // count or shift a row's index.
      expect(container.querySelectorAll("ul")).toHaveLength(1);
      expect(rowItems(container)).toHaveLength(MCP_CAPABILITY_ROWS.length);
      for (const row of MCP_CAPABILITY_ROWS) {
        expect(container.textContent).toContain(
          `settings.mcp.ladder.row.${row.id}.title`
        );
      }
      unmount();
    }
  });

  test("the tier tag rides the served rows only", () => {
    const readOnly = renderIntoContainer(
      ladder({ mode: MCPSetting_Capability.READ_ONLY })
    );
    expect(servedIds(readOnly.container)).toEqual([
      "read-schemas",
      "read-data",
      "read-workflow",
    ]);
    // No "write" badge may appear under a mode that serves no write row.
    expect(readOnly.container.textContent).not.toContain(
      "settings.mcp.ladder.tier.write"
    );
    expect(readOnly.container.textContent).toContain(
      "settings.mcp.ladder.tier.read"
    );
    readOnly.unmount();

    const readWrite = renderIntoContainer(
      ladder({ mode: MCPSetting_Capability.READ_WRITE })
    );
    expect(servedIds(readWrite.container)).toEqual(
      MCP_CAPABILITY_ROWS.map((row) => row.id)
    );
    expect(readWrite.container.textContent).toContain(
      "settings.mcp.ladder.tier.write"
    );
    readWrite.unmount();
  });

  test("collapsed, the trigger is the mode's description and nothing else shows", () => {
    const { container, unmount } = renderIntoContainer(
      ladder({ expanded: false })
    );
    expect(container.textContent).toContain(
      "settings.mcp.ladder.summary.read-only"
    );
    expect(container.querySelectorAll("li")).toHaveLength(0);
    expect(container.textContent).not.toContain(
      "settings.mcp.ladder.floor.label"
    );
    // The details control belongs to the list, so it is absent with the list.
    expect(container.textContent).not.toContain(
      "settings.mcp.ladder.show-details"
    );
    unmount();
  });

  test("expanded, the trigger becomes the list heading naming the mode", () => {
    const { container, unmount } = renderIntoContainer(ladder({}));
    expect(container.textContent).toContain(
      "settings.mcp.ladder.heading(settings.mcp.policy.mode.read-only.title)"
    );
    expect(container.textContent).not.toContain(
      "settings.mcp.ladder.summary.read-only"
    );
    unmount();
  });

  // One second-level control rather than eight: the toggle reveals the
  // sub-item line on every row at once.
  test("the details toggle reveals sub-items on every row at once", () => {
    const { container, rerender, unmount } = renderIntoContainer(ladder({}));
    for (const row of MCP_CAPABILITY_ROWS) {
      expect(container.textContent).not.toContain(
        `settings.mcp.ladder.row.${row.id}.details`
      );
    }
    expect(container.textContent).toContain("settings.mcp.ladder.show-details");

    rerender(ladder({ details: true }));
    for (const row of MCP_CAPABILITY_ROWS) {
      expect(container.textContent).toContain(
        `settings.mcp.ladder.row.${row.id}.details`
      );
    }
    expect(container.textContent).toContain("settings.mcp.ladder.hide-details");
    unmount();
  });

  test("toggling asks the owner rather than keeping its own state", () => {
    const expanded: boolean[] = [];
    const details: boolean[] = [];
    const { container, unmount } = renderIntoContainer(
      ladder({
        onExpandedChange: (next) => expanded.push(next),
        onDetailsChange: (next) => details.push(next),
      })
    );
    const buttons = [...container.querySelectorAll("button")];
    act(() => buttons[0]?.click());
    act(() => buttons[1]?.click());
    expect(expanded).toEqual([false]);
    expect(details).toEqual([true]);
    unmount();
  });

  // The floor is what keeps the list honest about the methods no mode reaches.
  // It belongs to no tier, so it is not a row.
  test("the floor line closes the list without being a row", () => {
    const { container, unmount } = renderIntoContainer(ladder({}));
    expect(container.textContent).toContain("settings.mcp.ladder.floor.label");
    expect(container.textContent).toContain("settings.mcp.ladder.floor.text");
    const floor = [...container.querySelectorAll("p")].find((node) =>
      node.textContent?.includes("settings.mcp.ladder.floor.text")
    );
    expect(floor).toBeDefined();
    expect(floor?.closest("li")).toBeNull();
    unmount();
  });

  // A divider marks a boundary between tiers, so it must not be read as part
  // of the row above it: nested inside that row's <li>, a screen reader
  // announces "Read the change workflow … Read-only stops here" as one item.
  test("each tier is closed by a divider outside its rows, in list order", () => {
    const { container, unmount } = renderIntoContainer(ladder({}));
    for (const tier of ["read", "write"]) {
      const divider = [...container.querySelectorAll("li")].find((node) =>
        node.textContent?.includes(`settings.mcp.ladder.stops.${tier}`)
      );
      expect(divider).toBeDefined();
      // Its own item, and presentational: not content inside the row it closes,
      // and not an item of the eight.
      expect(divider?.getAttribute("role")).toBe("presentation");
      expect(divider?.textContent).not.toContain("settings.mcp.ladder.row.");
    }

    const text = container.textContent ?? "";
    const readStop = text.indexOf("settings.mcp.ladder.stops.read");
    const writeStop = text.indexOf("settings.mcp.ladder.stops.write");
    expect(writeStop).toBeGreaterThan(readStop);
    // The read divider sits between the last read row and the first write row.
    expect(readStop).toBeGreaterThan(
      text.indexOf("settings.mcp.ladder.row.read-workflow.title")
    );
    expect(readStop).toBeLessThan(
      text.indexOf("settings.mcp.ladder.row.propose.title")
    );
    unmount();
  });

  // Served and unserved are otherwise carried by a glyph, muting and the
  // presence of a tier tag — all visual. Without a text alternative a refused
  // row is announced exactly like an allowed one.
  test("every row states whether it is allowed, not only shows it", () => {
    const { container, unmount } = renderIntoContainer(
      ladder({ mode: MCPSetting_Capability.READ_ONLY })
    );
    const markOf = (rowId: string) => {
      const item = rowItems(container).find((node) =>
        node.textContent?.includes(`row.${rowId}.title`)
      );
      return [...(item?.querySelectorAll("span.sr-only") ?? [])]
        .map((node) => node.textContent)
        .join("");
    };
    expect(markOf("read-schemas")).toBe("settings.mcp.ladder.mark.allowed");
    expect(markOf("propose")).toBe("settings.mcp.ladder.mark.refused");
    unmount();
  });
});
