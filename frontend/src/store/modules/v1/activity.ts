import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { loggingServiceClient } from "@/grpcweb";
import {
  IdType,
  FindActivityMessage,
  UNKNOWN_ID,
  EMPTY_ID,
  ComposedIssue,
} from "@/types";
import { ExportFormat } from "@/types/proto/v1/common";
import {
  LogEntity,
  LogEntity_Action,
  LogEntity_Level,
  logEntity_ActionToJSON,
  logEntity_LevelToJSON,
} from "@/types/proto/v1/logging_service";
import { isDatabaseRelatedIssue, extractRolloutUID } from "@/utils";
import { useCurrentUserV1 } from "../auth";
import { userNamePrefix, getLogId, logNamePrefix } from "./common";
import { experimentalFetchIssueByUID } from "./experimental-issue";

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
    const resp = await loggingServiceClient.listLogs({
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
    if (isDatabaseRelatedIssue(issue) && issue.rollout) {
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

  const fetchActivityListByIssueUID = async (uid: string) => {
    const issue = await experimentalFetchIssueByUID(uid);
    return fetchActivityListForIssueV1(issue);
  };

  const fetchActivityListForQueryHistory = async ({
    limit,
    order,
  }: {
    limit: number;
    order: "asc" | "desc";
  }) => {
    const currentUserV1 = useCurrentUserV1();

    return fetchActivityList({
      action: [LogEntity_Action.ACTION_DATABASE_SQL_EDITOR_QUERY],
      creatorEmail: currentUserV1.value.email,
      order,
      pageSize: limit,
      level: [LogEntity_Level.LEVEL_INFO, LogEntity_Level.LEVEL_WARNING],
    }).then((resp) => resp.logEntities);
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
    fetchActivityListByIssueUID,
    fetchActivityListForQueryHistory,
    fetchActivityByUID,
    getActivityListByIssueV1,
    getResourceId,
    exportData,
  };
});
