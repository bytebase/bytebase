import { create } from "@bufbuild/protobuf";
import { renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, test, vi } from "vitest";
import { IssueCommentSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { PlanDetailStoreContext } from "../shared/stores/usePlanDetailStore";

const mocks = vi.hoisted(() => ({
  comments: [] as unknown[],
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({ getIssueComments: () => mocks.comments }),
}));

import { usePlacementSync } from "./usePlacementSync";

const spec = (id: string, sha: string) =>
  create(Plan_SpecSchema, {
    id,
    config: {
      case: "changeDatabaseConfig",
      value: create(Plan_ChangeDatabaseConfigSchema, {
        sheet: `projects/p/sheets/${sha}`,
      }),
    },
  });

const setup = () => {
  const computePlacements = vi.fn(async () => {});
  const store = {
    getState: () => ({ computePlacements }),
    getInitialState: () => ({ computePlacements }),
    subscribe: () => () => {},
    setState: () => {},
  };
  const wrapper = ({ children }: { children: ReactNode }) => (
    <PlanDetailStoreContext.Provider
      value={store as unknown as React.ContextType<typeof PlanDetailStoreContext>}
    >
      {children}
    </PlanDetailStoreContext.Provider>
  );
  return { computePlacements, wrapper };
};

describe("usePlacementSync", () => {
  test("runs once the page is ready and has an issue, with the project name", () => {
    const { computePlacements, wrapper } = setup();
    const specs = [spec("s1", "a".repeat(64))];
    mocks.comments = [create(IssueCommentSchema, { name: "c1" })];
    const { rerender } = renderHook((props) => usePlacementSync(props), {
      wrapper,
      initialProps: {
        issueName: undefined as string | undefined,
        projectId: "p",
        ready: false,
        specs,
      },
    });
    expect(computePlacements).not.toHaveBeenCalled();
    rerender({ issueName: "projects/p/issues/1", projectId: "p", ready: false, specs });
    expect(computePlacements).not.toHaveBeenCalled();
    rerender({ issueName: "projects/p/issues/1", projectId: "p", ready: true, specs });
    expect(computePlacements).toHaveBeenCalledTimes(1);
    expect(computePlacements).toHaveBeenCalledWith({
      comments: mocks.comments,
      projectName: "projects/p",
      specs,
    });
  });

  test("reruns when comments or a spec's sheet change, not on a mere plan refresh", () => {
    const { computePlacements, wrapper } = setup();
    const base = {
      issueName: "projects/p/issues/1",
      projectId: "p",
      ready: true,
    };
    mocks.comments = [];
    const { rerender } = renderHook((props) => usePlacementSync(props), {
      wrapper,
      initialProps: { ...base, specs: [spec("s1", "a".repeat(64))] },
    });
    expect(computePlacements).toHaveBeenCalledTimes(1);

    // A fresh but identical spec list (polling) does not rerun.
    rerender({ ...base, specs: [spec("s1", "a".repeat(64))] });
    expect(computePlacements).toHaveBeenCalledTimes(1);

    // A new sheet on the spec does.
    const saved = [spec("s1", "b".repeat(64))];
    rerender({ ...base, specs: saved });
    expect(computePlacements).toHaveBeenCalledTimes(2);
    expect(computePlacements).toHaveBeenLastCalledWith(
      expect.objectContaining({ specs: saved })
    );

    // New comments do.
    mocks.comments = [create(IssueCommentSchema, { name: "c1" })];
    rerender({ ...base, specs: saved });
    expect(computePlacements).toHaveBeenCalledTimes(3);
  });
});
