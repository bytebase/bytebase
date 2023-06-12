import { defineStore } from "pinia";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";

import { loggingServiceClient } from "@/grpcweb";
import { FindActivityMessage } from "@/types";
import { userNamePrefix } from "./common";
import {
  logEntity_ActionToJSON,
  logEntity_LevelToJSON,
} from "@/types/proto/v1/logging_service";

dayjs.extend(utc);

export const useActivityV1Store = defineStore("activity_v1", () => {
  const fetchActivityList = async (find: FindActivityMessage) => {
    const resp = await loggingServiceClient.listLogs({
      orderBy: find.order ? `create_time ${find.order}` : "",
      filter: buildFilter(find),
      pageSize: find.pageSize,
      pageToken: find.pageToken,
    });

    return resp;
  };

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
        `level = "${find.level
          .map((l) => logEntity_LevelToJSON(l))
          .join(" | ")}"`
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

  return {
    fetchActivityList,
  };
});
