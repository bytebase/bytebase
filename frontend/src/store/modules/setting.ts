import axios from "axios";
import {
  Principal,
  ResourceIdentifier,
  ResourceObject,
  SettingState,
} from "../../types";
import { Setting, SettingName } from "../../types/setting";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  setting: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Setting {
  const creatorId = (setting.relationships!.creator.data as ResourceIdentifier)
    .id;
  const updaterId = (setting.relationships!.updater.data as ResourceIdentifier)
    .id;
  return {
    ...(setting.attributes as Omit<Setting, "id" | "creator" | "updater">),
    id: parseInt(setting.id),
    creator: getPrincipalFromIncludedList(creatorId, includedList) as Principal,
    updater: getPrincipalFromIncludedList(updaterId, includedList) as Principal,
  };
}

const state: () => SettingState = () => ({
  settingByName: new Map(),
});

const getters = {
  settingByName:
    (state: SettingState) =>
    (name: SettingName): Setting | undefined => {
      return state.settingByName.get(name);
    },
};

const actions = {
  async fetchSetting({ commit, rootGetters }: any): Promise<Setting[]> {
    const data = (await axios.get(`/api/setting`)).data;
    const settingList = data.data.map((setting: ResourceObject) => {
      return convert(setting, data.included, rootGetters);
    });
    for (const setting of settingList) {
      commit("setSettingByName", { name: setting.name, setting });
    }
    return settingList;
  },

  async updateSettingByName(
    { commit, rootGetters }: any,
    { name, value }: { name: SettingName; value: string }
  ): Promise<Setting> {
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

    const setting = convert(data.data, data.included, rootGetters);

    commit("setSettingByName", { name: setting.name, setting });

    return setting;
  },
};

const mutations = {
  setSettingByName(
    state: SettingState,
    {
      name,
      setting,
    }: {
      name: SettingName;
      setting: Setting;
    }
  ) {
    state.settingByName.set(name, setting);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
