import { create } from "@bufbuild/protobuf";
import { renderHook } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import {
  IssueCommentSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";

const mocks = vi.hoisted(() => ({
  focusPlanPhase: vi.fn(),
  push: vi.fn(),
  requestActivityFocus: vi.fn(),
  requestThreadFocus: vi.fn(),
}));

vi.mock("@/app/router", () => ({ router: { push: mocks.push } }));

vi.mock("../shell/focusPhase", () => ({
  focusPlanPhase: mocks.focusPlanPhase,
}));

vi.mock("../shared/stores/usePlanDetailStore", () => ({
  usePlanDetailStoreApi: () => ({
    getState: () => ({
      requestActivityFocus: mocks.requestActivityFocus,
      requestThreadFocus: mocks.requestThreadFocus,
      selectedSpecId: "spec-1",
    }),
  }),
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({
    expandPhase: vi.fn(),
    plan: create(PlanSchema, {
      specs: [
        create(Plan_SpecSchema, { id: "spec-1" }),
        create(Plan_SpecSchema, { id: "spec-2" }),
      ],
    }),
    planId: "1",
    projectId: "p",
  }),
}));

import { useFocusThread } from "./useFocusThread";

const thread = (spec?: string) =>
  create(IssueCommentSchema, {
    name: "projects/p/issues/1/issueComments/7",
    statementAnchor: spec ? create(StatementAnchorSchema, { spec }) : undefined,
  });

beforeEach(() => vi.clearAllMocks());

test("opens a thread on another change in that change's editor", () => {
  const { result } = renderHook(() => useFocusThread());
  result.current.focusInStatement(thread("spec-2"), 4);
  expect(mocks.requestThreadFocus).toHaveBeenCalledWith({
    commentName: "projects/p/issues/1/issueComments/7",
    specId: "spec-2",
    lineNumber: 4,
  });
  expect(mocks.push).toHaveBeenCalledOnce();
  expect(mocks.requestActivityFocus).not.toHaveBeenCalled();
});

test.each([
  ["without an anchor", undefined],
  ["anchored to a change the plan no longer has", "spec-gone"],
])("sends a thread %s to its Activity card", (_, spec) => {
  const { result } = renderHook(() => useFocusThread());
  result.current.focusInStatement(thread(spec));
  expect(mocks.focusPlanPhase).toHaveBeenCalledWith(
    "review",
    expect.any(Function)
  );
  expect(mocks.requestActivityFocus).toHaveBeenCalledWith(
    "projects/p/issues/1/issueComments/7"
  );
  expect(mocks.requestThreadFocus).not.toHaveBeenCalled();
  expect(mocks.push).not.toHaveBeenCalled();
});
