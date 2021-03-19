import axios from "axios";
import {
  RoleMapping,
  RoleMappingId,
  RoleMappingNew,
  RoleMappingPatch,
  RoleMappingState,
  ResourceObject,
  PrincipalId,
} from "../../types";

function convert(roleMapping: ResourceObject): RoleMapping {
  return {
    id: roleMapping.id,
    ...(roleMapping.attributes as Omit<RoleMapping, "id">),
  };
}

const state: () => RoleMappingState = () => ({
  roleMappingList: [],
});

const getters = {
  roleMappingList: (state: RoleMappingState) => (): RoleMapping[] => {
    return state.roleMappingList;
  },
  roleMappingByPrincipalId: (state: RoleMappingState) => (
    id: PrincipalId
  ): RoleMapping => {
    const ts = Date.now();
    return (
      state.roleMappingList.find((item) => item.principalId == id) || {
        id: "-1",
        createdTs: ts,
        lastUpdatedTs: ts,
        role: "GUEST",
        principalId: "-1",
        updaterId: "-1",
      }
    );
  },
};

const actions = {
  async fetchRoleMappingList({ commit }: any) {
    const roleMappingList = (await axios.get(`/api/rolemapping`)).data.data.map(
      (roleMapping: ResourceObject) => {
        return convert(roleMapping);
      }
    );

    commit("setRoleMappingList", roleMappingList);
    return roleMappingList;
  },

  async createdRoleMapping({ commit }: any, newRoleMapping: RoleMappingNew) {
    const createdRoleMapping = convert(
      (
        await axios.post(`/api/rolemapping`, {
          data: {
            type: "roleMapping",
            attributes: newRoleMapping,
          },
        })
      ).data.data
    );

    commit("appendRoleMapping", createdRoleMapping);

    return createdRoleMapping;
  },

  async patchRoleMapping({ commit }: any, roleMapping: RoleMappingPatch) {
    const { id, ...attrs } = roleMapping;
    const updatedRoleMapping = convert(
      (
        await axios.patch(`/api/rolemapping/${roleMapping.id}`, {
          data: {
            type: "roleMapping",
            attributes: attrs,
          },
        })
      ).data.data
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
