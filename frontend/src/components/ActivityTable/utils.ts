import { head } from "lodash-es";
import { getIssueId } from "@/store/modules/v1/common";
import type {
  ActivityIssueCreatePayload,
  ActivityProjectRepositoryPushPayload,
  ActivityProjectDatabaseTransferPayload,
} from "@/types";
import { UNKNOWN_ID, EMPTY_ID } from "@/types";
import type { LogEntity } from "@/types/proto/v1/logging_service";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";
import type { Link } from "./types";

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
    case LogEntity_Action.ACTION_PIPELINE_TASK_PRIOR_BACKUP:
    case LogEntity_Action.ACTION_ISSUE_STATUS_UPDATE:
    case LogEntity_Action.ACTION_ISSUE_CREATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityIssueCreatePayload;

      // TODO:
      // activity.resource is now like "issues/1234"
      // We need to DML them to "projects/xxxxx/issues/1234"
      // to migrate to the new issue detail routes
      const issueId = getIssueId(activity.resource);
      if (issueId === UNKNOWN_ID || issueId === EMPTY_ID) {
        return undefined;
      }
      return {
        title: payload.issueName,
        path: `/issue/${issueId}`,
        external: false,
      };
    }
  }
  return undefined;
};
