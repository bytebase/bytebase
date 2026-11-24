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
    onExpandedChange={() => undefined}
    {...props}
  />
);

const rowItems = (container: HTMLElement) => [
  ...container.querySelectorAll("li"),
];

const servedIds = (container: HTMLElement) =>
  rowItems(container)
    .filter((item) => item.textContent?.includes("settings.mcp.ladder.tier."))
    .map(
      (item) =>
        MCP_CAPABILITY_ROWS.find((row) =>
          item.textContent?.includes(`row.${row.id}.title`)
        )?.id
    );

describe("MCPCapabilityLadder", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("rows split into allowed, then not allowed ending with the floor", () => {
    const groupItems = (container: HTMLElement) =>
      [...container.querySelectorAll("ul")].map((list) =>
        [...list.querySelectorAll("li")].map((item) => {
          const text = item.textContent ?? "";
          if (text.includes("settings.mcp.ladder.floor.text")) return "floor";
          return MCP_CAPABILITY_ROWS.find((row) =>
            text.includes(`row.${row.id}.title`)
          )?.id;
        })
      );

    const readOnly = renderIntoContainer(
      ladder({ mode: MCPSetting_Capability.READ_ONLY })
    );
    expect(groupItems(readOnly.container)).toEqual([
      ["read-schemas", "read-data", "read-workflow", "read-access"],
      [
        "propose",
        "run-statements",
        "request-access",
        "approve",
        "masking",
        "add-instances",
        "manage",
        "floor",
      ],
    ]);
    readOnly.unmount();

    const readWrite = renderIntoContainer(
      ladder({ mode: MCPSetting_Capability.READ_WRITE })
    );
    expect(groupItems(readWrite.container)).toEqual([
      MCP_CAPABILITY_ROWS.map((row) => row.id),
      ["floor"],
    ]);
    readWrite.unmount();
  });

  test("the tier tag rides the served rows only", () => {
    const readOnly = renderIntoContainer(
      ladder({ mode: MCPSetting_Capability.READ_ONLY })
    );
    expect(servedIds(readOnly.container)).toEqual([
      "read-schemas",
      "read-data",
      "read-workflow",
      "read-access",
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
    expect(container.querySelector("a")).toBeNull();
    unmount();
  });

  test("expanded, the two groups are headed allowed and not allowed", () => {
    const { container, unmount } = renderIntoContainer(ladder({}));
    const headings = [...container.querySelectorAll("h4")].map(
      (node) => node.textContent
    );
    expect(headings).toEqual([
      "settings.mcp.ladder.heading",
      "settings.mcp.ladder.heading-refused",
    ]);
    expect(
      container
        .querySelector('[data-testid="mcp-ladder-trigger"]')
        ?.closest("h4")
    ).not.toBeNull();
    expect(container.textContent).not.toContain(
      "settings.mcp.ladder.summary.read-only"
    );
    unmount();
  });

  test("what each row covers is a docs link, not a details control", () => {
    const { container, unmount } = renderIntoContainer(ladder({}));
    const link = container.querySelector("a");
    expect(link?.getAttribute("href")).toBe(
      "https://docs.bytebase.com/integrations/mcp#what-each-mode-allows"
    );
    expect(link?.getAttribute("target")).toBe("_blank");
    expect(link?.textContent).toContain("settings.mcp.ladder.docs-link");
    expect(container.textContent).not.toContain("details");
    expect(container.querySelectorAll("button")).toHaveLength(1);
    unmount();
  });

  test("prioritizes granted capability titles over refused titles", () => {
    const { container, unmount } = renderIntoContainer(ladder({}));
    const allowed = [...container.querySelectorAll("span")].find((node) =>
      node.textContent?.includes("settings.mcp.ladder.row.read-schemas.title")
    );
    const refused = [...container.querySelectorAll("span")].find((node) =>
      node.textContent?.includes("settings.mcp.ladder.row.propose.title")
    );

    expect(allowed).toHaveClass("text-main");
    expect(refused).toHaveClass("text-control");
    unmount();
  });

  test("toggling asks the owner rather than keeping its own state", () => {
    const expanded: boolean[] = [];
    const { container, unmount } = renderIntoContainer(
      ladder({ onExpandedChange: (next) => expanded.push(next) })
    );
    act(() => container.querySelector("button")?.click());
    expect(expanded).toEqual([false]);
    unmount();
  });

  test("uses spacing rather than row borders to separate capabilities", () => {
    const { container, unmount } = renderIntoContainer(ladder({}));
    for (const list of container.querySelectorAll("ul")) {
      expect(list).toHaveClass("gap-y-2");
    }
    for (const row of rowItems(container)) {
      expect(row).not.toHaveClass("border-b");
    }
    unmount();
  });

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

    const floor = rowItems(container).find((node) =>
      node.textContent?.includes("settings.mcp.ladder.floor.text")
    );
    expect(floor?.querySelector("span.sr-only")?.textContent).toBe(
      "settings.mcp.ladder.mark.refused"
    );
    expect(floor?.textContent).not.toContain("settings.mcp.ladder.tier.");
    unmount();
  });
});
