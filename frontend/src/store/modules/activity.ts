import axios from "axios";
import {
  UserId,
  TaskId,
  ActivityId,
  Activity,
  ActivityNew,
  ActivityPatch,
  ActivityState,
  ResourceObject,
} from "../../types";

function convert(activity: ResourceObject): Activity {
  return {
    id: activity.id,
    ...(activity.attributes as Omit<Activity, "id">),
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
  async fetchActivityListForUser({ commit }: any, userId: UserId) {
    const activityList = (await axios.get(`/api/activity`)).data.data.map(
      (activity: ResourceObject) => {
        return convert(activity);
      }
    );

    commit("setActivityListForUser", { userId, activityList });
    return activityList;
  },

  async fetchActivityListForTask({ commit }: any, taskId: TaskId) {
    const activityList = (
      await axios.get(`/api/activity?containerid=${taskId}&type=bytebase.task.`)
    ).data.data.map((activity: ResourceObject) => {
      return convert(activity);
    });

    commit("setActivityListForTask", { taskId, activityList });
    return activityList;
  },

  async createActivity({ dispatch }: any, newActivity: ActivityNew) {
    const createdActivity: Activity = convert(
      (
        await axios.post(`/api/activity`, {
          data: {
            type: "activity",
            attributes: newActivity,
          },
        })
      ).data.data
    );

    // There might exist other activities happened since the last fetch, so we do a full refetch.
    if (newActivity.actionType.startsWith("bytebase.task.")) {
      dispatch("fetchActivityListForTask", newActivity.containerId);
    }

    return createdActivity;
  },

  async updateComment(
    { dispatch }: any,
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
      ).data.data
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
