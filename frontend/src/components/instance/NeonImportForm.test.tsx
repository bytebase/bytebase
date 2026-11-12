import {
  ErrorInfoSchema,
  RetryInfoSchema,
} from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import { create } from "@bufbuild/protobuf";
import { anyPack } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import {
  DiscoverNeonBranchesResponseSchema,
  ImportInstancesResponseSchema,
  InstanceImportFailure_CreationOutcome,
  InstanceSchema,
  NeonBranchSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { NeonImportForm } from "./NeonImportForm";

const mocks = vi.hoisted(() => ({
  discover: vi.fn(),
  batch: vi.fn(),
  getInstance: vi.fn(),
  refresh: vi.fn(async () => undefined),
  // Whether the plan lets an instance's sync interval be changed.
  intervalFeature: true,
  license: {
    instanceCountLimit: 100,
    totalInstanceCount: 0,
    instanceLicenseCount: 100,
    activatedInstanceCount: 0,
  },
}));
vi.mock("@/api", () => ({
  instanceServiceClientConnect: {
    discoverNeonBranches: mocks.discover,
    importInstances: mocks.batch,
    getInstance: mocks.getInstance,
  },
}));
const t = (key: string, values?: Record<string, unknown>) =>
  values ? `${key}${JSON.stringify(values)}` : key;
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t }),
}));
vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: vi.fn(),
}));
vi.mock("@/hooks/useAppState", () => ({
  useEnvironmentList: () => [{ id: "prod", title: "Prod" }],
}));
vi.mock("@/components/EngineIcon", () => ({ EngineIcon: () => null }));
vi.mock("@/components/FeatureBadge", () => ({ FeatureBadge: () => null }));
vi.mock("@/components/EnvironmentSelect", () => ({
  EnvironmentSelect: () => null,
}));
vi.mock("@/stores/app", () => {
  const state = {
    instanceCountLimit: () => mocks.license.instanceCountLimit,
    totalInstanceCount: () => mocks.license.totalInstanceCount,
    instanceLicenseCount: () => mocks.license.instanceLicenseCount,
    activatedInstanceCount: () => mocks.license.activatedInstanceCount,
    refreshServerInfo: mocks.refresh,
    hasFeature: () => mocks.intervalFeature,
  };
  return {
    useAppStore: Object.assign(
      (selector: (value: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});

const onDismiss = vi.fn();
const onDirtyChange = vi.fn();

beforeEach(() => {
  window.matchMedia = ((query: string) => ({
    matches: true,
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  })) as unknown as typeof window.matchMedia;
  for (const mock of [mocks.discover, mocks.batch, mocks.getInstance]) {
    mock.mockReset();
  }
  onDismiss.mockClear();
  onDirtyChange.mockClear();
  mocks.refresh.mockReset();
  mocks.refresh.mockResolvedValue(undefined);
  mocks.intervalFeature = true;
  Object.assign(mocks.license, {
    instanceCountLimit: 100,
    totalInstanceCount: 0,
    instanceLicenseCount: 100,
    activatedInstanceCount: 0,
  });
});
afterEach(cleanup);

const branch = (index: number, projectId = "autumn-disk-48433112") =>
  create(NeonBranchSchema, {
    projectId,
    projectTitle: "storefront",
    branchId: `br-${projectId}-${index}`,
    branchTitle: `preview-${index}`,
    databases: [{ name: "app", owner: "app_owner" }],
    roles: ["app_owner", "migrator"],
  });
const branches = (count: number, from = 0) =>
  Array.from({ length: count }, (_, index) => branch(from + index));
const page = (
  list: ReturnType<typeof branch>[],
  nextPageToken = "",
  failures: { projectId: string; reason: string }[] = []
) =>
  create(DiscoverNeonBranchesResponseSchema, {
    branches: list,
    nextPageToken,
    failures: failures.map(({ projectId, reason }) => ({
      projectId,
      status: {
        code: 7,
        message: "Server wording",
        details: [anyPack(ErrorInfoSchema, create(ErrorInfoSchema, { reason }))],
      },
    })),
  });
const reasonError = (code: Code, reason: string) =>
  new ConnectError("Server wording", code, undefined, [
    { desc: ErrorInfoSchema, value: create(ErrorInfoSchema, { reason }) },
  ]);

type BatchRequest = {
  parent?: string;
  credential: { case: "neonCredential"; value: { apiKey: string } };
  imports: { instanceId: string; password?: string }[];
};
const created = (request: BatchRequest) =>
  create(ImportInstancesResponseSchema, {
    results: request.imports.map(({ instanceId }) => ({
      outcome: {
        case: "imported" as const,
        value: create(InstanceSchema, { name: `instances/${instanceId}` }),
      },
    })),
  });
const outcome = (
  request: BatchRequest,
  index: number,
  reason: string,
  creationOutcome: InstanceImportFailure_CreationOutcome
) => {
  const response = created(request);
  response.results[index].outcome = {
    case: "failure",
    value: {
      $typeName: "bytebase.v1.InstanceImportFailure",
      creationOutcome,
      status: {
        $typeName: "google.rpc.Status",
        code: 9,
        message: "Server wording",
        details: [anyPack(ErrorInfoSchema, create(ErrorInfoSchema, { reason }))],
      },
    },
  };
  return response;
};

const button = (name: string | RegExp) => screen.getByRole("button", { name });
const mount = (props: Partial<Parameters<typeof NeonImportForm>[0]> = {}) =>
  render(
    <NeonImportForm
      onDismiss={onDismiss}
      onDirtyChange={onDirtyChange}
      {...props}
    />
  );
const connect = async (key = "napi_example") => {
  fireEvent.change(screen.getByLabelText("instance.neon.connect.api-key"), {
    target: { value: key },
  });
  fireEvent.click(button("instance.neon.connect.find"));
  await screen.findByRole("heading", { name: "instance.neon.steps.select" });
};
const selectAll = () =>
  fireEvent.click(
    screen.getByRole("checkbox", {
      name: /instance.neon.select.select-all\{/,
    })
  );
const startImport = async (count: number) => {
  fireEvent.click(button(new RegExp(`select.configure.*"count":${count}`)));
  fireEvent.click(button(new RegExp(`neon.import-count.*"count":${count}`)));
  const dialog = await screen.findByRole("dialog");
  fireEvent.click(
    within(dialog).getByRole("button", { name: /neon.import-count/ })
  );
};

test("keeps a rejected key and its value on the Connect step", async () => {
  mocks.discover.mockRejectedValue(
    reasonError(Code.FailedPrecondition, "VENDOR_CREDENTIAL_INVALID")
  );
  mount();
  fireEvent.click(button("instance.neon.connect.find"));
  expect(
    screen.getByText("instance.neon.connect.error.key-empty")
  ).toBeVisible();
  expect(mocks.discover).not.toHaveBeenCalled();

  const key = screen.getByLabelText("instance.neon.connect.api-key");
  fireEvent.change(key, { target: { value: "napi_expired" } });
  fireEvent.click(button("instance.neon.connect.find"));
  expect(
    await screen.findByText("instance.neon.connect.error.key-invalid")
  ).toBeVisible();
  expect(key).toHaveValue("napi_expired");
  expect(screen.queryByText("Server wording")).toBeNull();
});

test("keeps a Neon project ID that lists nothing on the Connect step, with the error on its field", async () => {
  mocks.discover.mockResolvedValue(
    page([], "", [
      { projectId: "wrong-project-12345678", reason: "VENDOR_RESOURCE_NOT_FOUND" },
    ])
  );
  mount();
  fireEvent.change(screen.getByLabelText("instance.neon.connect.api-key"), {
    target: { value: "napi_example" },
  });
  const projectId = screen.getByLabelText(/instance.neon.connect.project-id/);
  fireEvent.change(projectId, { target: { value: "wrong-project-12345678" } });
  fireEvent.click(button("instance.neon.connect.find"));
  await waitFor(() =>
    expect(projectId).toHaveAccessibleDescription(
      "instance.neon.connect.error.project-id-unreadable"
    )
  );
  expect(
    screen.queryByRole("heading", { name: "instance.neon.steps.select" })
  ).toBeNull();
  expect(projectId).toHaveValue("wrong-project-12345678");
  // Correcting the field clears the error and lets discovery run again.
  fireEvent.change(projectId, { target: { value: "autumn-disk-48433112" } });
  expect(projectId).not.toHaveAccessibleDescription(
    "instance.neon.connect.error.project-id-unreadable"
  );
  mocks.discover.mockResolvedValue(page(branches(1)));
  fireEvent.click(button("instance.neon.connect.find"));
  await screen.findByRole("heading", { name: "instance.neon.steps.select" });
});

test("does not start discovery when the workspace cannot add an instance", () => {
  Object.assign(mocks.license, { instanceCountLimit: 10, totalInstanceCount: 10 });
  mount();
  expect(button("instance.neon.connect.find")).toBeDisabled();
});

test("disables Find branches until a rate limit's retry delay passes", async () => {
  mocks.discover.mockRejectedValue(
    new ConnectError("Server wording", Code.ResourceExhausted, undefined, [
      {
        desc: ErrorInfoSchema,
        value: create(ErrorInfoSchema, { reason: "VENDOR_RATE_LIMITED" }),
      },
      {
        desc: RetryInfoSchema,
        value: create(RetryInfoSchema, { retryDelay: { seconds: 25n } }),
      },
    ])
  );
  mount();
  fireEvent.change(screen.getByLabelText("instance.neon.connect.api-key"), {
    target: { value: "napi_example" },
  });
  fireEvent.click(button("instance.neon.connect.find"));
  expect(
    await screen.findByText(/connect.error.rate-limited-wait.*\d\d:\d\d:\d\d/)
  ).toBeVisible();
  expect(button(/connect.retry-in.*0:2\d/)).toBeDisabled();
});

test("imports 45 selections with one confirmation as batches of 20, 20, and 5", async () => {
  mocks.discover.mockResolvedValue(page(branches(45)));
  mocks.batch.mockImplementation(async (request: BatchRequest) =>
    created(request)
  );
  mount({ parent: "projects/checkout" });
  await connect();
  expect(mocks.discover).toHaveBeenCalledWith(
    expect.objectContaining({
      parent: "projects/checkout",
      apiKey: "napi_example",
      projectId: "",
      pageToken: "",
    }),
    expect.anything()
  );
  for (const checkbox of screen.getAllByRole("checkbox")) {
    expect(checkbox).not.toBeChecked();
  }
  selectAll();
  await startImport(45);
  expect(
    await screen.findByText('instance.neon.import.title.done{"count":45}')
  ).toBeVisible();
  expect(mocks.batch.mock.calls.map(([request]) => request.imports.length)).toEqual(
    [20, 20, 5]
  );
  const first = mocks.batch.mock.calls[0][0];
  expect(first).toMatchObject({
    parent: "projects/checkout",
    credential: { case: "neonCredential", value: { apiKey: "napi_example" } },
  });
  expect(first.imports[0]).not.toHaveProperty("activation");
  expect(first.imports[0]).toMatchObject({
    instanceId: "neon-storefront-preview-0",
    username: "app_owner",
    source: {
      case: "neonBranch",
      value: {
        projectId: "autumn-disk-48433112",
        branchId: "br-autumn-disk-48433112-0",
      },
    },
    instance: { syncInterval: { seconds: 3600n } },
  });
  expect(mocks.refresh).toHaveBeenCalled();
  // Nothing is left to lose, so leaving no longer warns and setup just closes.
  await waitFor(() => expect(onDirtyChange).toHaveBeenLastCalledWith(false));
  expect(vi.mocked(useUnsavedChangesGuard)).toHaveBeenLastCalledWith(false);
  fireEvent.click(button("instance.neon.import.close"));
  await waitFor(() => expect(onDismiss).toHaveBeenCalledOnce());
});

test("shows the destination it was opened from and offers no control to change it", async () => {
  mocks.discover.mockResolvedValue(page(branches(1)));
  mount({
    parent: "projects/checkout",
    project: { title: "Checkout Service" } as Project,
  });
  await connect();
  expect(
    screen.getByText('instance.neon.destination.project{"project":"Checkout Service"}')
  ).toBeVisible();
  expect(screen.queryByRole("combobox", { name: /destination/i })).toBeNull();
});

test("retries a failed page with the same token and keeps rows and selections", async () => {
  mocks.discover
    .mockResolvedValueOnce(page(branches(2), "token-1"))
    .mockRejectedValueOnce(reasonError(Code.Unavailable, "VENDOR_UNAVAILABLE"))
    .mockResolvedValueOnce(page(branches(2, 2)));
  mount();
  await connect();
  selectAll();
  fireEvent.click(button(/select.load-more/));
  expect(
    await screen.findByText(/page-error-unavailable/)
  ).toBeVisible();
  expect(screen.getAllByRole("checkbox", { name: /row-select-label/ })).toHaveLength(2);
  expect(
    screen.getByRole("checkbox", { name: /row-select-label.*preview-0/ })
  ).toBeChecked();
  fireEvent.click(button("instance.neon.select.retry-page"));
  await waitFor(() =>
    expect(
      screen.getAllByRole("checkbox", { name: /row-select-label/ })
    ).toHaveLength(4)
  );
  expect(mocks.discover.mock.calls.map(([request]) => request.pageToken)).toEqual(
    ["", "token-1", "token-1"]
  );
  // Loading a page never changes the selection.
  expect(
    screen.getByRole("checkbox", { name: /row-select-label.*preview-0/ })
  ).toBeChecked();
  expect(
    screen.getByRole("checkbox", { name: /row-select-label.*preview-3/ })
  ).not.toBeChecked();
  expect(screen.getByText(/status.complete.*"count":4/)).toBeVisible();
});

test("retries a failed Neon project as its own discovery and merges its branches", async () => {
  const failedProject = "misty-credit-15529086";
  mocks.discover
    .mockResolvedValueOnce(
      page(branches(2), "token-1", [
        { projectId: failedProject, reason: "VENDOR_ACCESS_DENIED" },
      ])
    )
    .mockResolvedValueOnce(page([branch(0, failedProject)], "project-token"))
    .mockResolvedValueOnce(page([branch(1, failedProject)]))
    .mockResolvedValueOnce(page(branches(1, 2)));
  mount();
  await connect();
  expect(screen.getByText(/project-failed.*misty-credit-15529086/)).toBeVisible();
  fireEvent.click(button(/select.retry-project.*misty-credit-15529086/));
  await waitFor(() =>
    expect(screen.queryByText(/select.project-failed/)).toBeNull()
  );
  expect(screen.getAllByRole("checkbox", { name: /row-select-label/ })).toHaveLength(4);
  fireEvent.click(button(/select.load-more/));
  await waitFor(() => expect(mocks.discover).toHaveBeenCalledTimes(4));
  expect(
    mocks.discover.mock.calls.map(([request]) => [
      request.projectId,
      request.pageToken,
    ])
  ).toEqual([
    ["", ""],
    [failedProject, ""],
    [failedProject, "project-token"],
    ["", "token-1"],
  ]);
});

test("restarting discovery keeps the selection of branches returned again", async () => {
  mocks.discover
    .mockResolvedValueOnce(page(branches(3)))
    .mockResolvedValueOnce(page(branches(2, 1)));
  mount();
  await connect();
  fireEvent.click(
    screen.getByRole("checkbox", { name: /row-select-label.*preview-0/ })
  );
  fireEvent.click(
    screen.getByRole("checkbox", { name: /row-select-label.*preview-1/ })
  );
  fireEvent.click(button("instance.neon.select.restart"));
  await waitFor(() =>
    expect(
      screen.getAllByRole("checkbox", { name: /row-select-label/ })
    ).toHaveLength(2)
  );
  expect(
    screen.getByRole("checkbox", { name: /row-select-label.*preview-1/ })
  ).toBeChecked();
  expect(
    screen.getByRole("checkbox", { name: /row-select-label.*preview-2/ })
  ).not.toBeChecked();
  expect(button(/select.configure.*"count":1/)).toBeEnabled();
});

test("blocks a selection above what the workspace can still add", async () => {
  Object.assign(mocks.license, { instanceCountLimit: 5, totalInstanceCount: 3 });
  mocks.discover.mockResolvedValue(page(branches(3)));
  mount();
  await connect();
  selectAll();
  expect(button(/select.configure.*"count":3/)).toBeDisabled();
  expect(screen.getByText(/select.footer-over/)).toBeVisible();
  fireEvent.click(
    screen.getByRole("checkbox", { name: /row-select-label.*preview-2/ })
  );
  expect(button(/select.configure.*"count":2/)).toBeEnabled();
});

test("reads the remaining instance number again when Select and Configure open and before import", async () => {
  Object.assign(mocks.license, { instanceCountLimit: 5, totalInstanceCount: 0 });
  mocks.discover.mockResolvedValue(page(branches(3)));
  mount();
  await connect();
  await waitFor(() => expect(mocks.refresh).toHaveBeenCalledTimes(1));
  selectAll();
  fireEvent.click(button(/select.configure.*"count":3/));
  await waitFor(() => expect(mocks.refresh).toHaveBeenCalledTimes(2));

  // Instances added elsewhere leave room for two; the import must not start.
  mocks.refresh.mockImplementation(async () => {
    mocks.license.totalInstanceCount = 3;
    return undefined;
  });
  fireEvent.click(button(/neon.import-count.*"count":3/));
  await waitFor(() => expect(mocks.refresh).toHaveBeenCalledTimes(3));
  expect(await screen.findByText(/configure.footer-over/)).toBeVisible();
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(button(/neon.import-count.*"count":3/)).toBeDisabled();
  expect(mocks.batch).not.toHaveBeenCalled();
});

test("shows no instance limit number under an unlimited license", async () => {
  Object.assign(mocks.license, {
    instanceCountLimit: Number.MAX_VALUE,
    instanceLicenseCount: Number.MAX_VALUE,
  });
  mocks.discover.mockResolvedValue(page(branches(1)));
  mount();
  await connect();
  expect(screen.queryByText(/footer-limit/)).toBeNull();
});

test("never retries an unknown outcome on its own: check, confirm, then retry with the same ID", async () => {
  mocks.discover.mockResolvedValue(page(branches(21)));
  mocks.batch
    .mockImplementationOnce(async (request: BatchRequest) =>
      outcome(
        request,
        3,
        "INSTANCE_CREATION_FAILED",
        InstanceImportFailure_CreationOutcome.UNKNOWN
      )
    )
    .mockImplementation(async (request: BatchRequest) => created(request));
  mocks.getInstance.mockRejectedValue(
    new ConnectError("not found", Code.NotFound)
  );
  mount();
  await connect();
  selectAll();
  await startImport(21);
  expect(
    await screen.findByText("instance.neon.import.title.paused")
  ).toBeVisible();
  expect(mocks.batch).toHaveBeenCalledTimes(1);
  expect(
    screen.queryByRole("button", { name: "instance.neon.import.unknown.retry" })
  ).toBeNull();

  // The unsent batch can continue while the unknown entry stays excluded.
  fireEvent.click(button(/unsent.continue.*"count":1/));
  await waitFor(() => expect(mocks.batch).toHaveBeenCalledTimes(2));
  expect(mocks.batch.mock.calls[1][0].imports).toEqual([
    expect.objectContaining({ instanceId: "neon-storefront-preview-20" }),
  ]);

  fireEvent.click(button("instance.neon.import.unknown.check"));
  await waitFor(() =>
    expect(mocks.getInstance).toHaveBeenCalledWith(
      { name: "instances/neon-storefront-preview-3" },
      expect.anything()
    )
  );
  fireEvent.click(await screen.findByRole("button", {
    name: "instance.neon.import.unknown.retry",
  }));
  expect(mocks.batch).toHaveBeenCalledTimes(2);
  const confirm = await screen.findByRole("alertdialog");
  expect(
    within(confirm).getByText(/retry-unknown.body.*neon-storefront-preview-3/)
  ).toBeVisible();
  fireEvent.click(
    within(confirm).getByRole("button", {
      name: "instance.neon.import.unknown.retry",
    })
  );
  await waitFor(() => expect(mocks.batch).toHaveBeenCalledTimes(3));
  expect(mocks.batch.mock.calls[2][0].imports).toEqual([
    expect.objectContaining({ instanceId: "neon-storefront-preview-3" }),
  ]);
  expect(
    await screen.findByText('instance.neon.import.title.done{"count":21}')
  ).toBeVisible();
});

test.each([
  [Code.PermissionDenied, "instance.neon.reason.permission-denied"],
  [Code.Unauthenticated, "instance.neon.reason.session-expired"],
  [Code.NotFound, "instance.neon.reason.destination-not-found"],
  [Code.InvalidArgument, "instance.neon.reason.batch-rejected"],
] as const)(
  "a request rejected with code %s leaves its entries not created, with nothing to check",
  async (code, reason) => {
    mocks.discover.mockResolvedValue(page(branches(21)));
    mocks.batch.mockRejectedValue(new ConnectError("Server wording", code));
    mount();
    await connect();
    selectAll();
    await startImport(21);
    expect(
      await screen.findByText(
        "instance.neon.import.banner.rejected-title"
      )
    ).toBeVisible();
    expect(screen.getAllByText(reason)).toHaveLength(20);
    expect(
      screen.queryByRole("button", { name: "instance.neon.import.unknown.check" })
    ).toBeNull();
    expect(button(/failed.retry.*"count":20/)).toBeEnabled();
    expect(button(/unsent.continue.*"count":1/)).toBeEnabled();
    expect(mocks.batch).toHaveBeenCalledTimes(1);
  }
);

test("checks an instance ID edited on the Import step against the other entries", async () => {
  mocks.discover.mockResolvedValue(page(branches(2)));
  mocks.batch.mockImplementationOnce(async (request: BatchRequest) =>
    outcome(
      request,
      1,
      "INSTANCE_SETTINGS_INVALID",
      InstanceImportFailure_CreationOutcome.NOT_CREATED
    )
  );
  mount();
  await connect();
  selectAll();
  await startImport(2);
  fireEvent.click(
    await screen.findByRole("button", { name: "instance.neon.edit-settings" })
  );
  const sheet = await screen.findByRole("dialog");
  const instanceId = within(sheet).getByLabelText(
    "instance.neon.sheet.instance-id"
  );
  fireEvent.change(instanceId, {
    target: { value: "neon-storefront-preview-0" },
  });
  await waitFor(() =>
    expect(instanceId).toHaveAccessibleDescription(
      "instance.neon.sheet.id-duplicate"
    )
  );
  // The open sheet hides the page behind it from the accessibility tree.
  const behind = (name: string | RegExp) =>
    screen.getByRole("button", { name, hidden: true });
  expect(behind("instance.neon.import.failed.retry-none")).toBeDisabled();
  fireEvent.change(instanceId, {
    target: { value: "neon-storefront-preview-1-b" },
  });
  await waitFor(() =>
    expect(behind(/failed.retry.*"count":1/)).toBeEnabled()
  );
});

test("sends the default interval where the plan fixes it, and a schedule that is off where instances are not activated", async () => {
  const importOne = async () => {
    mocks.discover.mockResolvedValue(page(branches(1)));
    mocks.batch.mockImplementation(async (request: BatchRequest) =>
      created(request)
    );
    const view = mount();
    await connect();
    selectAll();
    await startImport(1);
    await screen.findByText('instance.neon.import.title.done{"count":1}');
    view.unmount();
    return mocks.batch.mock.calls.at(-1)?.[0].imports[0];
  };
  mocks.intervalFeature = false;
  expect((await importOne()).instance.syncInterval).toEqual({ seconds: 3600n });
  Object.assign(mocks.license, { instanceLicenseCount: 0 });
  expect((await importOne()).instance.syncInterval).toEqual({ seconds: 0n });
});

test("an unknown entry found in the instance list resolves as created", async () => {
  mocks.discover.mockResolvedValue(page(branches(1)));
  mocks.batch.mockRejectedValue(new Error("connection lost"));
  mocks.getInstance.mockResolvedValue(
    create(InstanceSchema, { name: "instances/neon-storefront-preview-0" })
  );
  mount();
  await connect();
  selectAll();
  await startImport(1);
  expect(
    await screen.findByText("instance.neon.import.banner.paused-lost-title")
  ).toBeVisible();
  fireEvent.click(button("instance.neon.import.unknown.check"));
  expect(
    await screen.findByText("instance.neon.import.created.by-check")
  ).toBeVisible();
  expect(mocks.batch).toHaveBeenCalledTimes(1);
});

test("corrects a missing password in place and retries explicitly", async () => {
  mocks.discover.mockResolvedValue(page(branches(2)));
  mocks.batch
    .mockImplementationOnce(async (request: BatchRequest) =>
      outcome(
        request,
        1,
        "VENDOR_PASSWORD_REQUIRED",
        InstanceImportFailure_CreationOutcome.NOT_CREATED
      )
    )
    .mockImplementation(async (request: BatchRequest) => created(request));
  mount();
  await connect();
  selectAll();
  await startImport(2);
  const password = await screen.findByLabelText(/password-for.*app_owner/);
  expect(mocks.batch).toHaveBeenCalledTimes(1);
  fireEvent.change(password, { target: { value: "example-password" } });
  fireEvent.click(button(/failed.retry.*"count":1/));
  await waitFor(() => expect(mocks.batch).toHaveBeenCalledTimes(2));
  expect(mocks.batch.mock.calls[1][0].imports).toEqual([
    expect.objectContaining({
      instanceId: "neon-storefront-preview-1",
      password: "example-password",
    }),
  ]);
});

test("stopping marks the active batch unknown and sends nothing more", async () => {
  mocks.discover.mockResolvedValue(page(branches(21)));
  mocks.batch.mockImplementation(
    (_request: BatchRequest, options: { signal: AbortSignal }) =>
      new Promise((_resolve, reject) =>
        options.signal.addEventListener("abort", () =>
          reject(new ConnectError("canceled", Code.Canceled))
        )
      )
  );
  mount();
  await connect();
  selectAll();
  await startImport(21);
  fireEvent.click(await screen.findByRole("button", {
    name: "instance.neon.import.stop",
  }));
  const confirm = await screen.findByRole("alertdialog");
  expect(within(confirm).getByText(/stop.awaiting.*"count":20/)).toBeVisible();
  expect(within(confirm).getByText(/stop.unsent.*"count":1/)).toBeVisible();
  fireEvent.click(
    within(confirm).getByRole("button", { name: "instance.neon.import.stop" })
  );
  expect(
    await screen.findByText("instance.neon.import.title.stopped")
  ).toBeVisible();
  expect(mocks.batch).toHaveBeenCalledTimes(1);
  expect(screen.getAllByText("instance.neon.import.unknown.stopped")).toHaveLength(20);

  // Closing with unresolved entries warns before discarding the key and list.
  fireEvent.click(button("instance.neon.import.close"));
  const close = await screen.findByRole("alertdialog");
  expect(within(close).getByText(/close.body.*"count":21/)).toBeVisible();
  expect(onDismiss).not.toHaveBeenCalled();
  fireEvent.click(
    within(close).getByRole("button", { name: "instance.neon.import.close" })
  );
  await waitFor(() => expect(onDismiss).toHaveBeenCalledOnce());
});

test("closes the stop confirmation when the last batch returns first", async () => {
  mocks.discover.mockResolvedValue(page(branches(1)));
  let finish = () => {};
  mocks.batch.mockImplementation(
    (request: BatchRequest) =>
      new Promise((resolve) => {
        finish = () => resolve(created(request));
      })
  );
  mount();
  await connect();
  selectAll();
  await startImport(1);
  fireEvent.click(await screen.findByRole("button", {
    name: "instance.neon.import.stop",
  }));
  expect(await screen.findByRole("alertdialog")).toBeVisible();
  finish();
  expect(
    await screen.findByText('instance.neon.import.title.done{"count":1}')
  ).toBeVisible();
  await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
  expect(screen.queryByText("instance.neon.import.title.stopped")).toBeNull();
});

test("Back preserves selections and settings", async () => {
  mocks.discover.mockResolvedValue(page(branches(2)));
  mount();
  await connect();
  selectAll();
  fireEvent.click(button(/select.configure.*"count":2/));
  fireEvent.click(button("common.back"));
  fireEvent.click(button("common.back"));
  expect(screen.getByLabelText("instance.neon.connect.api-key")).toHaveValue(
    "napi_example"
  );
  fireEvent.click(button("instance.neon.connect.find"));
  expect(mocks.discover).toHaveBeenCalledTimes(1);
  expect(button(/select.configure.*"count":2/)).toBeEnabled();
  // Cancelling with selections asks first.
  fireEvent.click(button("common.back"));
  fireEvent.click(button("common.cancel"));
  expect(
    await screen.findByText("instance.neon.leave.body")
  ).toBeVisible();
  expect(onDismiss).not.toHaveBeenCalled();
});
