import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import type {
  SavedQueryFilter,
  SavedQueryFolderNode,
} from "@/modules/sql-editor/model/Sheet";
import type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
import { useDropdown } from "./useDropdown";

const mocks = vi.hoisted(() => ({
  currentUser: { email: "me@example.com" },
  savedQuery: undefined as SavedQuery | undefined,
  isSavedQueryWritableV1: vi.fn(() => true),
  // Mirror the real predicates' creator arm; role-grant cases override
  // per test.
  isSavedQueryDeletableV1: vi.fn(
    (sheet: SavedQuery) => sheet.creator === "users/me@example.com"
  ),
  isSavedQueryShareableV1: vi.fn(
    (sheet: SavedQuery) => sheet.creator === "users/me@example.com"
  ),
  canCreateSavedQueryInProject: vi.fn(() => true),
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
      getSavedQueryByName: () => mocks.savedQuery,
    }),
}));

vi.mock("@/utils", () => ({
  isSavedQueryWritableV1: mocks.isSavedQueryWritableV1,
  isSavedQueryDeletableV1: mocks.isSavedQueryDeletableV1,
  isSavedQueryShareableV1: mocks.isSavedQueryShareableV1,
  canCreateSavedQueryInProject: mocks.canCreateSavedQueryInProject,
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (selector: (s: { project: string }) => unknown) =>
    selector({ project: "projects/proj1" }),
}));

const baseFilter: SavedQueryFilter = {
  keyword: "",
  onlyShowStarred: false,
  showMine: true,
  showShared: true,
  showDraft: true,
};

const savedQueryNode: SavedQueryFolderNode = {
  key: "/my/sheet",
  label: "sheet",
  editable: true,
  children: [],
  savedQuery: {
    name: "projects/proj/savedQueries/sheet",
    title: "sheet",
    folders: [],
    type: "savedQuery",
  },
};

const event = {
  preventDefault: vi.fn(),
  stopPropagation: vi.fn(),
} as unknown as React.MouseEvent;

describe("useDropdown", () => {
  afterEach(() => {
    vi.clearAllMocks();
    mocks.savedQuery = undefined;
  });

  test("hides delete for writable shared saved queries", () => {
    mocks.savedQuery = {
      name: "projects/proj/savedQueries/sheet",
      creator: "users/other@example.com",
    } as SavedQuery;

    const { result } = renderHook(() =>
      useDropdown("shared", baseFilter, false)
    );

    act(() => {
      result.current.handleContextMenu(event, savedQueryNode);
    });

    expect(
      result.current.options.map((item) => item.type === "item" && item.key)
    ).toEqual(["duplicate", "rename"]);
  });

  test("role-granted delete and share reach shared saved queries", () => {
    mocks.savedQuery = {
      name: "projects/proj/savedQueries/sheet",
      creator: "users/other@example.com",
    } as SavedQuery;
    mocks.isSavedQueryDeletableV1.mockReturnValue(true);
    mocks.isSavedQueryShareableV1.mockReturnValue(true);

    const { result } = renderHook(() =>
      useDropdown("shared", baseFilter, false)
    );

    act(() => {
      result.current.handleContextMenu(event, savedQueryNode);
    });

    expect(
      result.current.options.map((item) => item.type === "item" && item.key)
    ).toEqual(["duplicate", "share", "rename", "delete"]);
  });

  test("keeps delete for writable my saved queries", () => {
    mocks.savedQuery = {
      name: "projects/proj/savedQueries/sheet",
      creator: "users/me@example.com",
    } as SavedQuery;

    const { result } = renderHook(() => useDropdown("my", baseFilter, false));

    act(() => {
      result.current.handleContextMenu(event, savedQueryNode);
    });

    expect(
      result.current.options.map((item) => item.type === "item" && item.key)
    ).toEqual(["duplicate", "share", "rename", "delete"]);
  });
});
