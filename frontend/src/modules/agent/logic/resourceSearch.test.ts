// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";

const mocks = vi.hoisted(() => ({
  fetchProjectList: vi.fn(),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ fetchProjectList: mocks.fetchProjectList }),
  },
}));

import { searchAgentProjects } from "./resourceSearch";

const project = create(ProjectSchema, {
  name: "projects/project-sample",
  title: "Sample project",
});

describe("searchAgentProjects", () => {
  beforeEach(() => {
    mocks.fetchProjectList.mockReset();
  });

  test("searches accessible active projects by title or ID in pages of 20", async () => {
    const controller = new AbortController();
    mocks.fetchProjectList.mockResolvedValueOnce({
      projects: [project],
      nextPageToken: "next-page",
    });

    await expect(
      searchAgentProjects(" sample ", "", controller.signal)
    ).resolves.toEqual({ items: [project], nextPageToken: "next-page" });
    expect(mocks.fetchProjectList).toHaveBeenCalledExactlyOnceWith({
      filter: {
        query: "sample",
        state: State.ACTIVE,
        excludeDefault: true,
      },
      pageSize: 20,
      pageToken: "",
      signal: controller.signal,
      silent: true,
      orderBy: "title",
      cache: true,
    });
  });

  test("forwards the continuation token and normalizes an exhausted result", async () => {
    const controller = new AbortController();
    mocks.fetchProjectList.mockResolvedValueOnce({
      projects: [],
    });

    await expect(
      searchAgentProjects("", "continuation", controller.signal)
    ).resolves.toEqual({ items: [], nextPageToken: "" });
    expect(mocks.fetchProjectList).toHaveBeenCalledWith(
      expect.objectContaining({
        pageToken: "continuation",
        filter: expect.objectContaining({ query: "" }),
      })
    );
  });

  test("propagates fetch failures so the picker can offer Retry", async () => {
    const controller = new AbortController();
    const failure = new Error("projects unavailable");
    mocks.fetchProjectList.mockRejectedValueOnce(failure);

    await expect(
      searchAgentProjects("sample", "", controller.signal)
    ).rejects.toBe(failure);
  });
});
