import { create } from "@bufbuild/protobuf";
import type { FieldMask } from "@bufbuild/protobuf/wkt";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { computed } from "vue";
import { settingServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { settingNamePrefix } from "@/store/modules/v1/common";
import {
  type DataClassificationSetting_DataClassificationConfig,
  GetSettingRequestSchema,
  ListSettingsRequestSchema,
  type Setting,
  Setting_SettingName,
  SettingSchema,
  type SettingValue,
  SettingValueSchema,
  UpdateSettingRequestSchema,
  type WorkspaceProfileSetting,
  type WorkspaceProfileSetting_PasswordRestriction,
  WorkspaceProfileSetting_PasswordRestrictionSchema,
  WorkspaceProfileSettingSchema,
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
    workspaceProfileSetting(state): WorkspaceProfileSetting | undefined {
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.WORKSPACE_PROFILE]}`
      );
      if (!setting?.value?.value) return undefined;
      const value = setting.value.value;
      if (value.case === "workspaceProfile") {
        return value.value;
      }
      return undefined;
    },
    brandingLogo(): string | undefined {
      return this.workspaceProfileSetting?.brandingLogo;
    },
    classification(): DataClassificationSetting_DataClassificationConfig[] {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.DATA_CLASSIFICATION]}`
      );
      if (!setting?.value?.value) return [];
      const value = setting.value.value;
      if (value.case === "dataClassification") {
        return value.value.configs;
      }
      return [];
    },
    passwordRestriction(): WorkspaceProfileSetting_PasswordRestriction {
      return (
        this.workspaceProfileSetting?.passwordRestriction ??
        create(WorkspaceProfileSetting_PasswordRestrictionSchema, {
          minLength: 8,
          requireLetter: true,
        })
      );
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
      if (value.case === "dataClassification") {
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
            case: "workspaceProfile",
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
