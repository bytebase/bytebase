import axios from "axios";
import { UserId, TaskId, Activity, ActivityState } from "../../types";

const state: () => ActivityState = () => ({
  activityListByUser: new Map(),
  activityListByTask: new Map(),
});

const getters = {
  activityListByUser: (state: ActivityState) => (userId: UserId) => {
    return state.activityListByUser.get(userId);
  },
  activityListByTask: (state: ActivityState) => (taskId: TaskId) => {
    return state.activityListByTask.get(taskId);
  },
};

const actions = {
  async fetchActivityListForUser({ commit }: any, userId: UserId) {
    const activityList = (await axios.get(`/api/activity`)).data.data;
    commit("setActivityListForUser", { userId, activityList });
    return activityList;
  },

  async fetchActivityListForTask({ commit }: any, taskId: TaskId) {
    const activityList = (
      await axios.get(`/api/activity?containerid=${taskId}&type=bytebase.task.`)
    ).data.data;
    commit("setActivityListForTask", { taskId, activityList });
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

  setActivityListForTask(
    state: ActivityState,
    {
      taskId,
      activityList,
    }: {
      taskId: UserId;
      activityList: Activity[];
    }
  ) {
    state.activityListByTask.set(taskId, activityList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
