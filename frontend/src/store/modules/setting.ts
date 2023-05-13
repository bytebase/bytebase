import { computed, unref } from "vue";
import { defineStore } from "pinia";
import axios from "axios";
import { MaybeRef, ResourceObject, SettingState } from "@/types";
import { Setting, SettingName } from "@/types/setting";
import { WorkspaceProfileSetting } from "@/types/proto/store/setting";
import { useActuatorStore } from "./actuator";

function convert(
  setting: ResourceObject,
  includedList: ResourceObject[]
): Setting {
  return {
    ...(setting.attributes as Omit<Setting, "id">),
    id: parseInt(setting.id),
  };
}

export const useSettingStore = defineStore("setting", {
  state: (): SettingState => ({
    settingByName: new Map(),
  }),
  getters: {
    workspaceSetting(state): WorkspaceProfileSetting | undefined {
      const setting = state.settingByName.get("bb.workspace.profile");
      if (!setting || !setting.value) {
        return;
      }

      const profileSetting = WorkspaceProfileSetting.fromJSON(
        JSON.parse(setting.value)
      );
      if (profileSetting.gitopsWebhookUrl === "") {
        profileSetting.gitopsWebhookUrl = profileSetting.externalUrl;
      }
      return profileSetting;
    },
  },
  actions: {
    getSettingByName(name: SettingName) {
      return this.settingByName.get(name);
    },
    setSettingByName({
      name,
      setting,
    }: {
      name: SettingName;
      setting: Setting;
    }) {
      this.settingByName.set(name, setting);
    },
    async fetchSetting(): Promise<Setting[]> {
      const data = (await axios.get(`/api/setting`)).data;
      const settingList = data.data.map((setting: ResourceObject) => {
        return convert(setting, data.included);
      });
      for (const setting of settingList) {
        this.setSettingByName({ name: setting.name, setting });
      }
      return settingList;
    },
    async updateSettingByName({
      name,
      value,
    }: {
      name: SettingName;
      value: string;
    }): Promise<Setting> {
      const data = (
        await axios.patch(`/api/setting/${name}`, {
          data: {
            type: "settingPatch",
            attributes: {
              value,
            },
          },
        })
      ).data;

      const setting = convert(data.data, data.included);
      this.setSettingByName({ name: setting.name, setting });

      return setting;
    },
    async updateWorkspaceProfile(payload: object): Promise<void> {
      if (!this.workspaceSetting) {
        return;
      }
      const profileSetting: WorkspaceProfileSetting = {
        disallowSignup: this.workspaceSetting.disallowSignup,
        externalUrl: this.workspaceSetting.externalUrl,
        require2fa: this.workspaceSetting.require2fa,
        outboundIpList: this.workspaceSetting.outboundIpList,
        gitopsWebhookUrl: this.workspaceSetting.gitopsWebhookUrl,
        ...payload,
      };
      await this.updateSettingByName({
        name: "bb.workspace.profile",
        value: JSON.stringify(profileSetting),
      });
      // Fetch the latest server info.
      await useActuatorStore().fetchServerInfo();
    },
  },
});

export const useSettingByName = (name: MaybeRef<SettingName>) => {
  const store = useSettingStore();

  return computed(() => {
    return store.getSettingByName(unref(name));
  });
};
