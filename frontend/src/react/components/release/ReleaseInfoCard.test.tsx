import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { act, createElement, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import {
  type Release,
  Release_FileSchema,
  Release_Type,
  ReleaseSchema,
} from "@/types/proto-es/v1/release_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options && "count" in options ? `${key}:${options.count}` : key,
  }),
}));

let ReleaseInfoCard: typeof import("./ReleaseInfoCard").ReleaseInfoCard;

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

const buildRelease = (fileCount: number): Release =>
  create(ReleaseSchema, {
    name: "projects/p/releases/r1",
    type: Release_Type.VERSIONED,
    createTime: create(TimestampSchema, { seconds: BigInt(0) }),
    files: Array.from({ length: fileCount }, (_, i) =>
      create(Release_FileSchema, {
        version: String(i + 1).padStart(4, "0"),
        path: `V${String(i + 1).padStart(4, "0")}.sql`,
        sheet: `sheet-${i}`,
      })
    ),
  });

describe("ReleaseInfoCard", () => {
  test("renders all files without a truncation tile when count is at the cap", async () => {
    ({ ReleaseInfoCard } = await import("./ReleaseInfoCard"));
    const { container, unmount } = renderIntoContainer(
      createElement(ReleaseInfoCard, {
        release: buildRelease(4),
        releaseName: "projects/p/releases/r1",
      })
    );

    expect(container.textContent).toContain("V0001.sql");
    expect(container.textContent).toContain("V0004.sql");
    expect(container.textContent).not.toContain("release.and-n-more-files");
    unmount();
  });

  test("renders the +N tile when the file list exceeds the display cap", async () => {
    ({ ReleaseInfoCard } = await import("./ReleaseInfoCard"));
    const { container, unmount } = renderIntoContainer(
      createElement(ReleaseInfoCard, {
        release: buildRelease(100),
        releaseName: "projects/p/releases/r1",
      })
    );

    expect(container.textContent).toContain("V0001.sql");
    expect(container.textContent).toContain("V0004.sql");
    expect(container.textContent).not.toContain("V0005.sql");
    expect(container.textContent).toContain("release.and-n-more-files:96");
    unmount();
  });

  test("renders the loading state when isLoading is true", async () => {
    ({ ReleaseInfoCard } = await import("./ReleaseInfoCard"));
    const { container, unmount } = renderIntoContainer(
      createElement(ReleaseInfoCard, {
        isLoading: true,
        releaseName: "projects/p/releases/r1",
      })
    );

    expect(container.textContent).toContain("common.loading");
    unmount();
  });

  test("renders the not-found state when release is missing", async () => {
    ({ ReleaseInfoCard } = await import("./ReleaseInfoCard"));
    const { container, unmount } = renderIntoContainer(
      createElement(ReleaseInfoCard, {
        releaseName: "projects/p/releases/r1",
      })
    );

    expect(container.textContent).toContain("release.not-found");
    unmount();
  });

  test("renders the not-found state when release is the unknown sentinel", async () => {
    ({ ReleaseInfoCard } = await import("./ReleaseInfoCard"));
    // useReleaseByName returns an unknownRelease() sentinel on miss; its
    // name is the placeholder "projects/-1/releases/-1" which fails
    // isValidReleaseName. The component must treat that as not-found.
    const sentinel = create(ReleaseSchema, {
      name: "projects/-1/releases/-1",
    });
    const { container, unmount } = renderIntoContainer(
      createElement(ReleaseInfoCard, {
        release: sentinel,
        releaseName: "projects/p/releases/r1",
      })
    );

    expect(container.textContent).toContain("release.not-found");
    expect(container.textContent).not.toContain("release.files");
    unmount();
  });
});
