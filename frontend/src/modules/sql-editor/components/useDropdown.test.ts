import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import type {
  WorksheetFilter,
  WorksheetFolderNode,
} from "@/modules/sql-editor/model/Sheet";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import { useDropdown } from "./useDropdown";

const mocks = vi.hoisted(() => ({
  currentUser: { email: "me@example.com" },
  worksheet: undefined as Worksheet | undefined,
  isWorksheetWritableV1: vi.fn(() => true),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      getWorksheetByName: () => mocks.worksheet,
    }),
}));

vi.mock("@/utils", () => ({
  isWorksheetWritableV1: mocks.isWorksheetWritableV1,
}));

const baseFilter: WorksheetFilter = {
  keyword: "",
  onlyShowStarred: false,
  showMine: true,
  showShared: true,
  showDraft: true,
};

const worksheetNode: WorksheetFolderNode = {
  key: "/my/sheet",
  label: "sheet",
  editable: true,
  children: [],
  worksheet: {
    name: "projects/proj/worksheets/sheet",
    title: "sheet",
    folders: [],
    type: "worksheet",
  },
};

const event = {
  preventDefault: vi.fn(),
  stopPropagation: vi.fn(),
} as unknown as React.MouseEvent;

describe("useDropdown", () => {
  afterEach(() => {
    vi.clearAllMocks();
    mocks.worksheet = undefined;
  });

  test("hides delete for writable shared worksheets", () => {
    mocks.worksheet = {
      name: "projects/proj/worksheets/sheet",
      creator: "users/other@example.com",
    } as Worksheet;

    const { result } = renderHook(() =>
      useDropdown("shared", baseFilter, false)
    );

    act(() => {
      result.current.handleContextMenu(event, worksheetNode);
    });

    expect(
      result.current.options.map((item) => item.type === "item" && item.key)
    ).toEqual(["duplicate", "rename"]);
  });

  test("keeps delete for writable my worksheets", () => {
    mocks.worksheet = {
      name: "projects/proj/worksheets/sheet",
      creator: "users/me@example.com",
    } as Worksheet;

    const { result } = renderHook(() => useDropdown("my", baseFilter, false));

    act(() => {
      result.current.handleContextMenu(event, worksheetNode);
    });

    expect(
      result.current.options.map((item) => item.type === "item" && item.key)
    ).toEqual(["duplicate", "share", "rename", "delete"]);
  });
});
