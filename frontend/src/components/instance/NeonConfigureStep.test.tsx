import { create } from "@bufbuild/protobuf";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { NeonBranchSchema } from "@/types/proto-es/v1/instance_service_pb";
import { NeonConfigureStep } from "./NeonConfigureStep";
import {
  branchKey,
  defaultSharedSettings,
  makeDraft,
  type NeonImportDraft,
  type NeonScheduleMode,
  type NeonSharedSettings,
} from "./neon-import";

const t = (key: string, values?: Record<string, unknown>) =>
  values ? `${key}${JSON.stringify(values)}` : key;
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t }),
}));
vi.mock("@/components/FeatureBadge", () => ({ FeatureBadge: () => null }));
vi.mock("@/hooks/useAppState", () => ({
  useEnvironmentList: () => [
    { id: "test", title: "Test" },
    { id: "prod", title: "Prod" },
  ],
}));
vi.mock("@/components/EnvironmentSelect", () => ({
  EnvironmentSelect: ({
    value,
    onChange,
  }: {
    value: string;
    onChange: (value: string) => void;
  }) => (
    <input
      aria-label="environment select"
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  ),
}));

beforeEach(() => {
  // The selection action bar reads the viewport tier; show every action inline.
  window.matchMedia = ((query: string) => ({
    matches: true,
    media: query,
    addEventListener: () => {},
    removeEventListener: () => {},
  })) as unknown as typeof window.matchMedia;
  for (const handler of Object.values(handlers)) handler.mockClear();
});
afterEach(cleanup);

const branch = (projectTitle: string, title: string, owners = ["app_owner"]) =>
  create(NeonBranchSchema, {
    projectId: `${projectTitle}-id`,
    projectTitle,
    branchId: `br-${projectTitle}-${title}`,
    branchTitle: title,
    databases: owners.map((owner, index) => ({ name: `db${index}`, owner })),
    roles: [...owners, "neondb_owner"],
  });

const selected = (drafts: NeonImportDraft[]) =>
  drafts.map((draft) => ({ ...draft, selected: true }));

const fixture = () =>
  selected([
    makeDraft(branch("storefront", "main")),
    makeDraft(branch("storefront", "staging")),
    makeDraft(branch("analytics", "main", ["ingest", "analytics_rw"])),
    makeDraft(branch("analytics", "staging", ["ingest", "analytics_rw"])),
  ]);

let latest: NeonImportDraft[] = [];
const handlers = {
  onResetOverride: vi.fn(),
  onRemove: vi.fn(),
  onOpenRow: vi.fn(),
};

function Harness({
  initial = fixture(),
  capacity,
  schedule = "editable",
}: {
  initial?: NeonImportDraft[];
  capacity?: number;
  schedule?: NeonScheduleMode;
}) {
  const [drafts, setDrafts] = useState(initial);
  const [shared, setShared] = useState<NeonSharedSettings>(
    defaultSharedSettings
  );
  latest = drafts;
  return (
    <NeonConfigureStep
      drafts={drafts}
      shared={shared}
      capacity={capacity}
      schedule={schedule}
      onSharedChange={(change) => setShared((value) => ({ ...value, ...change }))}
      onDraftsChange={(keys, update) =>
        setDrafts((rows) =>
          rows.map((row) =>
            keys.includes(branchKey(row.branch)) ? update(row) : row
          )
        )
      }
      {...handlers}
    />
  );
}

const choose = async (trigger: HTMLElement, name: string | RegExp) => {
  fireEvent.click(trigger);
  const option = await screen.findByRole("option", { name });
  fireEvent.pointerDown(option, { pointerType: "mouse" });
  fireEvent.click(option);
};

test("asks for an ambiguous role once per Neon project and applies it to its rows", async () => {
  render(<Harness />);
  const panel = screen.getByRole("region", {
    name: "instance.neon.configure.needs.label",
  });
  expect(
    within(panel).getByText(/needs.title.*"count":2/)
  ).toBeVisible();
  const questions = within(panel).getAllByRole("combobox");
  expect(questions).toHaveLength(1);
  expect(
    within(panel).getByText(/needs.role-body.*db0: ingest, db1: analytics_rw/)
  ).toBeVisible();
  await choose(questions[0], "analytics_rw");
  expect(latest.map((row) => row.role)).toEqual([
    "app_owner",
    "app_owner",
    "analytics_rw",
    "analytics_rw",
  ]);
  // The panel stays to say nothing is left, and takes the focus the answered
  // question's control held.
  const status = within(panel).getByRole("status");
  expect(status).toHaveTextContent("instance.neon.configure.empty-needs");
  await waitFor(() => expect(status).toHaveFocus());
});

test("answering a role question moves focus to the next one", async () => {
  render(
    <Harness
      initial={selected([
        makeDraft(branch("analytics", "main", ["ingest", "analytics_rw"])),
        makeDraft(branch("billing", "main", ["ledger", "billing_rw"])),
      ])}
    />
  );
  const panel = screen.getByRole("region", {
    name: "instance.neon.configure.needs.label",
  });
  await choose(within(panel).getAllByRole("combobox")[0], "analytics_rw");
  const remaining = within(panel).getAllByRole("combobox");
  expect(remaining).toHaveLength(1);
  await waitFor(() => expect(remaining[0]).toHaveFocus());
});

test("asks separately when branches of one Neon project have different database owners", () => {
  const drafts = selected([
    makeDraft(branch("analytics", "main", ["ingest", "analytics_rw"])),
    makeDraft(branch("analytics", "staging", ["analytics_rw", "ingest"])),
  ]);
  render(<Harness initial={drafts} />);
  const panel = screen.getByRole("region", {
    name: "instance.neon.configure.needs.label",
  });
  expect(within(panel).getAllByRole("combobox")).toHaveLength(2);
  expect(
    within(panel).getByText(/needs.role-body.*db0: ingest, db1: analytics_rw/)
  ).toBeVisible();
  expect(
    within(panel).getByText(/needs.role-body.*db0: analytics_rw, db1: ingest/)
  ).toBeVisible();
});

test("names at most three rows in one role question", () => {
  const drafts = selected(
    ["a", "b", "c", "d", "e"].map((title) =>
      makeDraft(branch("analytics", title, ["ingest", "analytics_rw"]))
    )
  );
  render(<Harness initial={drafts} />);
  expect(
    screen.getByText(
      /needs.role-title.*analytics \/ a, analytics \/ b, analytics \/ c instance.neon.import.and-more.*count.*2/
    )
  ).toBeVisible();
});

test("lists a typed password that is empty and lets it fall back to Neon", () => {
  const drafts = fixture().slice(0, 2);
  drafts[1] = { ...drafts[1], passwordSource: "typed" };
  render(<Harness initial={drafts} />);
  const panel = screen.getByRole("region", {
    name: "instance.neon.configure.needs.label",
  });
  expect(
    within(panel).getByText(/needs.password-title.*storefront \/ staging/)
  ).toBeVisible();
  // Each character counts at once, and the field stays to finish typing in.
  fireEvent.change(within(panel).getByLabelText(/password-for.*app_owner/), {
    target: { value: "e" },
  });
  expect(latest[1].password).toBe("e");
  fireEvent.change(within(panel).getByLabelText(/password-for.*app_owner/), {
    target: { value: "example-password" },
  });
  expect(latest[1].password).toBe("example-password");
  expect(within(panel).getByLabelText(/password-for.*app_owner/)).toHaveValue(
    "example-password"
  );
  expect(
    within(panel).getByText("instance.neon.configure.empty-needs")
  ).toBeVisible();
  cleanup();
  render(<Harness initial={drafts} />);
  fireEvent.click(
    screen.getByRole("button", {
      name: "instance.neon.configure.needs.use-neon",
    })
  );
  expect(latest[1]).toMatchObject({ passwordSource: "neon", password: "" });
});

test("reports how many rows differ from a shared setting and resets them", async () => {
  render(<Harness initial={fixture().slice(0, 2)} />);
  await choose(
    screen.getByRole("combobox", {
      name: /row-environment-label.*storefront \/ main/,
    }),
    "Prod"
  );
  expect(latest[0].environment).toBe("environments/prod");
  expect(latest[1].environment).toBeUndefined();
  expect(screen.getByText(/configure.differ.*"count":1/)).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "common.reset" }));
  expect(handlers.onResetOverride).toHaveBeenCalledWith("environment");
});

test("a row inherits a shared value until it is changed on the row", () => {
  render(<Harness initial={fixture().slice(0, 1)} />);
  expect(
    screen.getByRole("combobox", { name: /row-environment-label/ })
  ).toHaveTextContent(/shared-option.*no-environment/);
  fireEvent.change(screen.getByLabelText("environment select"), {
    target: { value: "environments/test" },
  });
  expect(
    screen.getByRole("combobox", { name: /row-environment-label/ })
  ).toHaveTextContent('shared-option{"value":"Test"}');
  expect(latest[0].environment).toBeUndefined();
});

test("describes sync as scheduled or manual and has no activation control", () => {
  const drafts = fixture().slice(0, 2);
  drafts[1] = { ...drafts[1], syncSeconds: 0 };
  render(<Harness initial={drafts} />);
  expect(screen.getAllByText(/instance.neon.sync.scheduled/)).not.toHaveLength(0);
  expect(screen.getAllByText("instance.neon.sync.manual")).not.toHaveLength(0);
  expect(
    screen.getByLabelText("instance.neon.configure.interval")
  ).toHaveTextContent("instance.neon.interval.hourly");
  expect(screen.queryByText(/activat/i, { selector: "label" })).toBeNull();
  expect(screen.queryByRole("switch", { name: /assign-license/ })).toBeNull();
});

test("says once that instances are not synced on a schedule where the license provides no activation", () => {
  render(<Harness initial={fixture().slice(0, 2)} schedule="off" />);
  expect(
    screen.getAllByText("instance.neon.sync.unscheduled-help")
  ).toHaveLength(1);
  expect(
    screen.queryByLabelText("instance.neon.configure.interval")
  ).toBeNull();
  expect(screen.queryByText(/instance.neon.sync.scheduled/)).toBeNull();
});

test("says once that the plan fixes the interval and offers no control for it", () => {
  render(<Harness initial={fixture().slice(0, 2)} schedule="default" />);
  expect(screen.getAllByText("instance.neon.sync.default-help")).toHaveLength(
    1
  );
  expect(
    screen.queryByLabelText("instance.neon.configure.interval")
  ).toBeNull();
  expect(screen.getAllByText(/instance.neon.sync.scheduled/)).not.toHaveLength(
    0
  );
  fireEvent.click(
    screen.getByRole("checkbox", { name: /row-select-label.*storefront \/ main/ })
  );
  expect(
    screen.queryByRole("button", {
      name: "instance.neon.configure.bulk.set-interval",
    })
  ).toBeNull();
});

test("the shared interval reaches every row that has not set its own", async () => {
  const drafts = fixture().slice(0, 2);
  drafts[1] = { ...drafts[1], syncSeconds: 0 };
  render(<Harness initial={drafts} />);
  await choose(
    screen.getByLabelText("instance.neon.configure.interval"),
    'instance.neon.interval.every-n-hours{"hours":6}'
  );
  expect(latest.map((row) => row.syncSeconds)).toEqual([undefined, 0]);
  expect(
    screen.getAllByText(/sync.scheduled.*interval.hours.*count.*6/)
  ).not.toHaveLength(0);
  expect(screen.getAllByText("instance.neon.sync.manual")).not.toHaveLength(0);
  expect(screen.getByText(/configure.differ.*"count":1/)).toBeVisible();
});

test("shows the remaining instance limit and when the selection exceeds it", () => {
  render(<Harness capacity={3} />);
  expect(
    screen.getByText('instance.neon.configure.limit.usage{"count":4,"capacity":3}')
  ).toBeVisible();
  expect(screen.getByText(/configure.limit.over/)).toBeVisible();
});

test("bulk actions reach only the checked rows the current tab shows", () => {
  render(<Harness />);
  fireEvent.click(
    screen.getByRole("checkbox", { name: /row-select-label.*storefront \/ main/ })
  );
  expect(screen.getByText(/rows-selected.*"count":1/)).toBeVisible();
  // storefront / main needs no input, so the Needs input tab hides it.
  fireEvent.click(screen.getByRole("radio", { name: /configure.tab-needs/ }));
  expect(screen.queryByText(/rows-selected/)).toBeNull();
  fireEvent.click(screen.getByRole("radio", { name: /configure.tab-all/ }));
  expect(screen.getByText(/rows-selected.*"count":1/)).toBeVisible();
});

test("bulk edits go through the selection action bar and removal is confirmed with its count", async () => {
  render(<Harness initial={fixture().slice(0, 2)} />);
  expect(screen.queryByText(/rows-selected/)).toBeNull();
  fireEvent.click(
    screen.getByRole("checkbox", { name: /row-select-label.*storefront \/ main/ })
  );
  fireEvent.click(
    screen.getByRole("checkbox", {
      name: /row-select-label.*storefront \/ staging/,
    })
  );
  expect(screen.getByText(/rows-selected.*"count":2/)).toBeVisible();

  const applyInterval = async (option: string) => {
    fireEvent.click(
      screen.getByRole("button", {
        name: "instance.neon.configure.bulk.set-interval",
      })
    );
    const dialog = await screen.findByRole("dialog");
    const apply = within(dialog).getByRole("button", {
      name: "instance.neon.configure.bulk.apply",
    });
    // Nothing is chosen yet, so there is nothing to apply.
    expect(apply).toBeDisabled();
    await choose(
      within(dialog).getByLabelText("instance.neon.configure.interval"),
      option
    );
    fireEvent.click(apply);
  };
  fireEvent.click(
    screen.getByRole("button", {
      name: "instance.neon.configure.bulk.set-environment",
    })
  );
  const environmentDialog = await screen.findByRole("dialog");
  const applyEnvironment = within(environmentDialog).getByRole("button", {
    name: "instance.neon.configure.bulk.apply",
  });
  expect(applyEnvironment).toBeDisabled();
  await choose(
    within(environmentDialog).getByLabelText("common.environment"),
    "Prod"
  );
  fireEvent.click(applyEnvironment);
  expect(latest.map((row) => row.environment)).toEqual([
    "environments/prod",
    "environments/prod",
  ]);
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());

  await applyInterval("instance.neon.interval.off");
  expect(latest.map((row) => row.syncSeconds)).toEqual([0, 0]);
  await applyInterval("instance.neon.configure.bulk.use-shared-setting");
  expect(latest.map((row) => row.syncSeconds)).toEqual([undefined, undefined]);

  fireEvent.click(
    screen.getByRole("button", { name: "instance.neon.configure.bulk.remove" })
  );
  const confirm = await screen.findByRole("alertdialog");
  expect(
    within(confirm).getByText(/bulk.remove-title.*"count":2/)
  ).toBeVisible();
  expect(handlers.onRemove).not.toHaveBeenCalled();
  fireEvent.click(
    within(confirm).getByRole("button", {
      name: "instance.neon.configure.bulk.remove",
    })
  );
  expect(handlers.onRemove).toHaveBeenCalledWith(
    latest.map((row) => branchKey(row.branch))
  );
});

test("opens the row sheet from the instance name", () => {
  render(<Harness initial={fixture().slice(0, 1)} />);
  fireEvent.click(screen.getByRole("button", { name: "storefront / main" }));
  expect(handlers.onOpenRow).toHaveBeenCalledWith(branchKey(latest[0].branch));
});
