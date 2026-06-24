import { render, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import {
  type SheetStatementSnapshot,
  seedSheetStatement,
  useSheetStatement,
} from "./useSheetStatement";

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
vi.mock("@/utils/v1/sheet", () => ({
  getSheetStatement: (sheet: { statement: string }) => sheet.statement,
  extractSheetUID: (name: string) => name.split("/").pop() ?? "",
}));

const sheet = (statement: string, contentSize = statement.length): Sheet =>
  ({ statement, contentSize }) as unknown as Sheet;

const renderSnapshots = (props: Parameters<typeof useSheetStatement>[0]) => {
  const snapshots: SheetStatementSnapshot[] = [];
  const Probe = () => {
    snapshots.push(useSheetStatement(props));
    return null;
  };
  const result = render(<Probe />);
  return { snapshots, rerender: () => result.rerender(<Probe />) };
};

describe("useSheetStatement", () => {
  beforeEach(() => {
    getSheetByName.mockReset();
    getOrFetchSheetByName.mockReset();
  });

  test("paints a cached remote statement on the first render", () => {
    getSheetByName.mockReturnValue(sheet("select 1;"));
    const { snapshots } = renderSnapshots({
      enabled: true,
      sheetName: "projects/p/sheets/1",
    });
    expect(snapshots[0]).toEqual({
      statement: "select 1;",
      isLoading: false,
      isTruncated: false,
    });
    expect(getOrFetchSheetByName).not.toHaveBeenCalled();
  });

  test("shows loading (not empty) on the first render when a fetch is needed", async () => {
    getSheetByName.mockReturnValue(undefined);
    getOrFetchSheetByName.mockResolvedValue(sheet("select 2;"));
    const { snapshots } = renderSnapshots({
      enabled: true,
      sheetName: "projects/p/sheets/2",
    });
    expect(snapshots[0]).toEqual({
      statement: "",
      isLoading: true,
      isTruncated: false,
    });
    await waitFor(() => {
      expect(snapshots.at(-1)?.statement).toBe("select 2;");
      expect(snapshots.at(-1)?.isLoading).toBe(false);
    });
  });

  test("reads a draft (local) sheet synchronously without fetching", () => {
    const getLocalSheet = vi.fn().mockReturnValue(sheet("select 3;"));
    const { snapshots } = renderSnapshots({
      enabled: true,
      sheetName: "projects/p/sheets/-1",
      getLocalSheet,
    });
    expect(snapshots[0].statement).toBe("select 3;");
    expect(snapshots[0].isLoading).toBe(false);
    expect(getLocalSheet).toHaveBeenCalledWith("projects/p/sheets/-1");
    expect(getOrFetchSheetByName).not.toHaveBeenCalled();
  });

  test("seedSheetStatement marks oversized previews as truncated", () => {
    getSheetByName.mockReturnValue(sheet("select 1;", 9999));
    expect(seedSheetStatement("projects/p/sheets/1")).toEqual({
      statement: "select 1;",
      isLoading: false,
      isTruncated: true,
    });
  });

  test("is empty and not loading when disabled", () => {
    const { snapshots } = renderSnapshots({
      enabled: false,
      sheetName: "projects/p/sheets/1",
    });
    expect(snapshots[0]).toEqual({
      statement: "",
      isLoading: false,
      isTruncated: false,
    });
    expect(getSheetByName).not.toHaveBeenCalled();
  });
});
