import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useEnvironment: () => undefined,
  usePlanFeature: () => false,
}));

vi.mock("@/components/InstanceLabel", () => ({
  InstanceLabel: () => <span>Sample Project Instance</span>,
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({ children }: { children: ReactNode }) => <a>{children}</a>,
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: vi.fn(async () => true),
}));

vi.mock("@/utils", () => ({
  extractInstanceResourceName: (name: string) =>
    name.replace(/^instances\//, ""),
  getInstanceResource: () => ({
    name: "instances/sample-0123456789abcdef",
    title: "Sample Project Instance",
  }),
  hexToRgb: () => [0, 0, 0],
}));

vi.mock("./DatabaseSQLEditorButton", () => ({
  DatabaseSQLEditorButton: () => <button type="button">SQL Editor</button>,
}));

import { DatabaseDetailHeader } from "./DatabaseDetailHeader";

describe("DatabaseDetailHeader", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
  });

  it("truncates long database identifiers without hiding their full values", () => {
    const databaseName = "bb_sample_0123456789abcdef";
    const resourceName = `projects/project-a/instances/sample-0123456789abcdef/databases/${databaseName}`;

    act(() => {
      root.render(
        <DatabaseDetailHeader
          database={{ name: resourceName } as Database}
        />
      );
    });

    const id = container.querySelector(`[title="${databaseName}"]`);
    const resource = container.querySelector(`[title="${resourceName}"]`);
    expect(id?.getAttribute("class")).toContain("min-w-0");
    expect(id?.getAttribute("class")).toContain("truncate");
    expect(resource?.getAttribute("class")).toContain("min-w-0");
    expect(resource?.getAttribute("class")).toContain("truncate");
  });
});
