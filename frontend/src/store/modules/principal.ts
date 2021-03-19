import axios from "axios";
import {
  UNKNOWN_ID,
  PrincipalId,
  Principal,
  PrincipalNew,
  PrincipalPatch,
  PrincipalState,
  PrincipalStatus,
  ResourceObject,
} from "../../types";
import { isDevOrDemo, randomString } from "../../utils";

function convert(user: ResourceObject, rootGetters: any): Principal {
  const roleMapping = rootGetters["roleMapping/roleMappingByPrincipalId"](
    user.id
  );
  return {
    id: user.id,
    status: user.attributes.status as PrincipalStatus,
    name: user.attributes.name as string,
    email: user.attributes.email as string,
    role: roleMapping.role,
  };
}

const state: () => PrincipalState = () => ({
  principalList: [],
});

const getters = {
  principalList: (state: PrincipalState) => (): Principal[] => {
    return state.principalList;
  },

  principalByEmail: (state: PrincipalState) => (email: string): Principal => {
    if (!email) {
      return {
        id: UNKNOWN_ID,
        status: "UNKNOWN",
        name: "<<Email Missing>>",
        email: "",
        role: "GUEST",
      };
    }

    const principal = state.principalList.find((item) => item.email == email);
    if (principal) {
      return principal;
    }

    return {
      id: UNKNOWN_ID,
      status: "UNKNOWN",
      name: `<<Email ${email} not found>>`,
      email: "",
      role: "GUEST",
    };
  },

  principalById: (state: PrincipalState) => (
    principalId: PrincipalId
  ): Principal => {
    if (!principalId) {
      return {
        id: UNKNOWN_ID,
        status: "UNKNOWN",
        name: "<<ID Missing>>",
        email: "",
        role: "GUEST",
      };
    }

    const principal = state.principalList.find(
      (item) => item.id == principalId
    );
    if (principal) {
      return principal;
    }

    return {
      id: principalId,
      status: "UNKNOWN",
      name: `<<ID ${principalId} not found>>`,
      email: "",
      role: "GUEST",
    };
  },
};

const actions = {
  async fetchPrincipalList({ commit, rootGetters }: any) {
    const userList: ResourceObject[] = (await axios.get(`/api/user`)).data.data;

    const principalList = userList.map((user) => {
      return convert(user, rootGetters);
    });
    commit("setPrincipalList", principalList);

    return userList;
  },

  async fetchPrincipalById(
    { commit, rootGetters }: any,
    principalId: PrincipalId
  ) {
    const principal = convert(
      (await axios.get(`/api/user/${principalId}`)).data.data,
      rootGetters
    );

    commit("replacePrincipalInList", principal);

    return principal;
  },

  // Returns existing user if already created.
  async createPrincipal(
    { commit, rootGetters }: any,
    newPrincipal: PrincipalNew
  ) {
    const createdPrincipal = convert(
      (
        await axios.post(`/api/user`, {
          data: {
            type: "user",
            attributes: {
              status: "INVITED",
              email: newPrincipal.email,
              password: isDevOrDemo() ? "aaa" : randomString(),
            },
          },
        })
      ).data.data,
      rootGetters
    );

    commit("appendPrincipal", createdPrincipal);

    return createdPrincipal;
  },

  async patchPrincipal(
    { commit, rootGetters }: any,
    principal: PrincipalPatch
  ) {
    const { id, ...attrs } = principal;
    const updatedPrincipal = convert(
      (
        await axios.patch(`/api/user/${principal.id}`, {
          data: {
            type: "user",
            attributes: attrs,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("replacePrincipalInList", updatedPrincipal);

    return updatedPrincipal;
  },
};

const mutations = {
  setPrincipalList(state: PrincipalState, principalList: Principal[]) {
    state.principalList = principalList;
  },

  appendPrincipal(state: PrincipalState, newPrincipal: Principal) {
    state.principalList.push(newPrincipal);
  },

  replacePrincipalInList(state: PrincipalState, updatedPrincipal: Principal) {
    const i = state.principalList.findIndex(
      (item: Principal) => item.id == updatedPrincipal.id
    );
    if (i != -1) {
      state.principalList[i] = updatedPrincipal;
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
