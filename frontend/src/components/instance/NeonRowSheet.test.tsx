import { create } from "@bufbuild/protobuf";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import { useState } from "react";
import { afterEach, expect, test, vi } from "vitest";
import { NeonBranchSchema } from "@/types/proto-es/v1/instance_service_pb";
import { NeonRowSheet } from "./NeonRowSheet";
import {
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

afterEach(cleanup);

const branch = create(NeonBranchSchema, {
  projectId: "icy-math-33158204",
  projectTitle: "analytics",
  branchId: "br-example",
  branchTitle: "main",
  databases: [
    { name: "events", owner: "ingest" },
    { name: "reporting", owner: "analytics_rw" },
  ],
  roles: ["ingest", "analytics_rw"],
});

let latest: NeonImportDraft;
const onClose = vi.fn();

function Harness({
  initial = {},
  shared = defaultSharedSettings(),
  schedule = "editable",
  duplicateIds = new Set<string>(),
}: {
  initial?: Partial<NeonImportDraft>;
  shared?: NeonSharedSettings | null;
  schedule?: NeonScheduleMode;
  duplicateIds?: ReadonlySet<string>;
}) {
  const [draft, setDraft] = useState<NeonImportDraft>({
    ...makeDraft(branch),
    ...initial,
  });
  latest = draft;
  return (
    <NeonRowSheet
      open
      draft={draft}
      shared={shared ?? undefined}
      duplicateIds={duplicateIds}
      schedule={schedule}
      instanceScope="workspace instances"
      onChange={(update) => setDraft(update)}
      onClose={onClose}
    />
  );
}

const choose = async (trigger: HTMLElement, name: string | RegExp) => {
  fireEvent.click(trigger);
  const option = await screen.findByRole("option", { name });
  fireEvent.pointerDown(option, { pointerType: "mouse" });
  fireEvent.click(option);
};

test("holds every setting of a row and says the connection database does not limit sync", () => {
  render(<Harness />);
  expect(screen.getByLabelText("instance.instance-name")).toHaveValue(
    "analytics / main"
  );
  expect(screen.getByLabelText("instance.neon.sheet.instance-id")).toHaveValue(
    "neon-analytics-main"
  );
  expect(screen.getByLabelText("common.environment")).toHaveTextContent(
    /shared-option/
  );
  expect(screen.getByLabelText("instance.neon.sheet.role")).toBeVisible();
  expect(screen.getByLabelText("common.password")).toHaveTextContent(
    "instance.neon.sheet.password-neon"
  );
  expect(screen.getByLabelText("instance.neon.sheet.database")).toHaveTextContent(
    "events"
  );
  expect(screen.getByText("instance.neon.sheet.database-help")).toBeVisible();
  expect(
    screen.getByRole("switch", { name: "instance.sync-databases.sync-all" })
  ).toBeChecked();
  expect(
    screen.getByLabelText("instance.neon.configure.interval")
  ).toHaveTextContent(/shared-option.*interval.hourly/);
});

test("a row follows the shared interval until one is chosen, and can go back to it", async () => {
  render(<Harness />);
  const interval = () =>
    screen.getByLabelText("instance.neon.configure.interval");
  await choose(interval(), "instance.neon.interval.off");
  expect(latest.syncSeconds).toBe(0);
  expect(interval()).toHaveTextContent("instance.neon.interval.off");
  expect(interval()).toHaveAccessibleDescription(
    "instance.neon.sheet.interval-off"
  );
  await choose(interval(), /shared-option.*interval.hourly/);
  expect(latest.syncSeconds).toBeUndefined();
});

test("mounts a fresh form for another row", () => {
  const other = makeDraft(
    create(NeonBranchSchema, { ...branch, branchId: "br-other" })
  );
  const view = (draft: NeonImportDraft) => (
    <NeonRowSheet
      open
      draft={draft}
      shared={defaultSharedSettings()}
      duplicateIds={new Set()}
      schedule="editable"
      instanceScope="workspace instances"
      onChange={() => {}}
      onClose={onClose}
    />
  );
  const { rerender } = render(view(makeDraft(branch)));
  const before = screen.getByLabelText("instance.instance-name");
  rerender(view({ ...makeDraft(branch), title: "renamed" }));
  expect(screen.getByLabelText("instance.instance-name")).toBe(before);
  rerender(view(other));
  expect(screen.getByLabelText("instance.instance-name")).not.toBe(before);
});

test("requires a role when the databases have different owners", async () => {
  render(<Harness />);
  expect(screen.getByText("instance.neon.sheet.role-required")).toBeVisible();
  await choose(screen.getByLabelText("instance.neon.sheet.role"), "ingest");
  expect(latest.role).toBe("ingest");
  expect(screen.getByText("instance.neon.sheet.role-check")).toBeVisible();
});

test("an empty database subset is valid and summarized as none", () => {
  render(<Harness />);
  fireEvent.click(
    screen.getByRole("switch", { name: "instance.sync-databases.sync-all" })
  );
  expect(latest.syncAll).toBe(false);
  const databases = screen.getByRole("group", {
    name: "instance.neon.sheet.databases-legend",
  });
  expect(within(databases).getByText(/owner.*ingest/)).toBeVisible();
  fireEvent.click(within(databases).getByRole("checkbox", { name: "reporting" }));
  // The remaining database has one owner, which fills the empty role.
  expect(latest).toMatchObject({ databases: ["events"], role: "ingest" });
  fireEvent.click(within(databases).getByRole("checkbox", { name: "events" }));
  expect(latest.databases).toEqual([]);
  expect(screen.getByText("instance.neon.sheet.scope-none")).toBeVisible();
  expect(latest.database).toBe("events");
});

test("a typed password must be nonempty and is cleared when the role changes", async () => {
  render(<Harness initial={{ role: "ingest" }} />);
  await choose(
    screen.getByLabelText("common.password"),
    "instance.neon.sheet.password-typed"
  );
  expect(latest.passwordSource).toBe("typed");
  expect(screen.getByText("instance.neon.sheet.password-required")).toBeVisible();
  fireEvent.change(screen.getByLabelText(/password-for.*ingest/), {
    target: { value: "example-password" },
  });
  expect(latest.password).toBe("example-password");
  await choose(screen.getByLabelText("instance.neon.sheet.role"), "analytics_rw");
  expect(latest).toMatchObject({ role: "analytics_rw", password: "" });
});

test("resets a row to the shared settings", () => {
  render(<Harness initial={{ role: "ingest", environment: "", syncSeconds: 0 }} />);
  fireEvent.click(
    screen.getByRole("button", { name: "instance.neon.sheet.reset" })
  );
  expect(latest.environment).toBeUndefined();
  expect(latest.syncSeconds).toBeUndefined();
  expect(
    screen.queryByRole("button", { name: "instance.neon.sheet.reset" })
  ).toBeNull();
});

test("flags an instance ID that is invalid or used by another row", () => {
  render(<Harness duplicateIds={new Set(["neon-analytics-main"])} />);
  expect(
    screen.getByLabelText("instance.neon.sheet.instance-id")
  ).toHaveAccessibleDescription("instance.neon.sheet.id-duplicate");
  fireEvent.change(screen.getByLabelText("instance.neon.sheet.instance-id"), {
    target: { value: "Not Valid" },
  });
  expect(screen.getByText(/resource-id.validation.pattern/)).toBeVisible();
});

test.each([
  "off",
  "default",
] as const)("omits the interval control when the schedule is %s", (schedule) => {
  render(<Harness schedule={schedule} />);
  expect(
    screen.queryByLabelText("instance.neon.configure.interval")
  ).toBeNull();
});

test("offers no shared value once settings are frozen for import", () => {
  render(<Harness shared={null} initial={{ environment: "environments/prod" }} />);
  expect(screen.getByLabelText("common.environment")).toHaveTextContent("Prod");
  expect(
    screen.queryByRole("button", { name: "instance.neon.sheet.reset" })
  ).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "common.done" }));
  expect(onClose).toHaveBeenCalled();
});
