import axios from "axios";
import { cloneDeep } from "lodash";
import {
  UNKNOWN_ID,
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
  SYSTEM_BOT_ID,
} from "../../types";
import { isDevOrDemo, randomString } from "../../utils";

function convert(principal: ResourceObject, rootGetters: any): Principal {
  const creator = rootGetters["principal/principalById"](
    principal.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    principal.attributes.updaterId
  );

  const member = rootGetters["member/memberByPrincipalId"](principal.id);
  return {
    id: principal.id,
    creator,
    updater,
    createdTs: principal.attributes.createdTs as number,
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
    const userList: ResourceObject[] = (await axios.get(`/mock/principal`)).data
      .data;

    // Dealing with bootstraping here. The system bot is the first user and we
    // assign itself as the creator/updater.
    // We also convert the system bot alone first so that the remaining users
    // can convert their creater/updater properly.
    const systemBot = userList.find((item) => item.id == SYSTEM_BOT_ID);
    if (systemBot) {
      const theBot = convert(systemBot, rootGetters);
      // Use cloneDeep to avoid infinite recursion during JSON.stringify
      theBot.creator = cloneDeep(theBot);
      theBot.updater = cloneDeep(theBot);
      commit("upsertPrincipalInList", theBot);
    }

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
      (await axios.get(`/mock/principal/${principalId}`)).data.data,
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
        await axios.post(`/mock/principal`, {
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
        await axios.patch(`/mock/principal/${principalId}`, {
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
