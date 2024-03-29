import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { loggingServiceClient } from "@/grpcweb";
import type { IdType, FindActivityMessage, ComposedIssue } from "@/types";
import { UNKNOWN_ID, EMPTY_ID } from "@/types";
import type { ExportFormat } from "@/types/proto/v1/common";
import type { LogEntity } from "@/types/proto/v1/logging_service";
import {
  logEntity_ActionToJSON,
  logEntity_LevelToJSON,
} from "@/types/proto/v1/logging_service";
import { isDatabaseChangeRelatedIssue, extractRolloutUID } from "@/utils";
import { userNamePrefix, getLogId, logNamePrefix } from "./common";

dayjs.extend(utc);

const buildFilter = (find: FindActivityMessage): string => {
  const filter: string[] = [];
  if (find.resource) {
    filter.push(`resource = "${find.resource}"`);
  }
  if (find.creatorEmail) {
    filter.push(`creator = "${userNamePrefix}${find.creatorEmail}"`);
  }
  if (find.level) {
    filter.push(
      `level = "${find.level.map((l) => logEntity_LevelToJSON(l)).join(" | ")}"`
    );
  }
  if (find.action) {
    filter.push(
      `action = "${find.action
        .map((a) => logEntity_ActionToJSON(a))
        .join(" | ")}"`
    );
  }
  if (find.createdTsAfter) {
    filter.push(
      `create_time >= "${dayjs(find.createdTsAfter).utc().format()}"`
    );
  }
  if (find.createdTsBefore) {
    filter.push(
      `create_time <= "${dayjs(find.createdTsBefore).utc().format()}"`
    );
  }
  return filter.join(" && ");
};

export const useActivityV1Store = defineStore("activity_v1", () => {
  const activityListByIssueV1 = reactive(new Map<string, LogEntity[]>());

  const fetchActivityList = async (find: FindActivityMessage) => {
    const resp = await loggingServiceClient.searchLogs({
      orderBy: find.order ? `create_time ${find.order}` : "",
      filter: buildFilter(find),
      pageSize: find.pageSize,
      pageToken: find.pageToken,
    });

    return resp;
  };

  const getActivityListByIssueV1 = (uid: string): LogEntity[] => {
    return activityListByIssueV1.get(uid) || [];
  };
  const fetchActivityListForIssueV1 = async (issue: ComposedIssue) => {
    const requests = [
      fetchActivityList({
        resource: `issues/${issue.uid}`,
        order: "asc",
        pageSize: 1000, // Pagination is complex, and not high priority
      }).then((resp) => resp.logEntities),
    ];
    if (isDatabaseChangeRelatedIssue(issue) && issue.rollout) {
      const pipelineUID = extractRolloutUID(issue.rollout);
      requests.push(
        fetchActivityList({
          resource: `pipelines/${pipelineUID}`,
          order: "asc",
          pageSize: 1000, // Pagination is complex, and not high priority
        }).then((resp) => resp.logEntities)
      );
    } else {
      requests.push(Promise.resolve([]));
    }

    const [listForIssue, listForPipeline] = await Promise.all(requests);
    const mergedList = [...listForIssue, ...listForPipeline];
    mergedList.sort((a, b) => {
      return getLogId(a.name) - getLogId(b.name);
    });

    activityListByIssueV1.set(issue.uid, mergedList);
    return mergedList;
  };

  const fetchActivityByUID = async (uid: IdType) => {
    if (uid == EMPTY_ID || uid == UNKNOWN_ID) {
      return;
    }
    const entity = await loggingServiceClient.getLog({
      name: `${logNamePrefix}${uid}`,
    });
    return entity;
  };

  const getResourceId = (activity: LogEntity): IdType => {
    return activity.resource.split("/").slice(-1)[0];
  };

  const exportData = async ({
    find,
    format,
  }: {
    find: FindActivityMessage;
    format: ExportFormat;
  }) => {
    const resp = await loggingServiceClient.exportLogs({
      orderBy: find.order ? `create_time ${find.order}` : "",
      filter: buildFilter(find),
      format,
    });
    return resp.content;
  };

  return {
    fetchActivityList,
    fetchActivityListForIssueV1,
    fetchActivityByUID,
    getActivityListByIssueV1,
    getResourceId,
    exportData,
  };
});
