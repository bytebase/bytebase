import { create } from "@bufbuild/protobuf";
import { within } from "@testing-library/react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeAll, expect, test } from "vitest";
import i18n from "@/lib/i18n";
import { Task_ReleaseStateSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { ReleaseSpineView } from "./ReleaseSpineView";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

beforeAll(async () => {
  await i18n.changeLanguage("en-US");
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) =>
      ({
        matches: false,
        media: query,
        addEventListener: () => {},
        removeEventListener: () => {},
      }) as unknown as MediaQueryList,
  });
});

const sixty = Array.from({ length: 60 }, (_, i) => ({
  version: String(i + 1).padStart(3, "0"),
  path: `migrations/${String(i + 1).padStart(3, "0")}.sql`,
}));

const renderNotRun = () => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() =>
    root.render(
      createElement(ReleaseSpineView, {
        files: sixty,
        entries: [],
        state: create(Task_ReleaseStateSchema, {
          pendingRanges: [{ start: 0, end: sixty.length }],
        }),
        getSectionLabel: () => "",
      })
    )
  );
  return {
    container,
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

const fileRows = (container: HTMLElement) =>
  within(container).queryAllByTestId(/^spine-file-\d+$/).length;
const click = (container: HTMLElement, name: RegExp) =>
  act(() => within(container).getByRole("button", { name }).click());

test("a long stretch offers Load 20 more and Load all, a short one a single row, and loading is one way", () => {
  const view = renderNotRun();
  const { container } = view;
  expect(within(container).getByTestId("spine-count").textContent).toBe(
    "60 files · 60 pending"
  );
  expect(fileRows(container)).toBe(20);
  expect(
    within(container).getByRole("button", { name: "Load all (40)" })
  ).toBeDefined();
  expect(within(container).queryByTestId("spine-show-fewer")).toBeNull();

  click(container, /^Load 20 more pending$/);
  expect(fileRows(container)).toBe(40);
  // 20 left: one row loads them all.
  expect(within(container).queryByRole("button", { name: /Load all/ })).toBeNull();

  click(container, /^Load 20 more pending$/);
  expect(fileRows(container)).toBe(60);
  expect(within(container).queryByTestId("spine-load-pending")).toBeNull();

  // As in the UIOps log, nothing folds the loaded rows away again.
  expect(within(container).queryByTestId("spine-show-fewer")).toBeNull();
  view.unmount();
});

test("a version whose revision was deleted says so on a short chip and names who and when behind its info icon", () => {
  const files = [
    { version: "001", path: "migrations/001.sql" },
    { version: "002", path: "migrations/002.sql" },
  ];
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() =>
    root.render(
      createElement(ReleaseSpineView, {
        files,
        entries: [],
        state: create(Task_ReleaseStateSchema, {
          pendingRanges: [{ start: 0, end: 2 }],
          deletedRevisions: [
            {
              version: "002",
              deleter: "users/alex@example.com",
              deleteTime: { seconds: BigInt(Date.UTC(2026, 10, 3, 6, 21) / 1000), nanos: 0 },
            },
          ],
        }),
        getSectionLabel: () => "",
      })
    )
  );
  const row = within(container).getByTestId("spine-file-002");
  expect(row.textContent).toContain("Will apply again · revision deleted");
  expect(row.textContent).not.toContain("alex@example.com");
  const info = within(row).getByTestId("spine-revision-deleted-info");
  expect(info.getAttribute("aria-label")).toMatch(/^Earlier revision deleted .*2026.* by alex@example\.com$/);
  expect(info.tabIndex).toBe(0);
  expect(within(container).getByTestId("spine-file-001").textContent).toContain("Will apply");
  act(() => root.unmount());
  container.remove();
});
