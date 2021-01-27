import axios from "axios";
import { UserId, Group, GroupState } from "../../types";

const state: () => GroupState = () => ({
  groupListByUser: new Map(),
});

const getters = {
  groupListByUser: (state: GroupState) => (userId: UserId) => {
    return state.groupListByUser.get(userId);
  },
};

const actions = {
  async fetchGroupListForUser({ commit }: any, userId: UserId) {
    const groupList = (await axios.get(`/api/group?userid=${userId}`)).data
      .data;
    commit("setGroupListForUser", { userId, groupList });
    return groupList;
  },
};

const mutations = {
  setGroupListForUser(
    state: GroupState,
    {
      userId,
      groupList,
    }: {
      userId: UserId;
      groupList: Group[];
    }
  ) {
    state.groupListByUser.set(userId, groupList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
