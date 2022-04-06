import axios from "axios";
import {
  Activity,
  ActivityCreate,
  ActivityId,
  ActivityPatch,
  ActivityState,
  IssueId,
  PrincipalId,
  ProjectId,
  ResourceObject,
} from "../../types";
import { useAuthStore, getPrincipalFromIncludedList } from "../pinia";

function convert(
  activity: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Activity {
  const payload = activity.attributes.payload
    ? JSON.parse((activity.attributes.payload as string) || "{}")
    : {};
  return {
    ...(activity.attributes as Omit<Activity, "id" | "creator" | "updater">),
    creator: getPrincipalFromIncludedList(
      activity.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      activity.relationships!.updater.data,
      includedList
    ),
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
    const data = (await axios.get(`/api/activity`)).data;
    const activityList = data.data.map((activity: ResourceObject) => {
      return convert(activity, data.included, rootGetters);
    });

    commit("setActivityListForUser", { userId, activityList });
    return activityList;
  },

  async fetchActivityListForIssue(
    { commit, rootGetters }: any,
    issueId: IssueId
  ) {
    const data = (await axios.get(`/api/activity?container=${issueId}`)).data;
    const activityList = data.data.map((activity: ResourceObject) => {
      return convert(activity, data.included, rootGetters);
    });

    commit("setActivityListForIssue", { issueId, activityList });
    return activityList;
  },

  // We do not store the returned list because the caller will specify different limits
  async fetchActivityListForProject(
    { rootGetters }: any,
    {
      projectId,
      limit,
    }: {
      projectId: ProjectId;
      limit?: number;
    }
  ) {
    const queryList = [`container=${projectId}`];
    if (limit) {
      queryList.push(`limit=${limit}`);
    }
    const data = (await axios.get(`/api/activity?${queryList.join("&")}`)).data;
    const activityList = data.data.map((activity: ResourceObject) => {
      return convert(activity, data.included, rootGetters);
    });

    return activityList;
  },

  async fetchActivityListForQueryHistory(
    { rootGetters }: any,
    {
      limit,
    }: {
      limit: number;
    }
  ) {
    const { currentUser } = useAuthStore();
    const queryList = [
      "type=bb.sql-editor.query",
      `user=${currentUser.id}`,
      `limit=${limit}`,
      // only fetch the successful query history
      `level=INFO`,
    ];
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
      dispatch("fetchActivityListForIssue", newActivity.containerId);
    }

    return createdActivity;
  },

  async updateComment(
    { dispatch, rootGetters }: any,
    {
      activityId,
      updatedComment,
    }: {
      activityId: ActivityId;
      updatedComment: string;
    }
  ) {
    const activityPatch: ActivityPatch = {
      comment: updatedComment,
    };
    const data = (
      await axios.patch(`/api/activity/${activityId}`, {
        data: {
          type: "activityPatch",
          attributes: activityPatch,
        },
      })
    ).data;
    const updatedActivity = convert(data.data, data.included, rootGetters);

    dispatch("fetchActivityListForIssue", updatedActivity.containerId);

    return updatedActivity;
  },

  async deleteActivity({ dispatch }: any, activity: Activity) {
    await axios.delete(`/api/activity/${activity.id}`);

    if (activity.type.startsWith("bb.issue.")) {
      dispatch("fetchActivityListForIssue", activity.containerId);
    }
  },

  async deleteActivityById(_: any, id: number) {
    await axios.delete(`/api/activity/${id}`);
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
      issueId: IssueId;
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
