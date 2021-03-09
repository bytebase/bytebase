import axios from "axios";
import {
  RoleMapping,
  RoleMappingId,
  RoleMappingNew,
  RoleMappingPatch,
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
  roleMappingByEmail: (state: RoleMappingState) => (
    email: string
  ): RoleMapping | undefined => {
    return state.roleMappingList.find((item) => item.principal.email == email);
  },
};

const actions = {
  async fetchRoleMappingList({ commit, rootGetters }: any) {
    const roleMappingList = (await axios.get(`/api/rolemapping`)).data.data.map(
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
        await axios.post(`/api/rolemapping`, {
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

  async patchRoleMapping(
    { commit, rootGetters }: any,
    roleMapping: RoleMappingPatch
  ) {
    const { id, ...attrs } = roleMapping;
    const updatedRoleMapping = convert(
      (
        await axios.patch(`/api/rolemapping/${roleMapping.id}`, {
          data: {
            type: "roleMapping",
            attributes: attrs,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("replaceRoleMappingInList", updatedRoleMapping);

    return updatedRoleMapping;
  },

  async deleteRoleMappingById(
    { state, commit }: { state: RoleMappingState; commit: any },
    id: RoleMappingId
  ) {
    await axios.delete(`/api/rolemapping/${id}`);

    const newList = state.roleMappingList.filter((item: RoleMapping) => {
      return item.id != id;
    });

    commit("setRoleMappingList", newList);
  },
};

const mutations = {
  setRoleMappingList(state: RoleMappingState, roleMappingList: RoleMapping[]) {
    state.roleMappingList = roleMappingList;
  },

  appendRoleMapping(state: RoleMappingState, newRoleMapping: RoleMapping) {
    state.roleMappingList.push(newRoleMapping);
  },

  replaceRoleMappingInList(
    state: RoleMappingState,
    updatedRoleMapping: RoleMapping
  ) {
    const i = state.roleMappingList.findIndex(
      (item: RoleMapping) => item.id == updatedRoleMapping.id
    );
    if (i != -1) {
      state.roleMappingList[i] = updatedRoleMapping;
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
