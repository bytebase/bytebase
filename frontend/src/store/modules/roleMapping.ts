import axios from "axios";
import {
  RoleMapping,
  RoleMappingNew,
  RoleMappingState,
  ResourceObject,
} from "../../types";

function convert(roleMapping: ResourceObject, rootGetters: any): RoleMapping {
  const principal = rootGetters["principal/principalById"](
    roleMapping.attributes.principalId
  );
  const updater = rootGetters["principal/principalById"](
    roleMapping.attributes.updaterId
  );
  return {
    id: roleMapping.id,
    principal,
    updater,
    ...(roleMapping.attributes as Omit<
      RoleMapping,
      "id" | "principal" | "updater"
    >),
  };
}

const state: () => RoleMappingState = () => ({
  roleMappingList: [],
});

const getters = {
  roleMappingList: (state: RoleMappingState) => () => {
    return state.roleMappingList;
  },
};

const actions = {
  async fetchRoleMappingList({ commit, rootGetters }: any) {
    const roleMappingList = (await axios.get(`/api/roleMapping`)).data.data.map(
      (roleMapping: ResourceObject) => {
        return convert(roleMapping, rootGetters);
      }
    );

    commit("setRoleMappingList", roleMappingList);
    return roleMappingList;
  },

  async createdRoleMapping(
    { commit, rootGetters }: any,
    newRoleMapping: RoleMappingNew
  ) {
    const createdRoleMapping = convert(
      (
        await axios.post(`/api/roleMapping`, {
          data: {
            type: "roleMapping",
            attributes: newRoleMapping,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("appendRoleMapping", createdRoleMapping);

    return createdRoleMapping;
  },
};

const mutations = {
  setRoleMappingList(state: RoleMappingState, roleMappingList: RoleMapping[]) {
    state.roleMappingList = roleMappingList;
  },

  appendRoleMapping(state: RoleMappingState, newRoleMapping: RoleMapping) {
    state.roleMappingList.push(newRoleMapping);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
