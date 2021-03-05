import axios from "axios";
import { Membership, MembershipState, ResourceObject } from "../../types";

function convert(membership: ResourceObject, rootGetters: any): Membership {
  const principal = rootGetters["principal/principalById"](
    membership.attributes.principalId
  );
  const updater = rootGetters["principal/principalById"](
    membership.attributes.updaterId
  );
  return {
    id: membership.id,
    principal,
    updater,
    ...(membership.attributes as Omit<
      Membership,
      "id" | "principal" | "updater"
    >),
  };
}

const state: () => MembershipState = () => ({
  membershipList: [],
});

const getters = {
  membershipList: (state: MembershipState) => () => {
    return state.membershipList;
  },
};

const actions = {
  async fetchMembershipList({ commit, rootGetters }: any) {
    const membershipList = (await axios.get(`/api/membership`)).data.data.map(
      (membership: ResourceObject) => {
        return convert(membership, rootGetters);
      }
    );

    commit("setMembershipList", membershipList);
    return membershipList;
  },
};

const mutations = {
  setMembershipList(state: MembershipState, membershipList: Membership[]) {
    state.membershipList = membershipList;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
