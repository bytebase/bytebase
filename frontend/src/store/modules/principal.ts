import axios from "axios";
import {
  User,
  PrincipalId,
  Principal,
  PrincipalState,
  ResourceObject,
} from "../../types";

const state: () => PrincipalState = () => ({
  principalList: [],
});

const getters = {
  principalList: (state: PrincipalState) => (): Principal[] => {
    return state.principalList;
  },

  principalById: (state: PrincipalState) => (
    principalId: PrincipalId
  ): Principal => {
    for (const principal of state.principalList) {
      if (principal.id == principalId) {
        return principal;
      }
    }
    return {
      id: principalId,
      name: "Unknown User " + principalId,
    };
  },
};

const actions = {
  async fetchPrincipalList({ commit }: any) {
    const userList: ResourceObject[] = (await axios.get(`/api/user`)).data.data;

    const principalList = userList.map((user) => {
      return {
        id: user.id,
        name: user.attributes.name,
      };
    });
    commit("setPrincipalList", principalList);

    return userList;
  },
};

const mutations = {
  setPrincipalList(state: PrincipalState, principalList: Principal[]) {
    state.principalList = principalList;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
