import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { SyncDatabases } from "./InstanceFormBody";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("./InstanceFormContext", () => ({
  useInstanceFormContext: () => ({
    hideAdvancedFeatures: false,
    instance: undefined,
    pendingCreateInstance: undefined,
  }),
}));

describe("SyncDatabases", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  test("uses vertically arranged radio choices", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <SyncDatabases
          isCreating={true}
          showLabel={true}
          allowEdit={true}
          onSyncDatabasesChange={() => {}}
        />
      );
    });

    const group = container.querySelector('[role="radiogroup"]');
    expect(group?.classList.contains("flex-col")).toBe(true);
    expect(group?.querySelectorAll('[role="radio"]')).toHaveLength(2);

    act(() => {
      root.unmount();
    });
  });
});
