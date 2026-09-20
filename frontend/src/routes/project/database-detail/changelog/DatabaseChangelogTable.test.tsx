import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { shownTimestampModes } from "@/test-utils/humanizeTs";
import { ChangelogSchema } from "@/types/proto-es/v1/changelog_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    push: vi.fn(),
  },
}));

vi.mock("@/utils/v1/changelog", () => ({
  changelogLink: () => "/projects/p1/databases/d1/changelogs/1",
}));

vi.mock("@/types", () => ({
  getTimeForPbTimestampProtoEs: () => new Date("2026-04-15T00:00:00Z").getTime(),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

import { DatabaseChangelogTable } from "./DatabaseChangelogTable";

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const makeChangelog = (name: string) =>
  create(ChangelogSchema, {
    name,
    planTitle: `Plan ${name}`,
  });

beforeEach(() => {
  document.body.innerHTML = "";
});

describe("DatabaseChangelogTable", () => {
  test("dates each row in the embedded history form", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseChangelogTable
        loading={false}
        changelogs={[
          create(ChangelogSchema, {
            name: "changelogs/1",
            createTime: timestampFromMs(Date.UTC(2026, 2, 2, 12)),
          }),
        ]}
      />
    );

    render();

    // A row in a list orients the reader; the changelog's own page testifies.
    expect(
      shownTimestampModes(container)
    ).toEqual(["compact"]);

    unmount();
  });

  test("inherits default zebra striping from shared table primitives", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseChangelogTable
        loading={false}
        changelogs={[
          makeChangelog("changelogs/1"),
          makeChangelog("changelogs/2"),
        ]}
      />
    );

    render();

    const tbody = container.querySelector("tbody");

    expect(tbody?.className).toContain(
      "[&_tr:nth-child(even)]:bg-control-bg/50"
    );

    unmount();
  });
});
