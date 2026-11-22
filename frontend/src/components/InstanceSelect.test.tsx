import { act, render } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { InstanceSelect } from "./InstanceSelect";

type ListParams = { parent?: string; pageSize?: number; pageToken?: string };
type ComboboxProps = {
  options: { value: string }[];
  hasMore?: boolean;
  onLoadMore?: () => Promise<void>;
};

const mocks = vi.hoisted(() => ({
  fetchInstanceList: vi.fn(),
  canListProjectInstances: true,
  canListWorkspaceInstances: true,
  combobox: undefined as ComboboxProps | undefined,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ fetchInstanceList: mocks.fetchInstanceList }),
  },
}));
vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: (name: string) => ({ name }),
}));
vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  getDefaultPagination: () => 2,
  hasProjectPermissionV2: () => mocks.canListProjectInstances,
  hasWorkspacePermissionV2: () => mocks.canListWorkspaceInstances,
}));
vi.mock("@/components/ui/combobox", () => ({
  Combobox: (props: ComboboxProps) => {
    mocks.combobox = props;
    return null;
  },
}));
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

const instance = (name: string) => ({ name, title: name, engine: 0 });

// Two project instances over two pages, then three workspace instances.
const pages: Record<string, { names: string[]; next: string }> = {
  "projects/app:": { names: ["projects/app/instances/a"], next: "p2" },
  "projects/app:p2": { names: ["projects/app/instances/b"], next: "" },
  "workspace:": { names: ["instances/x", "instances/y"], next: "w2" },
  "workspace:w2": { names: ["instances/z"], next: "" },
};

const optionNames = () => mocks.combobox!.options.map((o) => o.value);

beforeEach(() => {
  mocks.canListProjectInstances = true;
  mocks.canListWorkspaceInstances = true;
  mocks.fetchInstanceList.mockReset();
  mocks.fetchInstanceList.mockImplementation(async (params: ListParams) => {
    const page = pages[`${params.parent ?? "workspace"}:${params.pageToken}`];
    return {
      instances: page.names.map(instance),
      nextPageToken: page.next,
    };
  });
});

test("lists the project's instances before the workspace's", async () => {
  render(<InstanceSelect value="" onChange={() => {}} project="projects/app" />);
  await act(async () => {});
  expect(optionNames()).toEqual(["projects/app/instances/a"]);

  await act(() => mocks.combobox!.onLoadMore!());
  expect(optionNames()).toEqual([
    "projects/app/instances/a",
    "projects/app/instances/b",
    "instances/x",
    "instances/y",
  ]);

  await act(() => mocks.combobox!.onLoadMore!());
  expect(optionNames()).toEqual([
    "projects/app/instances/a",
    "projects/app/instances/b",
    "instances/x",
    "instances/y",
    "instances/z",
  ]);
  expect(mocks.combobox!.hasMore).toBe(false);
});

test("lists only workspace instances without a project", async () => {
  render(<InstanceSelect value="" onChange={() => {}} />);
  await act(async () => {});
  expect(optionNames()).toEqual(["instances/x", "instances/y"]);
  expect(
    mocks.fetchInstanceList.mock.calls.every(
      (call) => (call[0] as ListParams).parent === undefined
    )
  ).toBe(true);
});

test("skips sources the user cannot list", async () => {
  mocks.canListWorkspaceInstances = false;
  render(<InstanceSelect value="" onChange={() => {}} project="projects/app" />);
  await act(async () => {});
  await act(() => mocks.combobox!.onLoadMore!());
  expect(optionNames()).toEqual([
    "projects/app/instances/a",
    "projects/app/instances/b",
  ]);
  expect(mocks.combobox!.hasMore).toBe(false);

  mocks.canListWorkspaceInstances = true;
  mocks.canListProjectInstances = false;
  mocks.fetchInstanceList.mockClear();
  render(<InstanceSelect value="" onChange={() => {}} project="projects/app" />);
  await act(async () => {});
  expect(mocks.fetchInstanceList).toHaveBeenCalledOnce();
  expect(mocks.fetchInstanceList.mock.calls[0][0].parent).toBeUndefined();
});
