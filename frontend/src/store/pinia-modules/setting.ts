import axios from "axios";
import { defineStore } from "pinia";
import { ResourceObject, SettingState } from "@/types";
import { Setting, SettingName } from "@/types/setting";
import { getPrincipalFromIncludedList } from "../modules/principal";

function convert(
  setting: ResourceObject,
  includedList: ResourceObject[]
): Setting {
  return {
    ...(setting.attributes as Omit<Setting, "id" | "creator" | "updater">),
    id: parseInt(setting.id),
    creator: getPrincipalFromIncludedList(
      setting.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      setting.relationships!.updater.data,
      includedList
    ),
  };
}

export const useSettingStore = defineStore("setting", {
  state: (): SettingState => ({
    settingByName: new Map(),
  }),
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
  },
});
