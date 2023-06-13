import { defineStore } from "pinia";
import axios from "axios";
import { cloneDeep } from "lodash-es";
import { ActivityCreate, ActivityId, ActivityPatch, IssueId } from "@/types";
import { useActivityV1Store } from "./v1";

export const useActivityLegacyStore = defineStore("activity", {
  actions: {
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
      await axios.post(`/api/activity`, postData);
      // There might exist other activities happened since the last fetch, so we do a full refetch.
      if (newActivity.type.startsWith("bb.issue.")) {
        useActivityV1Store().fetchActivityListByIssueId(
          newActivity.containerId
        );
      }
    },
    async updateComment({
      activityId,
      issueId,
      updatedComment,
    }: {
      activityId: ActivityId;
      issueId: IssueId;
      updatedComment: string;
    }) {
      const activityPatch: ActivityPatch = {
        comment: updatedComment,
      };
      await axios.patch(`/api/activity/${activityId}`, {
        data: {
          type: "activityPatch",
          attributes: activityPatch,
        },
      });
      useActivityV1Store().fetchActivityListByIssueId(issueId);
    },
  },
});
