import { head } from "lodash-es";
import { getIssueId } from "@/store/modules/v1/common";
import {
  ActivityIssueCreatePayload,
  ActivityProjectRepositoryPushPayload,
  ActivityProjectDatabaseTransferPayload,
} from "@/types";
import { LogEntity, LogEntity_Action } from "@/types/proto/v1/logging_service";
import { Link } from "./types";

export const getLinkFromActivity = (activity: LogEntity): Link | undefined => {
  switch (activity.action) {
    case LogEntity_Action.ACTION_PROJECT_REPOSITORY_PUSH: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityProjectRepositoryPushPayload;
      const commit =
        head(payload.pushEvent.commits) ?? payload.pushEvent.fileCommit;
      if (commit && commit.id && commit.url) {
        return {
          title: commit.id.substring(0, 7),
          path: commit.url,
          external: true,
        };
      }
      // Downgrade for legacy data.
      return undefined;
    }
    case LogEntity_Action.ACTION_PROJECT_DATABASE_TRANSFER: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityProjectDatabaseTransferPayload;
      return {
        title: payload.databaseName,
        path: `/db/${payload.databaseId}`,
        external: false,
      };
    }
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE:
    case LogEntity_Action.ACTION_PIPELINE_STAGE_STATUS_UPDATE:
    case LogEntity_Action.ACTION_PIPELINE_TASK_RUN_STATUS_UPDATE:
    case LogEntity_Action.ACTION_ISSUE_STATUS_UPDATE:
    case LogEntity_Action.ACTION_ISSUE_CREATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityIssueCreatePayload;
      return {
        title: payload.issueName,
        path: `/issue/${getIssueId(activity.resource)}`,
        external: false,
      };
    }
  }
  return undefined;
};
