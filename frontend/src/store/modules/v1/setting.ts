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
import { PasswordRestrictionSetting } from "@/types/proto/v1/setting_service";
import type { SettingName } from "@/types/setting";
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
        `${settingNamePrefix}bb.workspace.profile`
      );
      return setting?.value?.workspaceProfileSettingValue;
    },
    brandingLogo(state): string | undefined {
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}bb.branding.logo`
      );
      return setting?.value?.stringValue;
    },
    classification(): DataClassificationSetting_DataClassificationConfig[] {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}bb.workspace.data-classification`
      );
      return setting?.value?.dataClassificationSettingValue?.configs ?? [];
    },
    passwordRestriction(): PasswordRestrictionSetting {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}bb.workspace.password-restriction`
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
        `${settingNamePrefix}bb.workspace.data-classification`
      );
      return setting?.value?.dataClassificationSettingValue?.configs.find(
        (config) => config.id === classificationId
      );
    },
    async fetchSettingByName(name: SettingName, silent = false) {
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
    getOrFetchSettingByName(name: SettingName, silent = false) {
      const setting = this.getSettingByName(name);
      if (setting) {
        return setting;
      }
      return this.fetchSettingByName(name, silent);
    },
    getSettingByName(name: SettingName) {
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
      name: SettingName;
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
        name: "bb.workspace.profile",
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

export const useSettingByName = (name: SettingName) => {
  const store = useSettingV1Store();
  const setting = computed(() => store.getSettingByName(name));
  store.getOrFetchSettingByName(name, /* silent */ true);
  return setting;
};
