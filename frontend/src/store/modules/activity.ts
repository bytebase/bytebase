import { defineStore } from "pinia";
import axios from "axios";
import { stringify } from "qs";
import {
  Activity,
  ActivityCreate,
  ActivityId,
  ActivityPatch,
  ActivityState,
  Issue,
  IssueId,
  PrincipalId,
  ProjectId,
  ResourceObject,
  UNKNOWN_ID,
} from "@/types";
import { useAuthStore } from "./auth";
import { getPrincipalFromIncludedList } from "./principal";
import { useIssueStore } from "./issue";

function convert(
  activity: ResourceObject,
  includedList: ResourceObject[]
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

export const useActivityStore = defineStore("activity", {
  state: (): ActivityState => ({
    activityListByUser: new Map(),
    activityListByIssue: new Map(),
  }),
  actions: {
    convert(
      activity: ResourceObject,
      includedList: ResourceObject[]
    ): Activity {
      return convert(activity, includedList || []);
    },
    getActivityListByUser(userId: PrincipalId): Activity[] {
      return this.activityListByUser.get(userId) || [];
    },
    getActivityListByIssue(issueId: IssueId): Activity[] {
      return this.activityListByIssue.get(issueId) || [];
    },
    setActivityListForUser({
      userId,
      activityList,
    }: {
      userId: PrincipalId;
      activityList: Activity[];
    }) {
      this.activityListByUser.set(userId, activityList);
    },
    setActivityListForIssue({
      issueId,
      activityList,
    }: {
      issueId: IssueId;
      activityList: Activity[];
    }) {
      this.activityListByIssue.set(issueId, activityList);
    },
    async fetchActivityListForUser(userId: PrincipalId) {
      const data = (await axios.get(`/api/activity?order=DESC`)).data;
      const activityList: Activity[] = data.data.map(
        (activity: ResourceObject) => {
          return convert(activity, data.included);
        }
      );

      this.setActivityListForUser({ userId, activityList });
      return activityList;
    },
    async fetchActivityList(params: {
      typePrefix: string;
      container: number | string;
      order: "ASC" | "DESC";
      limit?: number;
    }) {
      const url = `/api/activity?${stringify(params)}`;
      const response = (await axios.get(url)).data;
      const activityList: Activity[] = response.data.map(
        (activity: ResourceObject) => {
          return convert(activity, response.included);
        }
      );
      return activityList;
    },
    async fetchActivityListForIssue(issue: Issue) {
      const requestListForIssue = this.fetchActivityList({
        typePrefix: "bb.issue.",
        container: issue.id,
        order: "ASC",
      });
      const requestListForPipeline = this.fetchActivityList({
        typePrefix: "bb.pipeline.",
        container: issue.pipeline.id,
        order: "ASC",
      });
      const [listForIssue, listForPipeline] = await Promise.all([
        requestListForIssue,
        requestListForPipeline,
      ]);

      const mergedList = [...listForIssue, ...listForPipeline];
      mergedList.sort((a, b) => {
        if (a.createdTs !== b.createdTs) {
          return a.createdTs - b.createdTs;
        }

        return a.id - b.id;
      });

      this.setActivityListForIssue({
        issueId: issue.id,
        activityList: mergedList,
      });
      return mergedList;
    },
    async fetchActivityListByIssueId(issueId: IssueId) {
      const issue = useIssueStore().getIssueById(issueId);
      if (issue.id === UNKNOWN_ID) {
        return;
      }
      this.fetchActivityListForIssue(issue);
    },
    // We do not store the returned list because the caller will specify different limits.
    async fetchActivityListForProject({
      projectId,
      limit,
    }: {
      projectId: ProjectId;
      limit?: number;
    }) {
      const queryList = [
        "typePrefix=bb.project.",
        `container=${projectId}`,
        `order=DESC`,
      ];
      if (limit) {
        queryList.push(`limit=${limit}`);
      }
      const data = (await axios.get(`/api/activity?${queryList.join("&")}`))
        .data;
      const activityList: Activity[] = data.data.map(
        (activity: ResourceObject) => {
          return convert(activity, data.included);
        }
      );

      return activityList;
    },
    async fetchActivityListForQueryHistory({ limit }: { limit: number }) {
      const { currentUser } = useAuthStore();
      const fetchQueryList = async (level: string) => {
        const queryList = [
          "typePrefix=bb.sql-editor.query",
          `user=${currentUser.id}`,
          `order=DESC`,
          `limit=${limit}`,
          // only fetch the successful query history
          `level=${level}`,
        ];
        const data = (await axios.get(`/api/activity?${queryList.join("&")}`))
          .data;
        const activityList: Activity[] = data.data.map(
          (activity: ResourceObject) => {
            return convert(activity, data.included);
          }
        );
        return activityList;
      };
      const [successful, withWarning] = await Promise.all([
        fetchQueryList("INFO"),
        fetchQueryList("WARN"),
      ]);
      const mixedList = [...successful, ...withWarning];

      // ORDER BY `id` DESC
      mixedList.sort((a, b) => b.id - a.id);

      // return the first `limit` rows
      return mixedList.slice(0, limit);
    },
    async fetchActivityListForDatabaseByProjectId({
      projectId,
      limit,
    }: {
      projectId: ProjectId;
      limit?: number;
    }) {
      const queryList = [
        "typePrefix=bb.database.",
        `container=${projectId}`,
        `order=DESC`,
      ];
      if (limit) {
        queryList.push(`limit=${limit}`);
      }
      const data = (await axios.get(`/api/activity?${queryList.join("&")}`))
        .data;
      const activityList: Activity[] = data.data.map(
        (activity: ResourceObject) => {
          return convert(activity, data.included);
        }
      );

      return activityList;
    },
    async createActivity(newActivity: ActivityCreate) {
      const data = (
        await axios.post(`/api/activity`, {
          data: {
            type: "activityCreate",
            attributes: newActivity,
          },
        })
      ).data;
      const createdActivity = convert(data.data, data.included);

      // There might exist other activities happened since the last fetch, so we do a full refetch.
      if (newActivity.type.startsWith("bb.issue.")) {
        this.fetchActivityListByIssueId(newActivity.containerId);
      }

      return createdActivity;
    },
    async updateComment({
      activityId,
      updatedComment,
    }: {
      activityId: ActivityId;
      updatedComment: string;
    }) {
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
      const updatedActivity = convert(data.data, data.included);

      this.fetchActivityListByIssueId(updatedActivity.containerId);

      return updatedActivity;
    },
    async deleteActivity(activity: Activity) {
      await axios.delete(`/api/activity/${activity.id}`);

      if (activity.type.startsWith("bb.issue.")) {
        this.fetchActivityListByIssueId(activity.containerId);
      }
    },
    async deleteActivityById(id: number) {
      await axios.delete(`/api/activity/${id}`);
    },
  },
});
