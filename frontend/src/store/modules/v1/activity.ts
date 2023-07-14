import { defineStore } from "pinia";
import dayjs from "dayjs";
import { reactive } from "vue";
import utc from "dayjs/plugin/utc";

import { loggingServiceClient } from "@/grpcweb";
import {
  IdType,
  FindActivityMessage,
  Issue,
  UNKNOWN_ID,
  EMPTY_ID,
} from "@/types";
import { userNamePrefix, getLogId, logNamePrefix } from "./common";
import {
  LogEntity,
  LogEntity_Action,
  LogEntity_Level,
  logEntity_ActionToJSON,
  logEntity_LevelToJSON,
} from "@/types/proto/v1/logging_service";
import { isDatabaseRelatedIssueType } from "@/utils";
import { useIssueStore } from "../issue";
import { useCurrentUserV1 } from "../auth";

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
  const activityListByIssue = reactive(new Map<IdType, LogEntity[]>());

  const fetchActivityList = async (find: FindActivityMessage) => {
    const resp = await loggingServiceClient.listLogs({
      orderBy: find.order ? `create_time ${find.order}` : "",
      filter: buildFilter(find),
      pageSize: find.pageSize,
      pageToken: find.pageToken,
    });

    return resp;
  };

  const getActivityListByIssue = (issueId: IdType): LogEntity[] => {
    return activityListByIssue.get(issueId) || [];
  };

  const fetchActivityListForIssue = async (issue: Issue) => {
    const requests = [
      fetchActivityList({
        resource: `issues/${issue.id}`,
        order: "asc",
        pageSize: 1000,
      }).then((resp) => resp.logEntities),
    ];
    if (isDatabaseRelatedIssueType(issue.type) && issue.pipeline) {
      requests.push(
        fetchActivityList({
          resource: `pipelines/${issue.pipeline.id}`,
          order: "asc",
          pageSize: 1000,
        }).then((resp) => resp.logEntities)
      );
    } else {
      requests.push(Promise.resolve([]));
    }

    const [listForIssue, listForPipeline] = await Promise.all(requests);
    const mergedList = [...listForIssue, ...listForPipeline];
    mergedList.sort((a, b) => {
      if (a.createTime !== b.createTime) {
        return (a.createTime?.getTime() ?? 0) - (b.createTime?.getTime() ?? 0);
      }

      return getLogId(a.name) - getLogId(b.name);
    });

    activityListByIssue.set(issue.id, mergedList);
    return mergedList;
  };

  const fetchActivityListByIssueId = async (issueId: IdType) => {
    const issue = useIssueStore().getIssueById(issueId);
    if (issue.id === UNKNOWN_ID) {
      return;
    }
    return fetchActivityListForIssue(issue);
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

  return {
    fetchActivityList,
    fetchActivityListForIssue,
    fetchActivityListByIssueId,
    fetchActivityListForQueryHistory,
    fetchActivityByUID,
    getActivityListByIssue,
    getResourceId,
  };
});
