import axios from "axios";
import {
  Member,
  MemberState,
  RoleType,
  UserDisplay,
  ResourceObject,
} from "../../types";

function convert(member: ResourceObject): Member {
  return {
    id: member.id,
    ...(member.attributes as Omit<Member, "id">),
  };
}

const state: () => MemberState = () => ({
  memberList: [],
});

const getters = {
  memberList: (state: MemberState) => () => {
    return state.memberList;
  },
};

const actions = {
  async fetchMemberList({ commit }: any) {
    const memberList = (await axios.get(`/api/member`)).data.data.map(
      (member: ResourceObject) => {
        return convert(member);
      }
    );

    commit("setMemberList", memberList);
    return memberList;
  },
};

const mutations = {
  setMemberList(state: MemberState, memberList: Member[]) {
    state.memberList = memberList;
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
