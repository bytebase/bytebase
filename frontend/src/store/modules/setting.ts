import axios from "axios";
import { ResourceObject, SettingState } from "../../types";
import { Setting, SettingName } from "../../types/setting";

function convert(setting: ResourceObject, rootGetters: any): Setting {
  return {
    ...(setting.attributes as Omit<Setting, "id">),
    id: parseInt(setting.id),
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
    const settingList = (await axios.get(`/api/setting`)).data.data.map(
      (setting: ResourceObject) => {
        return convert(setting, rootGetters);
      }
    );
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
    ).data.data;

    const setting = convert(data, rootGetters);

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
