import { defineStore } from "pinia";
import { computed } from "vue";
import { settingServiceClient } from "@/grpcweb";
import { settingNamePrefix } from "@/store/modules/v1/common";
import type {
  Setting,
  Value as SettingValue,
  WorkspaceProfileSetting,
  DataClassificationSetting_DataClassificationConfig,
} from "@/types/proto/v1/setting_service";
import { PasswordRestrictionSetting, Setting_SettingName } from "@/types/proto/v1/setting_service";
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
        `${settingNamePrefix}${Setting_SettingName.WORKSPACE_PROFILE}`
      );
      return setting?.value?.workspaceProfileSettingValue;
    },
    brandingLogo(state): string | undefined {
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName.BRANDING_LOGO}`
      );
      return setting?.value?.stringValue;
    },
    classification(): DataClassificationSetting_DataClassificationConfig[] {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName.DATA_CLASSIFICATION}`
      );
      return setting?.value?.dataClassificationSettingValue?.configs ?? [];
    },
    passwordRestriction(): PasswordRestrictionSetting {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName.PASSWORD_RESTRICTION}`
      );
      return (
        setting?.value?.passwordRestrictionSetting ??
        PasswordRestrictionSetting.fromPartial({
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
        `${settingNamePrefix}${Setting_SettingName.DATA_CLASSIFICATION}`
      );
      return setting?.value?.dataClassificationSettingValue?.configs.find(
        (config) => config.id === classificationId
      );
    },
    async fetchSettingByName(name: Setting_SettingName, silent = false) {
      try {
        const setting = await settingServiceClient.getSetting(
          {
            name: `${settingNamePrefix}${name}`,
          },
          { silent }
        );
        this.settingMapByName.set(setting.name, setting);
        return setting;
      } catch {
        return;
      }
    },
    getOrFetchSettingByName(name: Setting_SettingName, silent = false) {
      const setting = this.getSettingByName(name);
      if (setting) {
        return setting;
      }
      return this.fetchSettingByName(name, silent);
    },
    getSettingByName(name: Setting_SettingName) {
      return this.settingMapByName.get(`${settingNamePrefix}${name}`);
    },
    async fetchSettingList() {
      const { settings } = await settingServiceClient.listSettings({});
      for (const setting of settings) {
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
      updateMask?: string[] | undefined;
    }) {
      const resp = await settingServiceClient.updateSetting({
        setting: {
          name: `${settingNamePrefix}${name}`,
          value,
        },
        validateOnly,
        allowMissing: true,
        updateMask,
      });
      this.settingMapByName.set(resp.name, resp);
      return resp;
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
        name: Setting_SettingName.WORKSPACE_PROFILE,
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

export const useSettingByName = (name: Setting_SettingName) => {
  const store = useSettingV1Store();
  const setting = computed(() => store.getSettingByName(name));
  store.getOrFetchSettingByName(name, /* silent */ true);
  return setting;
};
