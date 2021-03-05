import axios from "axios";
import {
  UserId,
  TaskId,
  ActivityId,
  Activity,
  ActivityNew,
  ActivityState,
  ResourceObject,
} from "../../types";

function convert(activity: ResourceObject, rootGetters: any): Activity {
  const creator = rootGetters["principal/principalById"](
    activity.attributes.creatorId
  );
  return {
    id: activity.id,
    creator,
    ...(activity.attributes as Omit<Activity, "id" | "creator">),
  };
}

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
  async fetchActivityListForUser({ commit, rootGetters }: any, userId: UserId) {
    const activityList = (await axios.get(`/api/activity`)).data.data.map(
      (activity: ResourceObject) => {
        return convert(activity, rootGetters);
      }
    );

    commit("setActivityListForUser", { userId, activityList });
    return activityList;
  },

  async fetchActivityListForTask({ commit, rootGetters }: any, taskId: TaskId) {
    const activityList = (
      await axios.get(`/api/activity?containerid=${taskId}&type=bytebase.task.`)
    ).data.data.map((activity: ResourceObject) => {
      return convert(activity, rootGetters);
    });

    commit("setActivityListForTask", { taskId, activityList });
    return activityList;
  },

  async createActivity(
    { dispatch, rootGetters }: any,
    newActivity: ActivityNew
  ) {
    const createdActivity: Activity = convert(
      (
        await axios.post(`/api/activity`, {
          data: {
            type: "activity",
            attributes: newActivity,
          },
        })
      ).data.data,
      rootGetters
    );

    // There might exist other activities happened since the last fetch, so we do a full refetch.
    if (newActivity.actionType.startsWith("bytebase.task.")) {
      dispatch("fetchActivityListForTask", newActivity.containerId);
    }

    return createdActivity;
  },

  async updateComment(
    { dispatch, rootGetters }: any,
    {
      activityId,
      updatedComment,
    }: { activityId: ActivityId; updatedComment: string }
  ) {
    const updatedActivity = convert(
      (
        await axios.patch(`/api/activity/${activityId}`, {
          data: {
            type: "activitypatch",
            attributes: {
              payload: {
                comment: updatedComment,
              },
            },
          },
        })
      ).data.data,
      rootGetters
    );

    dispatch("fetchActivityListForTask", updatedActivity.containerId);

    return updatedActivity;
  },

  async deleteActivity({ dispatch }: any, activity: Activity) {
    await axios.delete(`/api/activity/${activity.id}`);

    if (activity.actionType.startsWith("bytebase.task.")) {
      dispatch("fetchActivityListForTask", activity.containerId);
    }
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
