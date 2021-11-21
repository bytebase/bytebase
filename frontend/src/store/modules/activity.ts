import axios from "axios";
import {
  Activity,
  ActivityCreate,
  ActivityID,
  ActivityPatch,
  ActivityState,
  IssueID,
  PrincipalID,
  ProjectID,
  ResourceObject,
} from "../../types";

function convert(
  activity: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Activity {
  const payload = activity.attributes.payload
    ? JSON.parse((activity.attributes.payload as string) || "{}")
    : {};

  return {
    ...(activity.attributes as Omit<Activity, "id">),
    id: parseInt(activity.id),
    payload,
  };
}

const state: () => ActivityState = () => ({
  activityListByUser: new Map(),
  activityListByIssue: new Map(),
});

const getters = {
  convert:
    (state: ActivityState, getters: any, rootState: any, rootGetters: any) =>
    (activity: ResourceObject, includedList: ResourceObject[]): Activity => {
      return convert(activity, includedList || [], rootGetters);
    },

  activityListByUser:
    (state: ActivityState) =>
    (userID: PrincipalID): Activity[] => {
      return state.activityListByUser.get(userID) || [];
    },
  activityListByIssue:
    (state: ActivityState) =>
    (issueID: IssueID): Activity[] => {
      return state.activityListByIssue.get(issueID) || [];
    },
};

const actions = {
  async fetchActivityListForUser(
    { commit, rootGetters }: any,
    userID: PrincipalID
  ) {
    const data = (await axios.get(`/api/activity`)).data;
    const activityList = data.data.map((activity: ResourceObject) => {
      return convert(activity, data.included, rootGetters);
    });

    commit("setActivityListForUser", { userID, activityList });
    return activityList;
  },

  async fetchActivityListForIssue(
    { commit, rootGetters }: any,
    issueID: IssueID
  ) {
    const data = (await axios.get(`/api/activity?container=${issueID}`)).data;
    const activityList = data.data.map((activity: ResourceObject) => {
      return convert(activity, data.included, rootGetters);
    });

    commit("setActivityListForIssue", { issueID, activityList });
    return activityList;
  },

  // We do not store the returned list because the caller will specify different limits
  async fetchActivityListForProject(
    { rootGetters }: any,
    {
      projectID,
      limit,
    }: {
      projectID: ProjectID;
      limit?: number;
    }
  ) {
    var queryList = [`container=${projectID}`];
    if (limit) {
      queryList.push(`limit=${limit}`);
    }
    const data = (await axios.get(`/api/activity?${queryList.join("&")}`)).data;
    const activityList = data.data.map((activity: ResourceObject) => {
      return convert(activity, data.included, rootGetters);
    });

    return activityList;
  },

  async createActivity(
    { dispatch, rootGetters }: any,
    newActivity: ActivityCreate
  ) {
    const data = (
      await axios.post(`/api/activity`, {
        data: {
          type: "activityCreate",
          attributes: newActivity,
        },
      })
    ).data;
    const createdActivity: Activity = convert(
      data.data,
      data.included,
      rootGetters
    );

    // There might exist other activities happened since the last fetch, so we do a full refetch.
    if (newActivity.type.startsWith("bb.issue.")) {
      dispatch("fetchActivityListForIssue", newActivity.containerID);
    }

    return createdActivity;
  },

  async updateComment(
    { dispatch, rootGetters }: any,
    {
      activityID,
      updatedComment,
    }: {
      activityID: ActivityID;
      updatedComment: string;
    }
  ) {
    const activityPatch: ActivityPatch = {
      comment: updatedComment,
    };
    const data = (
      await axios.patch(`/api/activity/${activityID}`, {
        data: {
          type: "activityPatch",
          attributes: activityPatch,
        },
      })
    ).data;
    const updatedActivity = convert(data.data, data.included, rootGetters);

    dispatch("fetchActivityListForIssue", updatedActivity.containerID);

    return updatedActivity;
  },

  async deleteActivity({ dispatch }: any, activity: Activity) {
    await axios.delete(`/api/activity/${activity.id}`);

    if (activity.type.startsWith("bb.issue.")) {
      dispatch("fetchActivityListForIssue", activity.containerID);
    }
  },
};

const mutations = {
  setActivityListForUser(
    state: ActivityState,
    {
      userID,
      activityList,
    }: {
      userID: PrincipalID;
      activityList: Activity[];
    }
  ) {
    state.activityListByUser.set(userID, activityList);
  },

  setActivityListForIssue(
    state: ActivityState,
    {
      issueID,
      activityList,
    }: {
      issueID: IssueID;
      activityList: Activity[];
    }
  ) {
    state.activityListByIssue.set(issueID, activityList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
