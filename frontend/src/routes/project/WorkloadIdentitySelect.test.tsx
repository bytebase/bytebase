import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

type ComboProps = {
  value: string;
  onChange: (value: string) => void;
  className?: string;
  onSearch?: (query: string) => void | Promise<void>;
  hasMore?: boolean;
  loadingMore?: boolean;
  onLoadMore?: () => void | Promise<void>;
  options: { value: string; label: string; description?: string }[];
};

const combo: { props?: ComboProps } = {};

const mocks = vi.hoisted(() => ({
  listWorkloadIdentities: vi.fn(),
  getWorkloadIdentity: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores/app", () => {
  const state = () => ({
    listWorkloadIdentities: mocks.listWorkloadIdentities,
    getWorkloadIdentity: mocks.getWorkloadIdentity,
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("@/components/ui/combobox", () => ({
  Combobox: (props: ComboProps) => {
    combo.props = props;
    return null;
  },
}));

vi.mock("@/utils", () => ({
  getDefaultPagination: () => 50,
}));

let WorkloadIdentitySelect: typeof import("./WorkloadIdentitySelect").WorkloadIdentitySelect;

const workloadIdentity = (
  name: string,
  title: string,
  email = `${title.toLowerCase()}@example.com`
): WorkloadIdentity => ({ name, title, email }) as WorkloadIdentity;

const render = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  act(() => root.render(element));
  return {
    container,
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

beforeEach(async () => {
  vi.clearAllMocks();
  combo.props = undefined;
  mocks.listWorkloadIdentities.mockResolvedValue({
    workloadIdentities: [],
    nextPageToken: "",
  });
  mocks.getWorkloadIdentity.mockImplementation((name: string) =>
    workloadIdentity(name, name)
  );
  ({ WorkloadIdentitySelect } = await import("./WorkloadIdentitySelect"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("WorkloadIdentitySelect", () => {
  test("fills the available width while preserving the caller width cap", async () => {
    const { unmount } = render(
      <WorkloadIdentitySelect
        projectName="projects/p"
        value=""
        onChange={() => {}}
        className="max-w-lg"
      />
    );
    await flush();

    expect(combo.props?.className).toContain("w-full");
    expect(combo.props?.className).toContain("max-w-lg");
    unmount();
  });

  test("fetches only the first page on mount", async () => {
    mocks.listWorkloadIdentities.mockResolvedValue({
      workloadIdentities: [workloadIdentity("workloadIdentities/one", "One")],
      nextPageToken: "page-2",
    });

    const { unmount } = render(
      <WorkloadIdentitySelect
        projectName="projects/p"
        value=""
        onChange={() => {}}
      />
    );
    await flush();

    expect(mocks.listWorkloadIdentities).toHaveBeenCalledTimes(1);
    expect(mocks.listWorkloadIdentities).toHaveBeenCalledWith({
      parent: "projects/p",
      filter: { query: "" },
      pageSize: 50,
      pageToken: "",
      showDeleted: false,
    });
    expect(combo.props?.options).toEqual([
      expect.objectContaining({
        value: "workloadIdentities/one",
        label: "One",
      }),
    ]);
    expect(combo.props?.hasMore).toBe(true);
    unmount();
  });

  test("loads and appends the next page", async () => {
    mocks.listWorkloadIdentities
      .mockResolvedValueOnce({
        workloadIdentities: [
          workloadIdentity("workloadIdentities/one", "One"),
        ],
        nextPageToken: "page-2",
      })
      .mockResolvedValueOnce({
        workloadIdentities: [
          workloadIdentity("workloadIdentities/two", "Two"),
        ],
        nextPageToken: "",
      });

    const { unmount } = render(
      <WorkloadIdentitySelect
        projectName="projects/p"
        value=""
        onChange={() => {}}
      />
    );
    await flush();

    await act(async () => {
      await combo.props?.onLoadMore?.();
    });

    expect(mocks.listWorkloadIdentities).toHaveBeenLastCalledWith(
      expect.objectContaining({
        filter: { query: "" },
        pageToken: "page-2",
      })
    );
    expect(combo.props?.options.map((option) => option.value)).toEqual([
      "workloadIdentities/one",
      "workloadIdentities/two",
    ]);
    expect(combo.props?.hasMore).toBe(false);
    unmount();
  });

  test("starts a new paginated request for backend search", async () => {
    mocks.listWorkloadIdentities
      .mockResolvedValueOnce({
        workloadIdentities: [
          workloadIdentity("workloadIdentities/one", "One"),
        ],
        nextPageToken: "page-2",
      })
      .mockResolvedValueOnce({
        workloadIdentities: [
          workloadIdentity("workloadIdentities/search", "Search result"),
        ],
        nextPageToken: "search-page-2",
      });

    const { unmount } = render(
      <WorkloadIdentitySelect
        projectName="projects/p"
        value=""
        onChange={() => {}}
      />
    );
    await flush();

    await act(async () => {
      await combo.props?.onSearch?.("search");
    });

    expect(mocks.listWorkloadIdentities).toHaveBeenLastCalledWith(
      expect.objectContaining({
        filter: { query: "search" },
        pageToken: "",
      })
    );
    expect(combo.props?.options.map((option) => option.value)).toEqual([
      "workloadIdentities/search",
    ]);
    expect(combo.props?.hasMore).toBe(true);
    unmount();
  });

  test("keeps the selected identity available when it is not on the page", async () => {
    const selected = workloadIdentity(
      "workloadIdentities/selected",
      "Selected identity"
    );
    mocks.getWorkloadIdentity.mockReturnValue(selected);

    const { unmount } = render(
      <WorkloadIdentitySelect
        projectName="projects/p"
        value={selected.name}
        onChange={() => {}}
      />
    );
    await flush();

    expect(combo.props?.options).toContainEqual(
      expect.objectContaining({
        value: selected.name,
        label: selected.title,
      })
    );
    unmount();
  });
});
