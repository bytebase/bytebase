import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ActuatorInfoSchema } from "@/types/proto-es/v1/actuator_service_pb";
import {
  AISettingSchema,
  Setting_SettingName,
  SettingSchema,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { createWorkspaceSlice } from "./workspace";

const mocks = vi.hoisted(() => ({
  updateSetting: vi.fn(),
}));

vi.mock("@/api", () => ({
  settingServiceClientConnect: { updateSetting: mocks.updateSetting },
}));
vi.mock("@/stores/workspaceSwitchChannel", () => ({
  workspaceSwitchChannel: { addEventListener: vi.fn() },
}));

const createStore = () => {
  const state: Record<string, unknown> = {};
  const set = (updater: unknown) => {
    const patch =
      typeof updater === "function"
        ? (updater as (value: typeof state) => object)(state)
        : updater;
    Object.assign(state, patch);
  };
  Object.assign(
    state,
    createWorkspaceSlice(set as never, (() => state) as never, {} as never)
  );
  state.serverInfo = create(ActuatorInfoSchema, { aiEnabled: false });
  return state as ReturnType<typeof createWorkspaceSlice>;
};

const aiSetting = (enabled: boolean) =>
  create(SettingSchema, {
    name: "settings/AI",
    value: create(SettingValueSchema, {
      value: {
        case: "ai",
        value: create(AISettingSchema, { enabled }),
      },
    }),
  });

describe("workspace AI setting cache", () => {
  beforeEach(() => vi.clearAllMocks());

  test("updates Ask AI availability from a successful setting response", async () => {
    const store = createStore();
    const enabled = aiSetting(true);
    mocks.updateSetting.mockResolvedValueOnce(enabled);

    await store.upsertSetting({
      name: Setting_SettingName.AI,
      value: enabled.value!,
    });

    expect(store.serverInfo?.aiEnabled).toBe(true);
    expect(store.settingsByName[enabled.name]).toBe(enabled);

    const disabled = aiSetting(false);
    mocks.updateSetting.mockResolvedValueOnce(disabled);
    await store.upsertSetting({
      name: Setting_SettingName.AI,
      value: disabled.value!,
    });
    expect(store.serverInfo?.aiEnabled).toBe(false);
  });

  test("does not change Ask AI availability when the setting save fails", async () => {
    const store = createStore();
    mocks.updateSetting.mockRejectedValueOnce(new Error("save failed"));

    await expect(
      store.upsertSetting({
        name: Setting_SettingName.AI,
        value: aiSetting(true).value!,
      })
    ).rejects.toThrow("save failed");

    expect(store.serverInfo?.aiEnabled).toBe(false);
  });

  test("does not change Ask AI availability for a validation-only response", async () => {
    const store = createStore();
    const setting = aiSetting(true);
    mocks.updateSetting.mockResolvedValueOnce(setting);

    await store.upsertSetting({
      name: Setting_SettingName.AI,
      value: setting.value!,
      validateOnly: true,
    });

    expect(store.serverInfo?.aiEnabled).toBe(false);
  });
});
