import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { v4 as uuidv4 } from "uuid";
import { ref } from "vue";
import { settingServiceClientConnect } from "@/connect";
import type { LocalApprovalConfig, LocalApprovalRule } from "@/types";
import type { Setting } from "@/types/proto-es/v1/setting_service_pb";
import {
  GetSettingRequestSchema,
  Setting_SettingName,
  SettingSchema,
  SettingValueSchema as SettingSettingValueSchema,
  UpdateSettingRequestSchema,
  WorkspaceApprovalSetting_Rule_Source,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  buildWorkspaceApprovalSetting,
  resolveLocalApprovalConfig,
} from "@/utils";
import { useGracefulRequest } from "./utils";

const SETTING_NAME = `settings/${Setting_SettingName[Setting_SettingName.WORKSPACE_APPROVAL]}`;

export const useWorkspaceApprovalSettingStore = defineStore(
  "workspaceApprovalSetting",
  () => {
    const config = ref<LocalApprovalConfig>({
      rules: [],
    });

    const setConfigSetting = async (setting: Setting) => {
      if (setting.value?.value?.case === "workspaceApproval") {
        const _config = setting.value.value.value;
        config.value = await resolveLocalApprovalConfig(_config);
      }
    };

    const fetchConfig = async () => {
      try {
        const request = create(GetSettingRequestSchema, {
          name: SETTING_NAME,
        });
        const response = await settingServiceClientConnect.getSetting(request);
        await setConfigSetting(response);
      } catch (ex) {
        console.error(ex);
      }
    };

    const updateConfig = async () => {
      const approvalSetting = await buildWorkspaceApprovalSetting(config.value);

      const setting = create(SettingSchema, {
        name: SETTING_NAME,
        value: create(SettingSettingValueSchema, {
          value: {
            case: "workspaceApproval",
            value: approvalSetting,
          },
        }),
      });

      const request = create(UpdateSettingRequestSchema, {
        allowMissing: true,
        setting,
      });
      await settingServiceClientConnect.updateSetting(request);
    };

    const useBackupAndUpdateConfig = async (update: () => Promise<void>) => {
      const backup = cloneDeep(config.value);
      try {
        await useGracefulRequest(update);
      } catch (err) {
        config.value = backup;
        throw err;
      }
    };

    // Get rules for a specific source
    const getRulesBySource = (
      source: WorkspaceApprovalSetting_Rule_Source
    ): LocalApprovalRule[] => {
      return config.value.rules.filter((r) => r.source === source);
    };

    // Add a new rule
    const addRule = async (rule: Omit<LocalApprovalRule, "uid">) => {
      await useBackupAndUpdateConfig(async () => {
        const newRule: LocalApprovalRule = {
          ...rule,
          uid: uuidv4(),
        };
        config.value.rules.push(newRule);
        await updateConfig();
      });
    };

    // Update an existing rule
    const updateRule = async (
      uid: string,
      updates: Partial<LocalApprovalRule>
    ) => {
      await useBackupAndUpdateConfig(async () => {
        const index = config.value.rules.findIndex((r) => r.uid === uid);
        if (index >= 0) {
          config.value.rules[index] = {
            ...config.value.rules[index],
            ...updates,
          };
          await updateConfig();
        }
      });
    };

    // Delete a rule
    const deleteRule = async (uid: string) => {
      await useBackupAndUpdateConfig(async () => {
        const index = config.value.rules.findIndex((r) => r.uid === uid);
        if (index >= 0) {
          config.value.rules.splice(index, 1);
          await updateConfig();
        }
      });
    };

    // Reorder rules (for drag-and-drop within a source)
    const reorderRules = async (
      source: WorkspaceApprovalSetting_Rule_Source,
      fromIndex: number,
      toIndex: number
    ) => {
      await useBackupAndUpdateConfig(async () => {
        // Get all rules for this source
        const sourceRules = config.value.rules.filter(
          (r) => r.source === source
        );
        const otherRules = config.value.rules.filter(
          (r) => r.source !== source
        );

        // Reorder within source rules
        const [moved] = sourceRules.splice(fromIndex, 1);
        sourceRules.splice(toIndex, 0, moved);

        // Rebuild config with reordered rules
        config.value.rules = [...otherRules, ...sourceRules];
        await updateConfig();
      });
    };

    return {
      config,
      fetchConfig,
      getRulesBySource,
      addRule,
      updateRule,
      deleteRule,
      reorderRules,
    };
  }
);
