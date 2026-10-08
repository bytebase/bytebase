import { create } from "@bufbuild/protobuf";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  InstanceSchema,
  type UpdateInstanceRequest,
} from "@/types/proto-es/v1/instance_service_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { InstanceAssignmentSheet } from "./InstanceAssignmentSheet";

const mocks = vi.hoisted(() => ({
  fetchInstanceList: vi.fn(),
  fetchProjectList: vi.fn(),
  batchUpdateInstances: vi.fn(),
  getOrFetchInstanceByName: vi.fn(),
  refreshServerInfo: vi.fn(),
  updateDatabaseInstance: vi.fn(),
  onOpenChange: vi.fn(),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: typeof mocks) => unknown) => selector(mocks),
    { getState: () => mocks }
  ),
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => ({ activatedInstanceCount: 1 }),
  useSubscriptionState: () => ({
    instanceLicenseCount: 1,
    currentPlan: PlanType.ENTERPRISE,
  }),
}));
vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => true,
  hasProjectPermissionV2: () => true,
  extractInstanceResourceName: (name: string) => name.split("/").at(-1),
  extractProjectResourceName: (name: string) =>
    name.startsWith("projects/") ? name.split("/")[1] : "",
}));
vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: () => null,
}));
vi.mock("@/components/LearnMoreLink", () => ({ LearnMoreLink: () => null }));
vi.mock("@/hooks/usePagedData", () => ({
  PagedTableFooter: ({
    hasMore,
    onLoadMore,
  }: {
    hasMore: boolean;
    onLoadMore: () => void;
  }) =>
    hasMore ? (
      <button type="button" onClick={onLoadMore}>
        Load more
      </button>
    ) : null,
}));

const workspace = create(InstanceSchema, {
  name: "instances/shared",
  title: "Shared",
  activation: true,
});
const project = create(InstanceSchema, {
  name: "projects/app/instances/dedicated",
  title: "Dedicated",
});

beforeEach(() => {
  vi.resetAllMocks();
  mocks.fetchProjectList.mockResolvedValue({
    projects: [{ name: "projects/app" }],
    nextPageToken: "",
  });
  mocks.fetchInstanceList.mockImplementation(async ({ parent }) => ({
    instances: parent === "projects/app" ? [project] : [workspace],
    nextPageToken: "",
  }));
  mocks.getOrFetchInstanceByName.mockImplementation(async (name) =>
    name === project.name ? project : workspace
  );
  mocks.batchUpdateInstances.mockImplementation(
    async (requests: UpdateInstanceRequest[]) =>
      requests.map((request) => request.instance)
  );
});
afterEach(cleanup);

test("lists project instances and releases licenses before assigning across scopes", async () => {
  render(<InstanceAssignmentSheet open onOpenChange={mocks.onOpenChange} />);
  await screen.findByText("Dedicated");
  const sharedRow = screen.getByText("Shared").closest("tr")!;
  const dedicatedRow = screen.getByText("Dedicated").closest("tr")!;
  fireEvent.click(within(sharedRow).getByRole("checkbox"));
  fireEvent.click(within(dedicatedRow).getByRole("checkbox"));
  fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));
  await waitFor(() => expect(mocks.onOpenChange).toHaveBeenCalledWith(false));
  expect(
    mocks.batchUpdateInstances.mock.calls.map(([requests, parent]) => ({
      parent,
      changes: requests.map((request: UpdateInstanceRequest) => [
        request.instance?.name,
        request.instance?.activation,
      ]),
    }))
  ).toEqual([
    { parent: undefined, changes: [[workspace.name, false]] },
    { parent: "projects/app", changes: [[project.name, true]] },
  ]);
});

test("paginates projects and keeps instance continuation tokens within their parent", async () => {
  mocks.fetchProjectList
    .mockResolvedValueOnce({
      projects: [{ name: "projects/empty" }],
      nextPageToken: "projects-next",
    })
    .mockResolvedValueOnce({
      projects: [{ name: "projects/app" }],
      nextPageToken: "",
    });
  mocks.fetchInstanceList.mockImplementation(async ({ parent, pageToken }) => {
    if (!parent) return { instances: [], nextPageToken: "" };
    if (parent === "projects/empty")
      return { instances: [], nextPageToken: "" };
    return {
      instances: [
        pageToken
          ? project
          : create(InstanceSchema, {
              name: "projects/app/instances/first",
              title: "First",
            }),
      ],
      nextPageToken: pageToken ? "" : "instances-next",
    };
  });
  render(<InstanceAssignmentSheet open onOpenChange={mocks.onOpenChange} />);
  await screen.findByText("First");
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  await screen.findByText("Dedicated");
  expect(mocks.fetchProjectList).toHaveBeenCalledWith(
    expect.objectContaining({
      pageToken: "projects-next",
      filter: { state: State.ACTIVE, excludeDefault: true },
    })
  );
  expect(mocks.fetchInstanceList).toHaveBeenLastCalledWith(
    expect.objectContaining({
      parent: "projects/app",
      pageToken: "instances-next",
    })
  );
  expect(screen.queryByRole("button", { name: "Load more" })).toBeNull();
});

test("counts licenses on instances outside the loaded page", async () => {
  mocks.fetchInstanceList.mockImplementation(async ({ parent }) => ({
    instances: parent ? [project] : [],
    nextPageToken: "",
  }));
  render(<InstanceAssignmentSheet open onOpenChange={mocks.onOpenChange} />);
  await screen.findByText("Dedicated");
  fireEvent.click(
    within(screen.getByText("Dedicated").closest("tr")!).getByRole("checkbox")
  );
  expect(screen.getByRole("button", { name: "common.confirm" })).toBeDisabled();
  expect(mocks.batchUpdateInstances).not.toHaveBeenCalled();
});

test("keeps updates for different projects in separate batches", async () => {
  const active = create(InstanceSchema, { ...project, activation: true });
  const other = create(InstanceSchema, {
    name: "projects/other/instances/other",
    title: "Other",
  });
  mocks.fetchProjectList.mockResolvedValue({
    projects: [{ name: "projects/app" }, { name: "projects/other" }],
    nextPageToken: "",
  });
  mocks.fetchInstanceList.mockImplementation(async ({ parent }) => ({
    instances:
      parent === "projects/app"
        ? [active]
        : parent === "projects/other"
          ? [other]
          : [],
    nextPageToken: "",
  }));
  mocks.getOrFetchInstanceByName.mockResolvedValue(other);
  render(<InstanceAssignmentSheet open onOpenChange={mocks.onOpenChange} />);
  await screen.findByText("Other");
  fireEvent.click(
    within(screen.getByText("Dedicated").closest("tr")!).getByRole("checkbox")
  );
  fireEvent.click(
    within(screen.getByText("Other").closest("tr")!).getByRole("checkbox")
  );
  fireEvent.click(screen.getByRole("button", { name: "common.confirm" }));
  await waitFor(() => expect(mocks.onOpenChange).toHaveBeenCalledWith(false));
  expect(
    mocks.batchUpdateInstances.mock.calls.map(([requests, parent]) => ({
      parent,
      names: requests.map(
        (request: UpdateInstanceRequest) => request.instance?.name
      ),
    }))
  ).toEqual([
    { parent: "projects/app", names: [active.name] },
    { parent: "projects/other", names: [other.name] },
  ]);
});
