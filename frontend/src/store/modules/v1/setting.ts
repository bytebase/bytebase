import { defineStore } from "pinia";
import { computed } from "vue";
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { settingServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { 
  GetSettingRequestSchema, 
  UpdateSettingRequestSchema, 
  ListSettingsRequestSchema,
  type Setting as NewSetting
} from "@/types/proto-es/v1/setting_service_pb";
import { convertNewSettingToOld, convertOldSettingToNew, convertOldSettingNameToNew } from "@/utils/v1/setting-conversions";
import { settingNamePrefix } from "@/store/modules/v1/common";
import type {
  Setting,
  Value as SettingValue,
  WorkspaceProfileSetting,
  DataClassificationSetting_DataClassificationConfig,
} from "@/types/proto/v1/setting_service";
import { PasswordRestrictionSetting, Setting_SettingName as OldSettingName } from "@/types/proto/v1/setting_service";
import { useActuatorV1Store } from "./actuator";

// Re-export the old Setting_SettingName for compatibility
export { Setting_SettingName } from "@/types/proto/v1/setting_service";

interface SettingState {
  settingMapByName: Map<string, NewSetting>;
}

export const useSettingV1Store = defineStore("setting_v1", {
  state: (): SettingState => ({
    settingMapByName: new Map(),
  }),
  getters: {
    workspaceProfileSetting(state): WorkspaceProfileSetting | undefined {
      const newName = convertOldSettingNameToNew(OldSettingName.WORKSPACE_PROFILE);
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}${newName}`
      );
      if (!setting?.value?.value) return undefined;
      const value = setting.value.value;
      if (value.case === "workspaceProfileSettingValue") {
        // Convert new proto to old for compatibility
        const oldSetting = convertNewSettingToOld(setting);
        return oldSetting.value?.workspaceProfileSettingValue;
      }
      return undefined;
    },
    brandingLogo(state): string | undefined {
      const newName = convertOldSettingNameToNew(OldSettingName.BRANDING_LOGO);
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}${newName}`
      );
      if (!setting?.value?.value) return undefined;
      const value = setting.value.value;
      if (value.case === "stringValue") {
        return value.value;
      }
      return undefined;
    },
    classification(): DataClassificationSetting_DataClassificationConfig[] {
      const newName = convertOldSettingNameToNew(OldSettingName.DATA_CLASSIFICATION);
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${newName}`
      );
      if (!setting?.value?.value) return [];
      const value = setting.value.value;
      if (value.case === "dataClassificationSettingValue") {
        // Convert new proto to old for compatibility
        const oldSetting = convertNewSettingToOld(setting);
        return oldSetting.value?.dataClassificationSettingValue?.configs ?? [];
      }
      return [];
    },
    passwordRestriction(): PasswordRestrictionSetting {
      const newName = convertOldSettingNameToNew(OldSettingName.PASSWORD_RESTRICTION);
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${newName}`
      );
      if (!setting?.value?.value) {
        return PasswordRestrictionSetting.fromPartial({
          minLength: 8,
          requireLetter: true,
        });
      }
      const value = setting.value.value;
      if (value.case === "passwordRestrictionSetting") {
        // Convert new proto to old for compatibility
        const oldSetting = convertNewSettingToOld(setting);
        return oldSetting.value?.passwordRestrictionSetting ?? PasswordRestrictionSetting.fromPartial({
          minLength: 8,
          requireLetter: true,
        });
      }
      return PasswordRestrictionSetting.fromPartial({
        minLength: 8,
        requireLetter: true,
      });
    },
  },
  actions: {
    getProjectClassification(
      classificationId: string
    ): DataClassificationSetting_DataClassificationConfig | undefined {
      const newName = convertOldSettingNameToNew(OldSettingName.DATA_CLASSIFICATION);
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${newName}`
      );
      if (!setting?.value?.value) return undefined;
      const value = setting.value.value;
      if (value.case === "dataClassificationSettingValue") {
        // Convert new proto to old for compatibility
        const oldSetting = convertNewSettingToOld(setting);
        return oldSetting.value?.dataClassificationSettingValue?.configs.find(
          (config) => config.id === classificationId
        );
      }
      return undefined;
    },
    async fetchSettingByName(name: OldSettingName, silent = false): Promise<Setting | undefined> {
      try {
        const newName = convertOldSettingNameToNew(name);
        const request = create(GetSettingRequestSchema, {
          name: `${settingNamePrefix}${newName}`,
        });
        const response = await settingServiceClientConnect.getSetting(request, {
          contextValues: createContextValues().set(silentContextKey, silent),
        });
        this.settingMapByName.set(response.name, response);
        // Return old proto for compatibility
        return convertNewSettingToOld(response);
      } catch {
        return;
      }
    },
    getOrFetchSettingByName(name: OldSettingName, silent = false): Promise<Setting | undefined> | Setting | undefined {
      const setting = this.getSettingByName(name);
      if (setting) {
        return setting;
      }
      return this.fetchSettingByName(name, silent);
    },
    getSettingByName(name: OldSettingName): Setting | undefined {
      const newName = convertOldSettingNameToNew(name);
      const newSetting = this.settingMapByName.get(`${settingNamePrefix}${newName}`);
      if (!newSetting) return undefined;
      // Return old proto for compatibility
      return convertNewSettingToOld(newSetting);
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
      name: OldSettingName;
      value: SettingValue;
      validateOnly?: boolean;
      updateMask?: string[] | undefined;
    }): Promise<Setting> {
      // Convert old value to new proto-es format
      const newName = convertOldSettingNameToNew(name);
      const oldSetting = {
        name: `${settingNamePrefix}${newName}`,
        value,
      };
      const newSetting = convertOldSettingToNew(oldSetting);
      
      const request = create(UpdateSettingRequestSchema, {
        setting: newSetting,
        validateOnly,
        allowMissing: true,
        updateMask: updateMask ? { paths: updateMask } : undefined,
      });
      const response = await settingServiceClientConnect.updateSetting(request);
      this.settingMapByName.set(response.name, response);
      // Return old proto for compatibility
      return convertNewSettingToOld(response);
    },
    async updateWorkspaceProfile({
      payload,
      updateMask,
    }: {
      payload: Partial<WorkspaceProfileSetting>;
      updateMask: string[];
    }): Promise<void> {
      if (!this.workspaceProfileSetting) {
        return;
      }
      const profileSetting: WorkspaceProfileSetting = {
        ...this.workspaceProfileSetting,
        ...payload,
      };
      await this.upsertSetting({
        name: OldSettingName.WORKSPACE_PROFILE,
        value: {
          workspaceProfileSettingValue: profileSetting,
        },
        updateMask,
      });
      // Fetch the latest server info to refresh the disallow signup flag.
      await useActuatorV1Store().fetchServerInfo();
    },
  },
});

export const useSettingByName = (name: OldSettingName) => {
  const store = useSettingV1Store();
  const setting = computed(() => store.getSettingByName(name));
  store.getOrFetchSettingByName(name, /* silent */ true);
  return setting;
};
