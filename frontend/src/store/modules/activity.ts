import { defineStore } from "pinia";
import axios from "axios";
import { stringify } from "qs";
import { cloneDeep } from "lodash-es";
import {
  Activity,
  ActivityCreate,
  ActivityId,
  ActivityPatch,
  ActivityState,
  Issue,
  IssueId,
  PrincipalId,
  ResourceObject,
  UNKNOWN_ID,
  isPagedResponse,
  ResourceIdentifier,
  empty,
  ActivityFind,
} from "@/types";
import { convertEntityList } from "./utils";
import { useCurrentUser } from "./auth";
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

function getActivityFromIncludedList(
  data:
    | ResourceIdentifier<ResourceObject>
    | ResourceIdentifier<ResourceObject>[]
    | undefined,
  includedList: ResourceObject[]
): Activity {
  if (data == null) {
    return empty("ACTIVITY");
  }
  for (const item of includedList || []) {
    if (item.type !== "activity") {
      continue;
    }
    if (item.id == (data as ResourceIdentifier).id) {
      return convert(item, includedList);
    }
  }
  return empty("ACTIVITY");
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
    async fetchPagedActivityList(params: ActivityFind) {
      const url = `/api/activity?${stringify(params, {
        arrayFormat: "repeat",
      })}`;
      const responseData = (await axios.get(url)).data;
      const activityList = convertEntityList(
        responseData,
        "activityList",
        convert,
        getActivityFromIncludedList
      );
      const nextToken = isPagedResponse(responseData, "activityList")
        ? responseData.data.attributes.nextToken
        : "";
      return {
        nextToken,
        activityList,
      };
    },
    async fetchActivityList(params: ActivityFind) {
      const result = await this.fetchPagedActivityList(params);
      return result.activityList;
    },
    async fetchActivityListForIssue(issue: Issue) {
      // We should use two separate requests here because we are using different container ids.
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
    async fetchActivityListForQueryHistory({ limit }: { limit: number }) {
      const currentUser = useCurrentUser();
      const activityList = await this.fetchActivityList({
        typePrefix: "bb.sql-editor.query",
        user: currentUser.value.id as number,
        order: "DESC",
        limit,
        level: ["INFO", "WARN"],
      });

      // return the first `limit` rows
      return activityList.slice(0, limit);
    },
    async createActivity(newActivity: ActivityCreate) {
      const postData = {
        data: {
          type: "activityCreate",
          attributes: cloneDeep(newActivity) as any,
        },
      };
      if (postData.data.attributes.payload) {
        postData.data.attributes.payload = JSON.stringify(
          postData.data.attributes.payload
        );
      }
      const data = (await axios.post(`/api/activity`, postData)).data;
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
  },
});
