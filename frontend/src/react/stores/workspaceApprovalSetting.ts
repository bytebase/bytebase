import { create as createProto } from "@bufbuild/protobuf";
import { v4 as uuidv4 } from "uuid";
import { create } from "zustand";
import { settingServiceClientConnect } from "@/connect";
import { settingNamePrefix } from "@/react/lib/resourceName";
import { useAppStore } from "@/react/stores/app";
import type { LocalApprovalConfig, LocalApprovalRule } from "@/types";
import {
  Setting_SettingName,
  SettingSchema,
  SettingValueSchema,
  UpdateSettingRequestSchema,
  type WorkspaceApprovalSetting_Rule_Source,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  buildWorkspaceApprovalSetting,
  resolveLocalApprovalConfig,
} from "@/utils";

// Standalone Zustand port of the legacy Pinia `useWorkspaceApprovalSettingStore`.
// Keeps the same surface (a reactive `config` + rule mutators) so consumers
// only swap the import path. Reads/writes the WORKSPACE_APPROVAL setting through
// the app store (fetch) and the setting service (upsert), then syncs the app
// store cache — no Pinia dependency.
export type WorkspaceApprovalSettingState = {
  config: LocalApprovalConfig;
  fetchConfig: () => Promise<void>;
  getRulesBySource: (
    source: WorkspaceApprovalSetting_Rule_Source
  ) => LocalApprovalRule[];
  addRule: (rule: Omit<LocalApprovalRule, "uid">) => Promise<void>;
  updateRule: (
    uid: string,
    updates: Partial<LocalApprovalRule>
  ) => Promise<void>;
  deleteRule: (uid: string) => Promise<void>;
  reorderRules: (
    source: WorkspaceApprovalSetting_Rule_Source,
    fromIndex: number,
    toIndex: number
  ) => Promise<void>;
};

export const useWorkspaceApprovalSettingStore =
  create<WorkspaceApprovalSettingState>()((set, get) => {
    const persist = async (config: LocalApprovalConfig) => {
      const approvalSetting = await buildWorkspaceApprovalSetting(config);
      const response = await settingServiceClientConnect.updateSetting(
        createProto(UpdateSettingRequestSchema, {
          setting: createProto(SettingSchema, {
            name: `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.WORKSPACE_APPROVAL]}`,
            value: createProto(SettingValueSchema, {
              value: { case: "workspaceApproval", value: approvalSetting },
            }),
          }),
          allowMissing: true,
        })
      );
      // Keep the app-store setting cache in sync (mirrors the Pinia bridge).
      useAppStore.getState().setSettingByName(response);
    };

    // Optimistically apply `nextConfig`, persist it, and roll back on failure.
    const applyAndPersist = async (nextConfig: LocalApprovalConfig) => {
      const backup = get().config;
      set({ config: nextConfig });
      try {
        await persist(nextConfig);
      } catch (err) {
        set({ config: backup });
        throw err;
      }
    };

    return {
      config: { rules: [] },

      fetchConfig: async () => {
        try {
          const setting = await useAppStore
            .getState()
            .getOrFetchSettingByName(Setting_SettingName.WORKSPACE_APPROVAL);
          if (setting?.value?.value?.case === "workspaceApproval") {
            const config = await resolveLocalApprovalConfig(
              setting.value.value.value
            );
            set({ config });
          }
        } catch (ex) {
          console.error(ex);
        }
      },

      getRulesBySource: (source) =>
        get().config.rules.filter((rule) => rule.source === source),

      addRule: async (rule) => {
        const newRule: LocalApprovalRule = { ...rule, uid: uuidv4() };
        await applyAndPersist({
          ...get().config,
          rules: [...get().config.rules, newRule],
        });
      },

      updateRule: async (uid, updates) => {
        await applyAndPersist({
          ...get().config,
          rules: get().config.rules.map((rule) =>
            rule.uid === uid ? { ...rule, ...updates } : rule
          ),
        });
      },

      deleteRule: async (uid) => {
        await applyAndPersist({
          ...get().config,
          rules: get().config.rules.filter((rule) => rule.uid !== uid),
        });
      },

      reorderRules: async (source, fromIndex, toIndex) => {
        const sourceRules = get().config.rules.filter(
          (rule) => rule.source === source
        );
        const otherRules = get().config.rules.filter(
          (rule) => rule.source !== source
        );
        const [moved] = sourceRules.splice(fromIndex, 1);
        sourceRules.splice(toIndex, 0, moved);
        await applyAndPersist({
          ...get().config,
          rules: [...otherRules, ...sourceRules],
        });
      },
    };
  });
