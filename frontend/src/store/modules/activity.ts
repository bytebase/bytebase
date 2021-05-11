import axios from "axios";
import {
  PrincipalId,
  IssueId,
  ActivityId,
  Activity,
  ActivityCreate,
  ActivityState,
  ResourceObject,
  ActivityPatch,
} from "../../types";

function convert(activity: ResourceObject, rootGetters: any): Activity {
  const creator = rootGetters["principal/principalById"](
    activity.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    activity.attributes.updaterId
  );
  return {
    ...(activity.attributes as Omit<Activity, "id" | "creator" | "updater">),
    id: activity.id,
    creator,
    updater,
  };
}

const state: () => ActivityState = () => ({
  activityListByUser: new Map(),
  activityListByIssue: new Map(),
});

const getters = {
  activityListByUser:
    (state: ActivityState) =>
    (userId: PrincipalId): Activity[] => {
      return state.activityListByUser.get(userId) || [];
    },
  activityListByIssue:
    (state: ActivityState) =>
    (issueId: IssueId): Activity[] => {
      return state.activityListByIssue.get(issueId) || [];
    },
};

const actions = {
  async fetchActivityListForUser(
    { commit, rootGetters }: any,
    userId: PrincipalId
  ) {
    const activityList = (await axios.get(`/api/activity`)).data.data.map(
      (activity: ResourceObject) => {
        return convert(activity, rootGetters);
      }
    );

    commit("setActivityListForUser", { userId, activityList });
    return activityList;
  },

  async fetchActivityListForIssue(
    { commit, rootGetters }: any,
    issueId: IssueId
  ) {
    const activityList = (
      await axios.get(`/api/activity?container=${issueId}`)
    ).data.data.map((activity: ResourceObject) => {
      return convert(activity, rootGetters);
    });

    commit("setActivityListForIssue", { issueId, activityList });
    return activityList;
  },

  async createActivity(
    { dispatch, rootGetters }: any,
    newActivity: ActivityCreate
  ) {
    const createdActivity: Activity = convert(
      (
        await axios.post(`/api/activity`, {
          data: {
            type: "ActivityCreate",
            attributes: newActivity,
          },
        })
      ).data.data,
      rootGetters
    );

    // There might exist other activities happened since the last fetch, so we do a full refetch.
    if (newActivity.actionType.startsWith("bb.issue.")) {
      dispatch("fetchActivityListForIssue", newActivity.containerId);
    }

    return createdActivity;
  },

  async updateComment(
    { dispatch, rootGetters }: any,
    {
      activityId,
      updatedComment,
      updaterId,
    }: {
      activityId: ActivityId;
      updatedComment: string;
      updaterId: PrincipalId;
    }
  ) {
    const activityPatch: ActivityPatch = {
      comment: updatedComment,
      updaterId,
    };
    const updatedActivity = convert(
      (
        await axios.patch(`/api/activity/${activityId}`, {
          data: {
            type: "activitypatch",
            attributes: activityPatch,
          },
        })
      ).data.data,
      rootGetters
    );

    dispatch("fetchActivityListForIssue", updatedActivity.containerId);

    return updatedActivity;
  },

  async deleteActivity({ dispatch }: any, activity: Activity) {
    await axios.delete(`/api/activity/${activity.id}`);

    if (activity.actionType.startsWith("bb.issue.")) {
      dispatch("fetchActivityListForIssue", activity.containerId);
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
      userId: PrincipalId;
      activityList: Activity[];
    }
  ) {
    state.activityListByUser.set(userId, activityList);
  },

  setActivityListForIssue(
    state: ActivityState,
    {
      issueId,
      activityList,
    }: {
      issueId: PrincipalId;
      activityList: Activity[];
    }
  ) {
    state.activityListByIssue.set(issueId, activityList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
