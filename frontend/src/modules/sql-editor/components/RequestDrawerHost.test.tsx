import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { RequestDrawerHost, useRequestDrawerHost } from "./RequestDrawerHost";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("./theme/useActiveSQLEditorTheme", () => ({
  useActiveSQLEditorTheme: () => ({ id: "light", tokens: {} }),
}));

vi.mock("./AccessGrantRequestDrawer", () => ({
  AccessGrantRequestDrawer: ({
    query,
    targets,
    schema,
    onClose,
  }: {
    query?: string;
    targets: string[];
    schema?: string;
    onClose: () => void;
  }) => (
    <div
      data-testid="grant-drawer"
      data-query={query}
      data-targets={targets.join(",")}
      data-schema={schema}
    >
      <button type="button" onClick={onClose}>
        Close draft
      </button>
    </div>
  ),
}));

vi.mock("@/modules/access-control/RequestRoleSheet", () => ({
  RequestRoleSheet: () => null,
}));

function Controls({ onAttempt }: { onAttempt: (opened: boolean) => void }) {
  const host = useRequestDrawerHost();
  return (
    <>
      <button
        type="button"
        onClick={() =>
          onAttempt(
            host?.openAccessGrantDrawer({
              query: "SELECT id FROM users",
              targets: ["instances/prod/databases/app"],
              schema: "public",
            }) ?? false
          )
        }
      >
        Open first
      </button>
      <button
        type="button"
        onClick={() =>
          onAttempt(
            host?.openAccessGrantDrawer({
              query: "SELECT secret FROM users",
              targets: ["instances/prod/databases/other"],
            }) ?? false
          )
        }
      >
        Open second
      </button>
    </>
  );
}

const roots: Array<ReturnType<typeof createRoot>> = [];
afterEach(() => {
  for (const root of roots.splice(0)) {
    act(() => root.unmount());
  }
  document.body.innerHTML = "";
});

describe("RequestDrawerHost", () => {
  test("preserves a prefilled draft while its drawer is open", () => {
    const onAttempt = vi.fn();
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);
    roots.push(root);
    act(() =>
      root.render(
        <RequestDrawerHost>
          <Controls onAttempt={onAttempt} />
        </RequestDrawerHost>
      )
    );

    act(() => container.querySelector<HTMLButtonElement>("button")?.click());
    const drawer = container.querySelector<HTMLElement>(
      "[data-testid='grant-drawer']"
    );
    expect(drawer?.dataset.query).toBe("SELECT id FROM users");
    expect(drawer?.dataset.targets).toBe("instances/prod/databases/app");
    expect(drawer?.dataset.schema).toBe("public");

    act(() =>
      container.querySelectorAll<HTMLButtonElement>("button")[1]?.click()
    );
    expect(
      container.querySelector<HTMLElement>("[data-testid='grant-drawer']")
        ?.dataset.query
    ).toBe("SELECT id FROM users");
    expect(onAttempt.mock.calls.map(([opened]) => opened)).toEqual([
      true,
      false,
    ]);
  });
});
