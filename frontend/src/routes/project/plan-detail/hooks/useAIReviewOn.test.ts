import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  serverInfo: undefined as
    | { aiEnabled: boolean; aiSqlReviewEnabled: boolean }
    | undefined,
  licensed: true,
}));

vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: () => mocks.licensed,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: { serverInfo: typeof mocks.serverInfo }) => unknown
  ) => selector({ serverInfo: mocks.serverInfo }),
}));

import { useAIReviewOn } from "./useAIReviewOn";

const reviewOn = () => renderHook(() => useAIReviewOn()).result.current;

beforeEach(() => {
  mocks.serverInfo = { aiEnabled: true, aiSqlReviewEnabled: true };
  mocks.licensed = true;
});

describe("useAIReviewOn", () => {
  test("is on while the workspace has AI and its review switch on", () => {
    expect(reviewOn()).toBe(true);
  });

  test.each([
    [
      "the workspace review switch is off",
      { aiEnabled: true, aiSqlReviewEnabled: false },
    ],
    [
      "AI is off for the workspace",
      { aiEnabled: false, aiSqlReviewEnabled: true },
    ],
    ["the server info is not read yet", undefined],
  ])("is off when %s", (_name, serverInfo) => {
    mocks.serverInfo = serverInfo;
    expect(reviewOn()).toBe(false);
  });

  test("is off without the Enterprise plan, whatever the switches say", () => {
    mocks.licensed = false;
    expect(reviewOn()).toBe(false);
  });
});
