import axios from "axios";
import {
  PrincipalId,
  Principal,
  PrincipalNew,
  PrincipalPatch,
  PrincipalState,
  PrincipalStatus,
  ResourceObject,
  unknown,
  empty,
  EMPTY_ID,
  PrincipalType,
} from "../../types";
import { isDevOrDemo, randomString } from "../../utils";

function convert(principal: ResourceObject, rootGetters: any): Principal {
  const member = rootGetters["member/memberByPrincipalId"](principal.id);
  return {
    id: principal.id,
    creatorId: principal.attributes.creatorId as PrincipalId,
    createdTs: principal.attributes.createdTs as number,
    updaterId: principal.attributes.updaterId as PrincipalId,
    updatedTs: principal.attributes.updatedTs as number,
    status: principal.attributes.status as PrincipalStatus,
    type: principal.attributes.type as PrincipalType,
    name: principal.attributes.name as string,
    email: principal.attributes.email as string,
    role: member.role,
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
    return (
      state.principalList.find((item) => item.email == email) ||
      (unknown("PRINCIPAL") as Principal)
    );
  },

  principalById: (state: PrincipalState) => (
    principalId: PrincipalId
  ): Principal => {
    if (principalId == EMPTY_ID) {
      return empty("PRINCIPAL") as Principal;
    }

    return (
      state.principalList.find((item) => item.id == principalId) ||
      (unknown("PRINCIPAL") as Principal)
    );
  },
};

const actions = {
  async fetchPrincipalList({ state, commit, rootGetters }: any) {
    const userList: ResourceObject[] = (await axios.get(`/api/principal`)).data
      .data;
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
      (await axios.get(`/api/principal/${principalId}`)).data.data,
      rootGetters
    );

    commit("upsertPrincipalInList", principal);

    return principal;
  },

  // Returns existing user if already created.
  async createPrincipal(
    { commit, rootGetters }: any,
    newPrincipal: PrincipalNew
  ) {
    const createdPrincipal = convert(
      (
        await axios.post(`/api/principal`, {
          data: {
            type: "principalnew",
            attributes: {
              status: "INVITED",
              name: newPrincipal.name,
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
    {
      principalId,
      principalPatch,
    }: {
      principalId: PrincipalId;
      principalPatch: PrincipalPatch;
    }
  ) {
    const updatedPrincipal = convert(
      (
        await axios.patch(`/api/principal/${principalId}`, {
          data: {
            type: "principalpatch",
            attributes: principalPatch,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("upsertPrincipalInList", updatedPrincipal);

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

  upsertPrincipalInList(state: PrincipalState, updatedPrincipal: Principal) {
    const i = state.principalList.findIndex(
      (item: Principal) => item.id == updatedPrincipal.id
    );
    if (i == -1) {
      state.principalList.push(updatedPrincipal);
    } else {
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
