import axios from "axios";
import {
  Membership,
  MembershipState,
  RoleType,
  UserDisplay,
  ResourceObject,
} from "../../types";

function convert(membership: ResourceObject): Membership {
  return {
    id: membership.id,
    ...(membership.attributes as Omit<Membership, "id">),
  };
}

const state: () => MembershipState = () => ({
  membershipList: [],
});

const getters = {
  membershipList: (state: MembershipState) => () => {
    return state.memberList;
  },
};

const actions = {
  async fetchMembershipList({ commit }: any) {
    const membershipList = (await axios.get(`/api/membership`)).data.data.map(
      (membership: ResourceObject) => {
        return convert(membership);
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
