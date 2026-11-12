import { ErrorInfoSchema } from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import { create } from "@bufbuild/protobuf";
import { anyPack } from "@bufbuild/protobuf/wkt";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { NeonBranchSchema } from "@/types/proto-es/v1/instance_service_pb";
import {
  type NeonDiscoveryStatus,
  type NeonProjectFailure,
  NeonSelectStep,
} from "./NeonSelectStep";
import { branchKey, makeDraft, type NeonImportDraft } from "./neon-import";

const t = (key: string, values?: Record<string, unknown>) =>
  values ? `${key}${JSON.stringify(values)}` : key;
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t }) }));

afterEach(cleanup);

const branch = (projectTitle: string, title: string, broken = false) =>
  create(NeonBranchSchema, {
    projectId: `${projectTitle}-id`,
    projectTitle,
    branchId: `br-${projectTitle}-${title}`,
    branchTitle: title,
    databases: [{ name: "app", owner: "app_owner" }],
    roles: ["app_owner"],
    discoveryError: broken
      ? {
          code: 9,
          message: "Server wording",
          details: [
            anyPack(
              ErrorInfoSchema,
              create(ErrorInfoSchema, { reason: "VENDOR_ENDPOINT_UNAVAILABLE" })
            ),
          ],
        }
      : undefined,
  });

const rows = (): NeonImportDraft[] => [
  makeDraft(branch("storefront", "main")),
  makeDraft(branch("storefront", "staging")),
  makeDraft(branch("storefront", "broken", true)),
  makeDraft(branch("billing", "main")),
];

const status = (change: Partial<NeonDiscoveryStatus> = {}): NeonDiscoveryStatus => ({
  started: true,
  loading: false,
  complete: false,
  retryInSeconds: 0,
  ...change,
});

function renderStep(props: Partial<Parameters<typeof NeonSelectStep>[0]> = {}) {
  const handlers = {
    onSelect: vi.fn(),
    onLoadMore: vi.fn(),
    onLoadAll: vi.fn(),
    onRetryProject: vi.fn(),
    onRestart: vi.fn(),
  };
  render(
    <NeonSelectStep
      drafts={rows()}
      status={status()}
      failures={[]}
      {...handlers}
      {...props}
    />
  );
  return handlers;
}

const keys = (drafts: NeonImportDraft[]) =>
  drafts.map((draft) => branchKey(draft.branch));

test("groups branches by Neon project and starts with nothing selected", () => {
  renderStep();
  expect(screen.getByText("storefront")).toBeVisible();
  expect(screen.getByText("storefront-id")).toBeInTheDocument();
  expect(screen.getByText("billing")).toBeVisible();
  for (const checkbox of screen.getAllByRole("checkbox")) {
    expect(checkbox).not.toBeChecked();
  }
});

test("keeps an unavailable branch visible with its reason and unselectable", () => {
  renderStep();
  expect(
    screen.getByRole("checkbox", {
      name: /row-select-label.*storefront \/ broken/,
    })
  ).toHaveAttribute("aria-disabled", "true");
  // Once in the Status column and once under the branch name, which is where
  // a narrow screen shows it.
  const reasons = screen.getAllByText(
    /instance.neon.reason.endpoint-unavailable/
  );
  expect(reasons).toHaveLength(2);
  expect(
    reasons.some((reason) => reason.closest("td")?.querySelector("label"))
  ).toBe(true);
  expect(screen.queryByText("Server wording")).toBeNull();
});

test("selects a branch from its name", () => {
  const drafts = rows();
  const { onSelect } = renderStep({ drafts });
  fireEvent.click(screen.getByText("staging"));
  expect(onSelect).toHaveBeenCalledWith(keys([drafts[1]]), true);
});

test("states no total and locks the filter until every page is loaded", () => {
  renderStep();
  expect(
    screen.getByText(/instance.neon.select.status.partial.*"count":4/)
  ).toBeVisible();
  expect(screen.queryByText(/status.complete/)).toBeNull();
  expect(
    screen.getByLabelText("instance.neon.select.filter-label")
  ).toBeDisabled();
  expect(screen.getByText("instance.neon.select.filter-locked")).toBeVisible();
});

test("filters by name once everything is loaded", async () => {
  renderStep({ status: status({ complete: true }) });
  expect(screen.getByText(/status.complete.*"count":4/)).toBeVisible();
  const filter = screen.getByLabelText("instance.neon.select.filter-label");
  expect(filter).toBeEnabled();
  fireEvent.change(filter, { target: { value: "billing" } });
  await waitFor(() => expect(screen.queryByText("storefront")).toBeNull());
  expect(screen.getByText("billing")).toBeVisible();
});

test("select-all acts on loaded, available branches only", () => {
  const drafts = rows();
  const { onSelect } = renderStep({ drafts });
  fireEvent.click(
    screen.getByRole("checkbox", {
      name: 'instance.neon.select.select-all{"count":3}',
    })
  );
  expect(onSelect).toHaveBeenCalledWith(
    keys([drafts[0], drafts[1], drafts[3]]),
    true
  );
  fireEvent.click(
    screen.getByRole("checkbox", { name: /group-select-label.*storefront/ })
  );
  expect(onSelect).toHaveBeenLastCalledWith(
    keys([drafts[0], drafts[1]]),
    true
  );
});

test("switches between loaded and selected rows", () => {
  const drafts = rows();
  drafts[3] = { ...drafts[3], selected: true };
  renderStep({ drafts });
  fireEvent.click(
    screen.getByRole("radio", { name: /view-selected.*"count":1/ })
  );
  expect(screen.queryByText("storefront")).toBeNull();
  expect(screen.getByText("billing")).toBeVisible();
});

test("the selected view counts what is selected, not every available branch", () => {
  const drafts = rows();
  drafts[0] = { ...drafts[0], selected: true };
  const { onSelect } = renderStep({ drafts });
  fireEvent.click(
    screen.getByRole("radio", { name: /view-selected.*"count":1/ })
  );
  expect(
    screen.getByText('instance.neon.select.selected-shown{"count":1}')
  ).toBeVisible();
  expect(screen.queryByText(/select.all-selected/)).toBeNull();
  // The Neon project still reports all three of its loaded branches.
  expect(
    screen.getByText(
      /group-summary\{"selected":1,"count":3\}.*group-shown\{"count":1\}/
    )
  ).toBeVisible();
  fireEvent.click(
    screen.getByRole("checkbox", {
      name: "instance.neon.select.deselect-shown",
    })
  );
  expect(onSelect).toHaveBeenCalledWith(keys([drafts[0]]), false);
});

test("a filter keeps each Neon project's loaded count and labels its checkbox as acting on shown rows", async () => {
  const drafts = rows();
  const { onSelect } = renderStep({
    drafts,
    status: status({ complete: true }),
  });
  fireEvent.change(screen.getByLabelText("instance.neon.select.filter-label"), {
    target: { value: "staging" },
  });
  await waitFor(() => expect(screen.queryByText("billing")).toBeNull());
  expect(
    screen.getByText(
      /group-summary\{"selected":0,"count":3\}.*group-shown\{"count":1\}/
    )
  ).toBeVisible();
  fireEvent.click(
    screen.getByRole("checkbox", {
      name: /group-select-shown-label.*storefront/,
    })
  );
  expect(onSelect).toHaveBeenCalledWith(keys([drafts[1]]), true);
});

test("loads one page or all remaining on demand", () => {
  const { onLoadMore, onLoadAll } = renderStep();
  fireEvent.click(
    screen.getByRole("button", { name: "instance.neon.select.load-more" })
  );
  fireEvent.click(
    screen.getByRole("button", { name: "instance.neon.select.load-all" })
  );
  expect(onLoadMore).toHaveBeenCalledOnce();
  expect(onLoadAll).toHaveBeenCalledOnce();
});

test("a failed page keeps rows and selections and retries in place", () => {
  const drafts = rows();
  drafts[0] = { ...drafts[0], selected: true };
  const { onLoadMore } = renderStep({
    drafts,
    status: status({
      pageFailure: { kind: "rate-limited", retryAfterSeconds: 12, message: "" },
      retryInSeconds: 12,
      retryAt: new Date(2030, 0, 1, 14, 32, 10).getTime(),
    }),
  });
  expect(
    screen.getByRole("checkbox", { name: /row-select-label.*storefront \/ main/ })
  ).toBeChecked();
  expect(
    screen.getByText(/page-error-retry-in.*14:32:10.*page-error-kept/)
  ).toBeVisible();
  expect(screen.queryByRole("button", { name: /load-more/ })).toBeNull();
  const retry = screen.getByRole("button", {
    name: "instance.neon.select.retry-page",
  });
  expect(retry).toBeDisabled();
  cleanup();
  const again = renderStep({
    drafts,
    status: status({
      pageFailure: { kind: "unavailable", retryAfterSeconds: 0, message: "" },
    }),
  });
  fireEvent.click(
    screen.getByRole("button", { name: "instance.neon.select.retry-page" })
  );
  expect(again.onLoadMore).toHaveBeenCalledOnce();
  expect(onLoadMore).not.toHaveBeenCalled();
});

test("lists a Neon project that could not be enumerated as its own row with retry", () => {
  const failures: NeonProjectFailure[] = [
    {
      projectId: "misty-credit-15529086",
      reason: "VENDOR_ACCESS_DENIED",
      message: "Server wording",
      retrying: false,
    },
  ];
  const { onRetryProject } = renderStep({ failures });
  const row = screen
    .getByText(/project-failed.*misty-credit-15529086/)
    .closest("tr") as HTMLElement;
  expect(
    within(row).getByText(
      /instance.neon.reason.access-denied.*project-failed-help/
    )
  ).toBeVisible();
  fireEvent.click(within(row).getByRole("button"));
  expect(onRetryProject).toHaveBeenCalledWith("misty-credit-15529086");
  expect(
    screen.getByText(/status.failed-projects.*"count":1/)
  ).toBeVisible();
});

test("does not call an incomplete listing empty", () => {
  renderStep({
    drafts: [],
    status: status({ complete: true }),
    failures: [
      {
        projectId: "misty-credit-15529086",
        reason: "VENDOR_UNAVAILABLE",
        message: "",
        retrying: false,
      },
    ],
  });
  expect(screen.getByText(/project-failed-help/)).toBeVisible();
  expect(screen.queryByText("instance.neon.select.empty")).toBeNull();
});

test("explains a selection above the instance limit", () => {
  const drafts = rows().map((draft) => ({ ...draft, selected: true }));
  renderStep({ drafts, capacity: 3 });
  expect(screen.getByText(/over-limit-title/)).toBeVisible();
  expect(screen.getByText(/over-limit-body/)).toBeVisible();
});

test("says the limit is reached, not which branches to deselect, when no instance can be added", () => {
  const drafts = rows();
  drafts[0] = { ...drafts[0], selected: true };
  renderStep({ drafts, capacity: 0 });
  expect(screen.getByText("instance.neon.limit-reached.title")).toBeVisible();
  expect(screen.getByText("instance.neon.limit-reached.body")).toBeVisible();
  expect(screen.queryByText(/over-limit-body/)).toBeNull();
});

test("restarts discovery on request", () => {
  const { onRestart } = renderStep();
  fireEvent.click(
    screen.getByRole("button", { name: "instance.neon.select.restart" })
  );
  expect(onRestart).toHaveBeenCalledOnce();
});
