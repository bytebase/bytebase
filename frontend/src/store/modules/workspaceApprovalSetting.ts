import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { v4 as uuidv4 } from "uuid";
import { ref } from "vue";
import { settingServiceClientConnect } from "@/grpcweb";
import {
  getBuiltinFlow,
  isBuiltinFlowId,
  type LocalApprovalConfig,
  type LocalApprovalRule,
} from "@/types";
import type { ApprovalTemplate } from "@/types/proto-es/v1/issue_service_pb";
import {
  ApprovalFlowSchema,
  ApprovalTemplateSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";
import type { Setting } from "@/types/proto-es/v1/setting_service_pb";
import {
  GetSettingRequestSchema,
  Setting_SettingName,
  SettingSchema,
  ValueSchema as SettingValueSchema,
  UpdateSettingRequestSchema,
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
      parsed: [],
      unrecognized: [],
    });

    const setConfigSetting = async (setting: Setting) => {
      if (setting.value?.value?.case === "workspaceApprovalSettingValue") {
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

        // Ensure new rule has an id (for custom rules, generate UUID if not provided)
        if (!newRule.template.id) {
          newRule.template.id = uuidv4();
        }

        if (oldRule) {
          const index = rules.indexOf(oldRule);
          if (index >= 0) {
            rules[index] = newRule;
          }
        } else {
          rules.unshift(newRule);
        }

        // Note: No explicit cleanup needed here.
        // buildWorkspaceApprovalSetting will save:
        // - All custom templates (even if unused)
        // - Only used built-in templates
        await updateConfig();
      });
    };

    const deleteRule = async (rule: LocalApprovalRule) => {
      await useBackupAndUpdateConfig(async () => {
        const { rules, parsed, unrecognized } = config.value;

        // Remove this template from parsed and unrecognized
        config.value.parsed = parsed.filter(
          (item) => item.rule !== rule.template.id
        );
        config.value.unrecognized = unrecognized.filter(
          (item) => item.rule !== rule.template.id
        );

        // Remove the template from rules
        const index = rules.indexOf(rule);
        if (index >= 0) {
          rules.splice(index, 1);
        }

        // Note: No explicit cleanup needed here.
        // buildWorkspaceApprovalSetting will save:
        // - All custom templates (even if unused)
        // - Only used built-in templates
        await updateConfig();
      });
    };

    // Helper: Ensure a template exists in the local cache for a given flow ID
    // This is used for UI display and editing. The actual save is handled by buildWorkspaceApprovalSetting.
    const getOrCreateTemplate = (flowId: string): LocalApprovalRule => {
      const { rules } = config.value;

      // Check if template already exists
      const existingRule = rules.find((rule) => rule.template.id === flowId);
      if (existingRule) {
        return existingRule;
      }

      // Template doesn't exist, create it
      let template: ApprovalTemplate;

      if (isBuiltinFlowId(flowId)) {
        // Built-in flow: get from constants
        const builtinFlow = getBuiltinFlow(flowId);
        if (!builtinFlow) {
          throw new Error(`Unknown built-in flow: ${flowId}`);
        }
        template = create(ApprovalTemplateSchema, {
          id: builtinFlow.id,
          title: builtinFlow.title,
          description: builtinFlow.description,
          flow: create(ApprovalFlowSchema, {
            roles: [...builtinFlow.roles],
          }),
        });
      } else {
        // Custom flow: should already exist, but create a placeholder if not
        template = create(ApprovalTemplateSchema, {
          id: flowId,
          title: "Custom Flow",
          description: "",
          flow: create(ApprovalFlowSchema, { roles: [] }),
        });
      }

      const newRule: LocalApprovalRule = { template };
      rules.push(newRule);
      return newRule;
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

        // Ensure template exists in local cache (for UI display)
        if (rule) {
          getOrCreateTemplate(rule);
        }

        // Update or remove the parsed rule
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

        // Note: No explicit cleanup needed here.
        // buildWorkspaceApprovalSetting will save:
        // - All custom templates (even if unused)
        // - Only used built-in templates
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
