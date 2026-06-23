import { render, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { useDeployTaskStatement } from "./useDeployTaskStatement";

const getSheetByName = vi.fn();
const getOrFetchSheetByName = vi.fn();

vi.mock("@/react/stores/app", () => ({
  useAppStore: {
    getState: () => ({ getSheetByName, getOrFetchSheetByName }),
  },
}));
vi.mock("@/utils/sheet", () => ({
  getStatementSize: (statement: string) => statement.length,
}));
vi.mock("@/utils/v1/issue/rollout", () => ({
  sheetNameOfTaskV1: () => "projects/p1/sheets/1",
}));
vi.mock("@/utils/v1/sheet", () => ({
  getSheetStatement: (sheet: { statement: string }) => sheet.statement,
  extractSheetUID: (name: string) => name.split("/").pop() ?? "",
}));

const task = { name: "tasks/t1" } as unknown as Task;

type HookResult = ReturnType<typeof useDeployTaskStatement>;

const renderHookRecording = () => {
  const renders: HookResult[] = [];
  const Probe = () => {
    renders.push(useDeployTaskStatement({ enabled: true, task }));
    return null;
  };
  render(<Probe />);
  return renders;
};

describe("useDeployTaskStatement initial render (BYT-9763)", () => {
  beforeEach(() => {
    getSheetByName.mockReset();
    getOrFetchSheetByName.mockReset();
  });

  test("renders a cached statement on the first paint — no 'No data' flash", () => {
    getSheetByName.mockReturnValue({ statement: "select 1;", contentSize: 9 });

    const renders = renderHookRecording();

    // The very first render must already carry the statement, so the expanded
    // task never paints the empty "No data" branch.
    expect(renders[0].statement).toBe("select 1;");
    expect(renders[0].isLoading).toBe(false);
  });

  test("reports loading (not empty) on the first paint when the sheet must be fetched", async () => {
    getSheetByName.mockReturnValue(undefined);
    getOrFetchSheetByName.mockResolvedValue({
      statement: "select 2;",
      contentSize: 9,
    });

    const renders = renderHookRecording();

    // First paint: a fetch is pending, so show loading rather than "No data".
    expect(renders[0].isLoading).toBe(true);
    expect(renders[0].statement).toBe("");

    // After the fetch resolves the statement appears.
    await waitFor(() => {
      expect(renders.at(-1)?.statement).toBe("select 2;");
      expect(renders.at(-1)?.isLoading).toBe(false);
    });
  });
});
