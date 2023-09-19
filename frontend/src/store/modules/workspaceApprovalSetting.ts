import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { settingServiceClient } from "@/grpcweb";
import type { LocalApprovalConfig, LocalApprovalRule } from "@/types";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { Setting } from "@/types/proto/v1/setting_service";
import {
  resolveLocalApprovalConfig,
  buildWorkspaceApprovalSetting,
  seedWorkspaceApprovalSetting,
} from "@/utils";
import { useGracefulRequest } from "./utils";

const SETTING_NAME = "settings/bb.workspace.approval";

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
        const setting = await settingServiceClient.getSetting({
          name: SETTING_NAME,
        });
        await setConfigSetting(setting);
      } catch (ex) {
        console.error(ex);
      }
    };

    const updateConfig = async () => {
      const setting = await buildWorkspaceApprovalSetting(config.value);
      await settingServiceClient.setSetting({
        setting: {
          name: SETTING_NAME,
          value: {
            workspaceApprovalSettingValue: setting,
          },
        },
      });
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
