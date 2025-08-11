import { create } from "@bufbuild/protobuf";
import type { FieldMask } from "@bufbuild/protobuf/wkt";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { computed } from "vue";
import { settingServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { settingNamePrefix } from "@/store/modules/v1/common";
import {
  GetSettingRequestSchema,
  UpdateSettingRequestSchema,
  ListSettingsRequestSchema,
  type Setting,
  SettingSchema,
  Setting_SettingName,
  type WorkspaceProfileSetting,
  WorkspaceProfileSettingSchema,
  type DataClassificationSetting_DataClassificationConfig,
  type PasswordRestrictionSetting,
  PasswordRestrictionSettingSchema,
  type Value as SettingValue,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { useActuatorV1Store } from "./actuator";

interface SettingState {
  settingMapByName: Map<string, Setting>;
}

export const useSettingV1Store = defineStore("setting_v1", {
  state: (): SettingState => ({
    settingMapByName: new Map(),
  }),
  getters: {
    maximumResultRows(state): number {
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.SQL_RESULT_SIZE_LIMIT]}`
      );
      if (setting?.value?.value?.case === "sqlQueryRestrictionSetting") {
        const limit = setting.value.value.value.maximumResultRows ?? -1;
        if (limit <= 0) {
          return Number.MAX_VALUE;
        }
        return limit;
      }
      return Number.MAX_VALUE;
    },
    workspaceProfileSetting(state): WorkspaceProfileSetting | undefined {
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.WORKSPACE_PROFILE]}`
      );
      if (!setting?.value?.value) return undefined;
      const value = setting.value.value;
      if (value.case === "workspaceProfileSettingValue") {
        return value.value;
      }
      return undefined;
    },
    brandingLogo(state): string | undefined {
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.BRANDING_LOGO]}`
      );
      if (!setting?.value?.value) return undefined;
      const value = setting.value.value;
      if (value.case === "stringValue") {
        return value.value;
      }
      return undefined;
    },
    classification(): DataClassificationSetting_DataClassificationConfig[] {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.DATA_CLASSIFICATION]}`
      );
      if (!setting?.value?.value) return [];
      const value = setting.value.value;
      if (value.case === "dataClassificationSettingValue") {
        return value.value.configs;
      }
      return [];
    },
    passwordRestriction(): PasswordRestrictionSetting {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.PASSWORD_RESTRICTION]}`
      );
      if (!setting?.value?.value) {
        return create(PasswordRestrictionSettingSchema, {
          minLength: 8,
          requireLetter: true,
        });
      }
      const value = setting.value.value;
      if (value.case === "passwordRestrictionSetting") {
        return value.value;
      }
      return create(PasswordRestrictionSettingSchema, {
        minLength: 8,
        requireLetter: true,
      });
    },
  },
  actions: {
    getProjectClassification(
      classificationId: string
    ): DataClassificationSetting_DataClassificationConfig | undefined {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.DATA_CLASSIFICATION]}`
      );
      if (!setting?.value?.value) return undefined;
      const value = setting.value.value;
      if (value.case === "dataClassificationSettingValue") {
        return value.value.configs.find(
          (config) => config.id === classificationId
        );
      }
      return undefined;
    },
    async fetchSettingByName(
      name: Setting_SettingName,
      silent = false
    ): Promise<Setting | undefined> {
      try {
        const request = create(GetSettingRequestSchema, {
          name: `${settingNamePrefix}${Setting_SettingName[name]}`,
        });
        const response = await settingServiceClientConnect.getSetting(request, {
          contextValues: createContextValues().set(silentContextKey, silent),
        });
        this.settingMapByName.set(response.name, response);
        return response;
      } catch {
        return;
      }
    },
    getOrFetchSettingByName(
      name: Setting_SettingName,
      silent = false
    ): Promise<Setting | undefined> | Setting | undefined {
      const setting = this.getSettingByName(name);
      if (setting) {
        return setting;
      }
      return this.fetchSettingByName(name, silent);
    },
    getSettingByName(name: Setting_SettingName): Setting | undefined {
      return this.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[name]}`
      );
    },
    async fetchSettingList() {
      const request = create(ListSettingsRequestSchema, {});
      const response = await settingServiceClientConnect.listSettings(request);
      for (const setting of response.settings) {
        this.settingMapByName.set(setting.name, setting);
      }
    },
    async upsertSetting({
      name,
      value,
      validateOnly = false,
      updateMask,
    }: {
      name: Setting_SettingName;
      value: SettingValue;
      validateOnly?: boolean;
      updateMask?: FieldMask | undefined;
    }): Promise<Setting> {
      const setting = create(SettingSchema, {
        name: `${settingNamePrefix}${Setting_SettingName[name]}`,
        value,
      });

      const request = create(UpdateSettingRequestSchema, {
        setting,
        validateOnly,
        allowMissing: true,
        updateMask: updateMask,
      });
      const response = await settingServiceClientConnect.updateSetting(request);
      this.settingMapByName.set(response.name, response);
      return response;
    },
    async updateWorkspaceProfile({
      payload,
      updateMask,
    }: {
      payload: Partial<WorkspaceProfileSetting>;
      updateMask: FieldMask;
    }): Promise<void> {
      if (!this.workspaceProfileSetting) {
        return;
      }
      const profileSetting: WorkspaceProfileSetting = create(
        WorkspaceProfileSettingSchema,
        {
          ...this.workspaceProfileSetting,
          ...payload,
        }
      );
      await this.upsertSetting({
        name: Setting_SettingName.WORKSPACE_PROFILE,
        value: create(SettingValueSchema, {
          value: {
            case: "workspaceProfileSettingValue",
            value: profileSetting,
          },
        }),
        updateMask: updateMask,
      });
      // Fetch the latest server info to refresh the disallow signup flag.
      await useActuatorV1Store().fetchServerInfo();
    },
  },
});

export const useSettingByName = (name: Setting_SettingName) => {
  const store = useSettingV1Store();
  const setting = computed(() => store.getSettingByName(name));
  store.getOrFetchSettingByName(name, /* silent */ true);
  return setting;
};
