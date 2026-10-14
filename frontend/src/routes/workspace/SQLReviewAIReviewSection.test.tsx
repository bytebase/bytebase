import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";

const SWITCH = { name: "sql-review.ai-review.enable.self" };

const mocks = vi.hoisted(() => ({
  permissions: {} as Record<string, boolean>,
  saas: { value: false },
  upsertSetting: vi.fn(),
  pushNotification: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => ({ isSaaSMode: mocks.saas.value }),
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: (permission: string) =>
    mocks.permissions[permission] ?? true,
}));

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({ children }: { children: ReactNode }) => (
    <a href="/">{children}</a>
  ),
}));

// Mirrors the real store: a saved AI setting updates the server info.
vi.mock("@/stores/app", async () => {
  const { create: createStore } = await import("zustand");
  type ServerInfo = { aiEnabled: boolean; aiSqlReviewEnabled: boolean };
  type Saved = {
    value: { value: { value: { sqlReviewEnabled: boolean } } };
  };
  const useAppStore = createStore<{
    serverInfo: ServerInfo;
    upsertSetting: (params: Saved) => Promise<void>;
  }>()((set) => ({
    serverInfo: { aiEnabled: true, aiSqlReviewEnabled: false },
    upsertSetting: async (params) => {
      await mocks.upsertSetting(params);
      set((state) => ({
        serverInfo: {
          ...state.serverInfo,
          aiSqlReviewEnabled: params.value.value.value.sqlReviewEnabled,
        },
      }));
    },
  }));
  return { useAppStore };
});

const { useAppStore } = await import("@/stores/app");
const seedServerInfo = (serverInfo: {
  aiEnabled: boolean;
  aiSqlReviewEnabled: boolean;
}) =>
  (
    useAppStore as unknown as {
      setState: (state: { serverInfo: typeof serverInfo }) => void;
    }
  ).setState({ serverInfo });

const { SQLReviewAIReviewSection, useWorkspaceAIReview } = await import(
  "./SQLReviewAIReviewSection"
);

function Harness() {
  const aiReview = useWorkspaceAIReview();
  return (
    <>
      <SQLReviewAIReviewSection aiReview={aiReview} />
      <output>{aiReview.isDirty ? "dirty" : "clean"}</output>
      <button type="button" onClick={() => void aiReview.save()}>
        save
      </button>
      <button
        type="button"
        onClick={() => void aiReview.save({ silent: true })}
      >
        save silently
      </button>
      <button type="button" onClick={aiReview.revert}>
        revert
      </button>
    </>
  );
}

const save = () =>
  act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "save" }));
  });

describe("SQLReviewAIReviewSection", () => {
  beforeEach(() => {
    mocks.permissions = {};
    mocks.saas.value = false;
    mocks.upsertSetting.mockReset();
    mocks.pushNotification.mockReset();
    seedServerInfo({ aiEnabled: true, aiSqlReviewEnabled: false });
  });

  test("saves the switch under its own mask path", async () => {
    render(<Harness />);
    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "false"
    );
    expect(screen.getByText("clean")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("switch", SWITCH));
    expect(screen.getByText("dirty")).toBeInTheDocument();
    await save();

    expect(mocks.upsertSetting).toHaveBeenCalledOnce();
    const saved = mocks.upsertSetting.mock.calls[0][0];
    expect(saved.name).toBe(Setting_SettingName.AI);
    expect(saved.updateMask.paths).toEqual(["value.ai.sql_review_enabled"]);
    expect(saved.value.value.value.sqlReviewEnabled).toBe(true);
    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByText("clean")).toBeInTheDocument();
    expect(mocks.pushNotification).toHaveBeenCalledOnce();
  });

  test("a silent save reports nothing", async () => {
    render(<Harness />);

    fireEvent.click(screen.getByRole("switch", SWITCH));
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "save silently" }));
    });

    expect(mocks.upsertSetting).toHaveBeenCalledOnce();
    expect(screen.getByText("clean")).toBeInTheDocument();
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("reverting drops the unsaved choice", () => {
    seedServerInfo({ aiEnabled: true, aiSqlReviewEnabled: true });
    render(<Harness />);

    fireEvent.click(screen.getByRole("switch", SWITCH));
    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "false"
    );
    fireEvent.click(screen.getByRole("button", { name: "revert" }));

    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByText("clean")).toBeInTheDocument();
  });

  test("a failed save keeps the unsaved choice", async () => {
    mocks.upsertSetting.mockRejectedValueOnce(new Error("save failed"));
    render(<Harness />);

    fireEvent.click(screen.getByRole("switch", SWITCH));
    await save();

    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByText("dirty")).toBeInTheDocument();
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("shows the state as text without bb.settings.set", () => {
    mocks.permissions["bb.settings.set"] = false;
    seedServerInfo({ aiEnabled: true, aiSqlReviewEnabled: true });
    render(<Harness />);

    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(screen.getByText("sql-review.standard-rules.on")).toBeInTheDocument();
  });

  test("shows the state as text in SaaS mode, where the AI setting is read only", () => {
    mocks.saas.value = true;
    render(<Harness />);

    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(
      screen.getByText("sql-review.standard-rules.off")
    ).toBeInTheDocument();
  });

  test("warns while AI is off for the workspace, with a link to turn it on", () => {
    seedServerInfo({ aiEnabled: false, aiSqlReviewEnabled: true });
    render(<Harness />);

    expect(
      screen.getByText("sql-review.ai-review.ai-off")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "sql-review.ai-review.turn-on-ai" })
    ).toBeInTheDocument();
  });

  test.each([
    ["without bb.settings.set", () => (mocks.permissions["bb.settings.set"] = false)],
    ["without bb.settings.get", () => (mocks.permissions["bb.settings.get"] = false)],
    [
      "without bb.settings.getWorkspaceProfile",
      () => (mocks.permissions["bb.settings.getWorkspaceProfile"] = false),
    ],
    ["in SaaS mode", () => (mocks.saas.value = true)],
  ])("warns without the link %s, where the user cannot turn AI on", (_name, arrange) => {
    arrange();
    seedServerInfo({ aiEnabled: false, aiSqlReviewEnabled: false });
    render(<Harness />);

    expect(
      screen.getByText("sql-review.ai-review.ai-off")
    ).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  test("shows no warning while AI is on", () => {
    render(<Harness />);

    expect(
      screen.queryByText("sql-review.ai-review.ai-off")
    ).not.toBeInTheDocument();
  });
});
