import axios from "axios";
import { UserId, Activity, ActivityState } from "../../types";

const state: () => ActivityState = () => ({
  activityListByUser: new Map(),
});

const getters = {
  activityListByUser: (state: ActivityState) => (userId: UserId) => {
    return state.activityListByUser.get(userId);
  },
};

const actions = {
  async fetchActivityListForUser({ commit }: any, userId: UserId) {
    const activityList = (await axios.get(`/api/activity`)).data.data;
    commit("setActivityListForUser", { userId, activityList });
    return activityList;
  },
};

const mutations = {
  setActivityListForUser(
    state: ActivityState,
    {
      userId,
      activityList,
    }: {
      userId: UserId;
      activityList: Activity[];
    }
  ) {
    state.activityListByUser.set(userId, activityList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
