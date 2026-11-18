import { create } from "@bufbuild/protobuf";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  InstanceSchema,
  NeonBranchSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { NeonImportStep } from "./NeonImportStep";
import {
  createImportRun,
  defaultSharedSettings,
  duplicateInstanceIds,
  makeDraft,
  type NeonImportEntry,
  type NeonImportRun,
  type NeonScheduleMode,
  updateEntryDraft,
} from "./neon-import";

const t = (key: string, values?: Record<string, unknown>) =>
  values ? `${key}${JSON.stringify(values)}` : key;
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t }),
}));
vi.mock("@/hooks/useAppState", () => ({
  useEnvironmentList: () => [{ id: "prod", title: "Prod" }],
}));

const handlers = {
  onCheck: vi.fn(),
  onRetryUnknown: vi.fn(),
  onRetryFailed: vi.fn(),
  onContinue: vi.fn(),
  onEditEntry: vi.fn(),
};
beforeEach(() => {
  for (const handler of Object.values(handlers)) handler.mockClear();
});
afterEach(cleanup);

// 25 entries in two batches, each state set by the test.
const fixture = (
  states: Partial<Record<number, Partial<NeonImportEntry>>>,
  rest: Partial<NeonImportEntry> = { state: "created" }
): NeonImportRun => {
  const base = createImportRun(
    Array.from({ length: 25 }, (_, index) => ({
      ...makeDraft(
        create(NeonBranchSchema, {
          projectId: "project",
          projectTitle: "storefront",
          branchId: `br-${index}`,
          branchTitle: `preview-${index}`,
          databases: [{ name: "app", owner: "app_owner" }],
          roles: ["app_owner", "migrator"],
        })
      ),
      selected: true,
    })),
    defaultSharedSettings()
  );
  return {
    batches: base.batches.map((batch) => ({
      ...batch,
      sentAt: new Date("2026-11-10T06:02:07Z"),
    })),
    entries: base.entries.map((entry, index) => {
      const change = states[index] ?? rest;
      return {
        ...entry,
        ...change,
        attempted: {
          role: entry.draft.role,
          database: entry.draft.database,
          instanceId: entry.draft.instanceId,
        },
        instance:
          change.state === "created"
            ? create(InstanceSchema, {
                name: `instances/${entry.draft.instanceId}`,
              })
            : undefined,
      };
    }),
  };
};

let latest: NeonImportRun;
function Harness({
  initial,
  stopped = false,
  schedule = "editable",
}: {
  initial: NeonImportRun;
  stopped?: boolean;
  schedule?: NeonScheduleMode;
}) {
  const [run, setRun] = useState(initial);
  latest = run;
  return (
    <NeonImportStep
      run={run}
      stopped={stopped}
      checking={false}
      schedule={schedule}
      duplicateIds={duplicateInstanceIds(
        run.entries.map((entry) => entry.draft)
      )}
      destination="Workspace"
      instancesHref="/instances"
      onEntryChange={(key, update) =>
        setRun((value) => updateEntryDraft(value, key, update))
      }
      {...handlers}
    />
  );
}

const failed = (reason: string): Partial<NeonImportEntry> => ({
  state: "failed",
  reason,
  message: "Server wording",
});

test("shows batch progress and one shared state for the active batch", () => {
  const run = fixture(
    Object.fromEntries(
      Array.from({ length: 5 }, (_, index) => [20 + index, { state: "awaiting" }])
    )
  );
  render(<Harness initial={run} />);
  expect(
    screen.getByText('instance.neon.import.progress{"processed":20,"total":25}')
  ).toBeVisible();
  const progress = screen.getByRole("region", {
    name: "instance.neon.import.progress-label",
  });
  expect(
    within(progress).getByText(/batch-label.*"index":2.*batch.awaiting/)
  ).toBeVisible();
  expect(
    within(progress).getByText(/progress-batch.*"index":2.*"count":5/)
  ).toBeVisible();
  expect(screen.getByText(/awaiting.description.*"count":5/)).toBeVisible();
  // The entries of the active batch are named together, with no per-entry state.
  expect(
    screen.getByText(/storefront \/ preview-20 · storefront \/ preview-21/)
  ).toBeVisible();
  expect(screen.getByText("instance.neon.import.banner.running")).toBeVisible();
});

test("orders the groups unknown, not created, awaiting, not sent, created", () => {
  const run = fixture({
    0: { state: "unknown", unknownCause: "reported" },
    1: failed("VENDOR_PASSWORD_REQUIRED"),
    2: { state: "awaiting", retrying: true },
    20: { state: "unsent" },
  });
  render(<Harness initial={run} />);
  const headings = screen
    .getAllByRole("heading", { level: 2 })
    .map((heading) => heading.parentElement?.textContent ?? "");
  expect(headings.map((text) => text.match(/state\.([a-z]+?)instance/)?.[1])).toEqual([
    "unknown",
    "failed",
    "awaiting",
    "unsent",
    "created",
  ]);
});

test("unknown entries are checked against the instance list before any retry", () => {
  const checkedAt = new Date("2026-11-10T06:05:00Z");
  const run = fixture({
    0: { state: "unknown", unknownCause: "lost" },
    1: { state: "unknown", unknownCause: "lost", checkedAt },
    20: { state: "unsent" },
  });
  render(<Harness initial={run} />);
  fireEvent.click(
    screen.getByRole("button", {
      name: "instance.neon.import.unknown.check-again",
    })
  );
  expect(handlers.onCheck).toHaveBeenCalledOnce();
  const retries = screen.getAllByRole("button", {
    name: "instance.neon.import.unknown.retry",
  });
  expect(retries).toHaveLength(1);
  expect(
    screen.getByText('instance.neon.import.unknown.checked{"time":"14:05:00"}')
  ).toBeVisible();
  fireEvent.click(retries[0]);
  expect(handlers.onRetryUnknown).toHaveBeenCalledWith(run.entries[1].key);
  // Unknown entries stay excluded, so the unsent batch can continue now.
  fireEvent.click(
    screen.getByRole("button", { name: /unsent.continue.*"count":1/ })
  );
  expect(handlers.onContinue).toHaveBeenCalledOnce();
});

test("a missing password is corrected in place and retried explicitly", () => {
  const run = fixture({ 3: failed("VENDOR_PASSWORD_REQUIRED") });
  render(<Harness initial={run} />);
  expect(
    screen.getByText(/failed.password-error.*app_owner/)
  ).toBeVisible();
  expect(screen.queryByText("Server wording")).toBeNull();
  const retry = screen.getByRole("button", {
    name: "instance.neon.import.failed.retry-none",
  });
  expect(retry).toBeDisabled();
  fireEvent.change(screen.getByLabelText(/password-for.*app_owner/), {
    target: { value: "example-password" },
  });
  expect(latest.entries[3].draft).toMatchObject({
    passwordSource: "typed",
    password: "example-password",
    instanceId: run.entries[3].draft.instanceId,
  });
  fireEvent.click(
    screen.getByRole("button", { name: /failed.retry.*"count":1/ })
  );
  expect(handlers.onRetryFailed).toHaveBeenCalledOnce();
});

test("a role that no longer exists is replaced in place", async () => {
  const run = fixture({ 3: failed("VENDOR_USER_NOT_FOUND") });
  render(<Harness initial={run} />);
  expect(screen.getByText(/failed.role-error.*app_owner/)).toBeVisible();
  fireEvent.click(screen.getByLabelText("instance.neon.sheet.role"));
  expect(screen.queryByRole("option", { name: "app_owner" })).toBeNull();
  const option = await screen.findByRole("option", { name: "migrator" });
  fireEvent.pointerDown(option, { pointerType: "mouse" });
  fireEvent.click(option);
  expect(latest.entries[3].draft.role).toBe("migrator");
  expect(
    screen.getByRole("button", { name: /failed.retry.*"count":1/ })
  ).toBeEnabled();
});

test("an instance ID that already exists is changed in place", () => {
  const run = fixture({ 3: failed("INSTANCE_ID_EXISTS") });
  render(<Harness initial={run} />);
  fireEvent.change(screen.getByLabelText("instance.neon.sheet.instance-id"), {
    target: { value: "neon-storefront-preview-3-b" },
  });
  expect(latest.entries[3].draft.instanceId).toBe("neon-storefront-preview-3-b");
  expect(
    screen.getByRole("button", { name: /failed.retry.*"count":1/ })
  ).toBeEnabled();
});

test("an edited instance ID that another entry uses is flagged and held back from the retry", () => {
  const run = fixture({
    3: failed("INSTANCE_ID_EXISTS"),
    4: failed("INSTANCE_ID_EXISTS"),
  });
  render(<Harness initial={run} />);
  const [first, second] = screen.getAllByLabelText(
    "instance.neon.sheet.instance-id"
  );
  fireEvent.change(first, { target: { value: "neon-storefront-renamed" } });
  expect(
    screen.getByRole("button", { name: /failed.retry.*"count":1/ })
  ).toBeEnabled();
  fireEvent.change(second, { target: { value: "neon-storefront-renamed" } });
  expect(first).toHaveAccessibleDescription(
    "instance.neon.sheet.id-duplicate"
  );
  expect(second).toHaveAccessibleDescription(
    "instance.neon.sheet.id-duplicate"
  );
  expect(screen.queryByText("instance.neon.import.failed.ready")).toBeNull();
  expect(
    screen.getByRole("button", { name: "instance.neon.import.failed.retry-none" })
  ).toBeDisabled();
  // An ID a created entry already holds collides the same way.
  fireEvent.change(second, { target: { value: "neon-storefront-preview-0" } });
  expect(second).toHaveAccessibleDescription(
    "instance.neon.sheet.id-duplicate"
  );
  expect(first).not.toHaveAccessibleDescription(
    "instance.neon.sheet.id-duplicate"
  );
});

test("entries rejected by the instance limit share one explanation and need no correction", () => {
  const run = fixture({
    22: failed("INSTANCE_LIMIT_EXCEEDED"),
    23: failed("INSTANCE_LIMIT_EXCEEDED"),
    24: failed("INSTANCE_LIMIT_EXCEEDED"),
  });
  render(<Harness initial={run} />);
  expect(
    screen.getAllByText(/failed.limit-title.*"count":3/)
  ).toHaveLength(1);
  expect(
    screen.getAllByText("instance.neon.import.failed.limit-error")
  ).toHaveLength(1);
  expect(
    screen.getByRole("button", { name: /failed.retry.*"count":3/ })
  ).toBeEnabled();
  expect(
    screen.getByText("instance.neon.import.banner.finished-title")
  ).toBeVisible();
});

test("another reason shows its structured wording and opens the row sheet", () => {
  const run = fixture({
    3: { ...failed("INSTANCE_CONNECTION_FAILED"), category: "auth_failed" },
  });
  render(<Harness initial={run} />);
  expect(
    screen.getByText("instance.neon.reason.connection-auth-failed")
  ).toBeVisible();
  fireEvent.click(
    screen.getByRole("button", { name: "instance.neon.edit-settings" })
  );
  expect(handlers.onEditEntry).toHaveBeenCalledWith(run.entries[3].key);
});

test("a rejected batch says why, and that nothing in it was created", () => {
  const run = fixture({
    0: { state: "failed", rejection: "permission-denied", reason: "", message: "" },
    1: { state: "failed", rejection: "permission-denied", reason: "", message: "" },
    20: { state: "unsent" },
  });
  render(<Harness initial={run} />);
  expect(
    screen.getByText("instance.neon.import.banner.rejected-title")
  ).toBeVisible();
  expect(
    screen.getByText(
      /reason.permission-denied.*banner.rejected-body.*banner.unsent/
    )
  ).toBeVisible();
  expect(
    screen.getAllByText("instance.neon.reason.permission-denied")
  ).toHaveLength(2);
  expect(screen.queryByText("instance.neon.import.unknown.description")).toBeNull();
  expect(
    screen.getByRole("button", { name: /failed.retry.*"count":2/ })
  ).toBeEnabled();
});

test("a rejected batch is still named once nothing is left to send", () => {
  const run = fixture({
    0: { state: "failed", rejection: "unauthenticated", reason: "", message: "" },
  });
  render(<Harness initial={run} />);
  expect(
    screen.getByText("instance.neon.import.banner.rejected-title")
  ).toBeVisible();
  expect(
    screen.getByText(/reason.session-expired.*banner.rejected-body/)
  ).toBeVisible();
  expect(
    screen.queryByText("instance.neon.import.banner.finished-title")
  ).toBeNull();
});

test("a pause with nothing left to send does not promise to wait for a continue", () => {
  const lost = { state: "unknown", unknownCause: "lost" } as const;
  const { unmount } = render(<Harness initial={fixture({ 0: lost })} />);
  expect(screen.getByText(/banner.paused-note-all-sent/)).toBeVisible();
  unmount();
  render(<Harness initial={fixture({ 0: lost, 20: { state: "unsent" } })} />);
  expect(screen.getByText(/banner.paused-note$/)).toBeVisible();
});

test("gives sync instructions only for the kinds of instance that were created", () => {
  render(<Harness initial={fixture({})} />);
  expect(
    screen.getByText('instance.neon.import.created.scheduled{"count":25}')
  ).toBeVisible();
  expect(screen.queryByText(/import.created.manual/)).toBeNull();
});

test("created entries are labeled Created with a link and their sync expectation", () => {
  const run = fixture({});
  run.entries[1].draft.syncSeconds = 0;
  render(<Harness initial={run} />);
  expect(
    screen.getByText('instance.neon.import.title.done{"count":25}')
  ).toBeVisible();
  expect(
    screen.getByText('instance.neon.import.created.scheduled{"count":24}')
  ).toBeVisible();
  expect(
    screen.getByText('instance.neon.import.created.manual{"count":1}')
  ).toBeVisible();
  expect(
    screen.getByText('instance.neon.import.created.manual-help{"count":1}')
  ).toBeVisible();
  expect(
    screen.getByRole("link", { name: /created.open.*storefront \/ preview-0/ })
  ).toHaveAttribute("href", "/instances/neon-storefront-preview-0");
  expect(screen.queryByText(/ready|synchronized|download/i)).toBeNull();
  fireEvent.click(
    screen.getByRole("button", { name: /created.show-all.*"count":25/ })
  );
  expect(screen.getAllByRole("link", { name: /created.open/ })).toHaveLength(25);
});

test("states what happened after a stop", () => {
  const run = fixture({
    0: { state: "unknown", unknownCause: "stopped" },
    20: { state: "unsent" },
  });
  render(<Harness initial={run} stopped />);
  expect(screen.getByText("instance.neon.import.title.stopped")).toBeVisible();
  expect(
    screen.getByText(/banner.stopped-created.*stopped-unknown.*stopped-unsent/)
  ).toBeVisible();
  expect(screen.getByText("instance.neon.import.unknown.stopped")).toBeVisible();
});

test("says once that created instances are not synced on a schedule without activation", () => {
  render(<Harness initial={fixture({})} schedule="off" />);
  expect(
    screen.getAllByText("instance.neon.sync.unscheduled-help")
  ).toHaveLength(1);
  expect(screen.queryByText(/created.badge-scheduled/)).toBeNull();
});
