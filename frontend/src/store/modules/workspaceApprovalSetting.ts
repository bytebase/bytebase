import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { create } from "@bufbuild/protobuf";
import { settingServiceClientConnect } from "@/grpcweb";
import type { Setting } from "@/types/proto-es/v1/setting_service_pb";
import { 
  GetSettingRequestSchema, 
  UpdateSettingRequestSchema,
  SettingSchema,
  Setting_SettingName,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { type LocalApprovalConfig, type LocalApprovalRule } from "@/types";
import type { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";
import {
  resolveLocalApprovalConfig,
  buildWorkspaceApprovalSetting,
  seedWorkspaceApprovalSetting,
} from "@/utils";

import { useGracefulRequest } from "./utils";

const SETTING_NAME = `settings/${Setting_SettingName[Setting_SettingName.WORKSPACE_APPROVAL]}`;

export const useWorkspaceApprovalSettingStore = defineStore(
  "workspaceApprovalSetting",
  () => {
    const config = ref<LocalApprovalConfig>({
      rules: [],
      parsed: [],
      unrecognized: [],
    });

    const setConfigSetting = async (setting: Setting) => {
      if (setting.value?.value?.case === "workspaceApprovalSettingValue") {
        const _config = setting.value.value.value;
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
        await setConfigSetting(response);
      } catch (ex) {
        console.error(ex);
      }
    };

    const updateConfig = async () => {
      const approvalSetting = await buildWorkspaceApprovalSetting(config.value);
      
      const setting = create(SettingSchema, {
        name: SETTING_NAME,
        value: create(SettingValueSchema, {
          value: {
            case: "workspaceApprovalSettingValue",
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