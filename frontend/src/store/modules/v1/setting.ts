import { defineStore } from "pinia";
import { settingServiceClient } from "@/grpcweb";
import {
  Setting,
  Value as SettingValue,
  WorkspaceProfileSetting,
} from "@/types/proto/v1/setting_service";
import { settingNamePrefix } from "@/store/modules/v1/common";
import { SettingName } from "@/types/setting";
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
  },
  actions: {
    async fetchSettingByName(name: SettingName, silent = false) {
      const setting = await settingServiceClient.getSetting(
        {
          name: `${settingNamePrefix}${name}`,
        },
        {
          silent,
        }
      );
      this.settingMapByName.set(setting.name, setting);
      return setting;
    },
    getOrFetchSettingByName(name: SettingName) {
      const setting = this.getSettingByName(name);
      if (setting) {
        return setting;
      }
      return this.fetchSettingByName(name);
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
    }: {
      name: SettingName;
      value: SettingValue;
      validateOnly?: boolean;
    }) {
      const resp = await settingServiceClient.setSetting({
        setting: {
          name: `${settingNamePrefix}${name}`,
          value,
        },
        validateOnly,
      });
      this.settingMapByName.set(resp.name, resp);
      return resp;
    },
    async updateWorkspaceProfile(
      payload: Partial<WorkspaceProfileSetting>
    ): Promise<void> {
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
      });
      // Fetch the latest server info to refresh the disallow signup flag.
      await useActuatorV1Store().fetchServerInfo();
    },
  },
});
