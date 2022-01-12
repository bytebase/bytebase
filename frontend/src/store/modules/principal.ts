import axios from "axios";
import {
  PrincipalId,
  Principal,
  PrincipalCreate,
  PrincipalPatch,
  PrincipalState,
  ResourceObject,
  unknown,
  empty,
  EMPTY_ID,
  PrincipalType,
  RoleType,
} from "../../types";
import { randomString } from "../../utils";

function convert(principal: ResourceObject): Principal {
  return {
    id: parseInt(principal.id),
    creatorId: principal.attributes.creatorId as PrincipalId,
    createdTs: principal.attributes.createdTs as number,
    updaterId: principal.attributes.updaterId as PrincipalId,
    updatedTs: principal.attributes.updatedTs as number,
    type: principal.attributes.type as PrincipalType,
    name: principal.attributes.name as string,
    email: principal.attributes.email as string,
    role: principal.attributes.role as RoleType,
  };
}

const state: () => PrincipalState = () => ({
  principalList: [],
});

const getters = {
  convert:
    (state: PrincipalState, getters: any, rootState: any, rootGetters: any) =>
    (principal: ResourceObject, includedList: ResourceObject[]): Principal => {
      return convert(principal);
    },

  principalList: (state: PrincipalState) => (): Principal[] => {
    return state.principalList;
  },

  principalById:
    (state: PrincipalState) =>
    (principalId: PrincipalId): Principal => {
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
  async fetchPrincipalList({ commit }: any) {
    const userList: ResourceObject[] = (await axios.get(`/api/principal`)).data
      .data;
    const principalList = userList.map((user) => {
      return convert(user);
    });

    commit("setPrincipalList", principalList);

    return principalList;
  },

  async fetchPrincipalById({ commit }: any, principalId: PrincipalId) {
    const principal = convert(
      (await axios.get(`/api/principal/${principalId}`)).data.data
    );

    commit("upsertPrincipalInList", principal);

    return principal;
  },

  // Returns existing user if already created.
  async createPrincipal({ commit }: any, newPrincipal: PrincipalCreate) {
    const createdPrincipal = convert(
      (
        await axios.post(`/api/principal`, {
          data: {
            type: "PrincipalCreate",
            attributes: {
              authProvider: newPrincipal.authProvider,
              name: newPrincipal.name,
              email: newPrincipal.email,
              password: randomString(),
            },
          },
        })
      ).data.data
    );

    commit("appendPrincipal", createdPrincipal);

    return createdPrincipal;
  },

  async patchPrincipal(
    { commit, dispatch }: any,
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
            type: "principalPatch",
            attributes: principalPatch,
          },
        })
      ).data.data
    );

    commit("upsertPrincipalInList", updatedPrincipal);

    dispatch("auth/refreshUserIfNeeded", updatedPrincipal.id, { root: true });

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
