import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { state } = vi.hoisted(() => ({
  state: {
    classification: () => [
      {
        id: "classification-config",
        title: "Data classification",
        levels: [{ title: "Confidential", level: 1 }],
        classification: {
          confidential: { id: "confidential", title: "Confidential", level: 1 },
        },
      },
    ],
    getOrFetchSettingByName: vi.fn(async () => undefined),
    hasInstanceFeature: vi.fn(() => true),
    settingsByName: {},
    upsertSetting: vi.fn(async () => undefined),
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (store: typeof state) => unknown) => selector(state),
    { getState: () => state }
  ),
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/utils", () => ({ hasWorkspacePermissionV2: () => true }));
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/ClassificationTree", () => ({
  ClassificationTree: () => null,
}));
vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));
vi.mock("@/modules/agent/components/AgentPageHelpButton", () => ({
  AgentPageHelpButton: () => (
    <button type="button">Ask Agent about masking</button>
  ),
}));
vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init: Record<string, unknown>) => ({ ...init }),
}));
vi.mock("@/types/proto-es/v1/setting_service_pb", () => ({
  DataClassificationSetting_DataClassificationConfig_DataClassificationSchema: {},
  DataClassificationSetting_DataClassificationConfig_LevelSchema: {},
  DataClassificationSetting_DataClassificationConfigSchema: {},
  DataClassificationSettingSchema: {},
  Setting_SettingName: { DATA_CLASSIFICATION: "DATA_CLASSIFICATION" },
  SettingValueSchema: {},
}));
vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_DATA_CLASSIFICATION: 1 },
}));

import { DataClassificationPage } from "./DataClassificationPage";

beforeEach(() => {
  vi.clearAllMocks();
  state.hasInstanceFeature.mockReturnValue(true);
});

afterEach(cleanup);

async function openClearConfirmation() {
  render(<DataClassificationPage />);
  const trigger = await screen.findByRole("button", { name: "common.clear" });
  fireEvent.click(trigger);
  return { trigger, dialog: await screen.findByRole("dialog") };
}

describe("DataClassificationPage", () => {
  it("lets the toolbar actions wrap within the available width", async () => {
    render(<DataClassificationPage />);
    const helpButton = await screen.findByRole("button", {
      name: "Ask Agent about masking",
    });
    const group = helpButton.parentElement;

    expect(group?.classList.contains("flex-wrap")).toBe(true);
    expect(group?.classList.contains("gap-2")).toBe(true);
    expect(group?.classList.contains("min-w-0")).toBe(true);
    expect(group?.classList.contains("max-w-full")).toBe(true);
    expect(group?.classList.contains("shrink-0")).toBe(false);
  });

  it("opens Clear confirmation outside the toolbar without replacing its trigger", async () => {
    const { trigger, dialog } = await openClearConfirmation();
    const toolbar = trigger.closest("[data-slot='workspace-page-toolbar']");

    expect(toolbar).toContainElement(trigger);
    expect(toolbar).not.toContainElement(dialog);
    expect(
      within(toolbar as HTMLElement).getByRole("button", { name: "common.clear" })
    ).toBe(trigger);
    expect(
      within(toolbar as HTMLElement).getByRole("button", { name: "common.edit" })
    ).toBeVisible();
    expect(
      within(dialog).getByText("bbkit.confirm-button.sure-to-delete")
    ).toBeVisible();
    expect(state.upsertSetting).not.toHaveBeenCalled();
  });

  it("cancels Clear without changing the saved configuration", async () => {
    const { trigger, dialog } = await openClearConfirmation();

    fireEvent.click(within(dialog).getByRole("button", { name: "common.cancel" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
    expect(trigger).toBeVisible();
    expect(state.upsertSetting).not.toHaveBeenCalled();
  });

  it("clears the saved configuration only after confirmation", async () => {
    const { dialog } = await openClearConfirmation();

    fireEvent.click(within(dialog).getByRole("button", { name: "common.clear" }));

    await waitFor(() =>
      expect(state.upsertSetting).toHaveBeenCalledExactlyOnceWith({
        name: "DATA_CLASSIFICATION",
        value: {
          value: { case: "dataClassification", value: { configs: [] } },
        },
      })
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
  });

  it("keeps Clear disabled when data classification is unavailable", async () => {
    state.hasInstanceFeature.mockReturnValue(false);
    render(<DataClassificationPage />);
    const trigger = await screen.findByRole("button", { name: "common.clear" });

    expect(trigger).toBeDisabled();
    fireEvent.click(trigger);

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(state.upsertSetting).not.toHaveBeenCalled();
  });
});
