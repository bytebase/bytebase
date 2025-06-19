import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { create } from "@bufbuild/protobuf";
import { settingServiceClientConnect } from "@/grpcweb";
import { 
  GetSettingRequestSchema, 
  UpdateSettingRequestSchema
} from "@/types/proto-es/v1/setting_service_pb";
import { convertNewSettingToOld, convertOldSettingToNew, convertOldSettingNameToNew } from "@/utils/v1/setting-conversions";
import { type LocalApprovalConfig, type LocalApprovalRule } from "@/types";
import type { Risk_Source } from "@/types/proto/v1/risk_service";
import type { Setting } from "@/types/proto/v1/setting_service";
import { Setting_SettingName as OldSettingName } from "@/types/proto/v1/setting_service";
import {
  resolveLocalApprovalConfig,
  buildWorkspaceApprovalSetting,
  seedWorkspaceApprovalSetting,
} from "@/utils";
import { useGracefulRequest } from "./utils";

const newName = convertOldSettingNameToNew(OldSettingName.WORKSPACE_APPROVAL);
const SETTING_NAME = `settings/${newName}`;

export const useWorkspaceApprovalSettingStore = defineStore(
  "workspaceApprovalSetting",
  () => {
    const config = ref<LocalApprovalConfig>({
      rules: [],
      parsed: [],
      unrecognized: [],
    });

    const setConfigSetting = async (setting: Setting) => {
      const _config = setting.value?.workspaceApprovalSettingValue;
      if (_config) {
        if (_config.rules.length === 0) {
          _config.rules.push(...seedWorkspaceApprovalSetting());
        }
        config.value = await resolveLocalApprovalConfig(_config);
      }
    };

    const fetchConfig = async () => {
      try {
        const request = create(GetSettingRequestSchema, {
          name: SETTING_NAME,
        });
        const response = await settingServiceClientConnect.getSetting(request);
        // Convert to old format for compatibility with setConfigSetting
        const setting = convertNewSettingToOld(response);
        await setConfigSetting(setting);
      } catch (ex) {
        console.error(ex);
      }
    };

    const updateConfig = async () => {
      const setting = await buildWorkspaceApprovalSetting(config.value);
      // Create old setting object and convert to new format
      const oldSetting = {
        name: SETTING_NAME,
        value: {
          workspaceApprovalSettingValue: setting,
        },
      };
      const newSetting = convertOldSettingToNew(oldSetting);
      
      const request = create(UpdateSettingRequestSchema, {
        allowMissing: true,
        setting: newSetting,
      });
      await settingServiceClientConnect.updateSetting(request);
    };

    const useBackupAndUpdateConfig = async (update: () => Promise<any>) => {
      const backup = cloneDeep(config.value);
      try {
        await useGracefulRequest(update);
      } catch (err) {
        config.value = backup;
        throw err;
      }
    };

    const upsertRule = async (
      newRule: LocalApprovalRule,
      oldRule: LocalApprovalRule | undefined
    ) => {
      await useBackupAndUpdateConfig(async () => {
        const { rules } = config.value;
        if (oldRule) {
          const index = rules.indexOf(oldRule);
          if (index >= 0) {
            rules[index] = newRule;
            await updateConfig();
          }
        } else {
          rules.unshift(newRule);
          await updateConfig();
        }
      });
    };

    const deleteRule = async (rule: LocalApprovalRule) => {
      await useBackupAndUpdateConfig(async () => {
        const { rules, parsed, unrecognized } = config.value;
        config.value.parsed = parsed.filter((item) => item.rule !== rule.uid);
        config.value.unrecognized = unrecognized.filter(
          (item) => item.rule !== rule.uid
        );
        const index = rules.indexOf(rule);
        if (index >= 0) {
          rules.splice(index, 1);
        }
        await updateConfig();
      });
    };

    const updateRuleFlow = async (
      source: Risk_Source,
      level: number,
      rule: string | undefined
    ) => {
      await useBackupAndUpdateConfig(async () => {
        const { parsed } = config.value;
        const index = parsed.findIndex(
          (item) => item.source == source && item.level === level
        );
        if (index >= 0) {
          if (rule) {
            parsed[index].rule = rule;
          } else {
            parsed.splice(index, 1);
          }
        } else {
          if (rule) {
            parsed.push({
              source,
              level,
              rule,
            });
          }
        }
        await updateConfig();
      });
    };

    return {
      config,
      fetchConfig,
      upsertRule,
      deleteRule,
      updateRuleFlow,
    };
  }
);
