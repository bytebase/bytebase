import axios from "axios";
import { Member, MemberState } from "../../types";

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
    const memberList = (await axios.get(`/api/member`)).data.data;
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
