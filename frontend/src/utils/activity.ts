import { Store } from "vuex";
import { IssueBuiltinFieldId } from "../plugins";
import {
  Activity,
  ActivityIssueFieldUpdatePayload,
  ActivityIssueStatusUpdatePayload,
} from "../types";

let store: Store<any>;

export function registerStoreWithActivityUtil(theStore: Store<any>) {
  store = theStore;
}

export function issueActivityActionSentence(
  activity: Activity
): [string, Record<string, any>] {
  switch (activity.type) {
    case "bb.issue.create":
      return ["activity.sentence.created-issue", {}];
    case "bb.issue.comment.create":
      return ["activity.sentence.commented", {}];
    case "bb.issue.field.update": {
      const update = activity.payload as ActivityIssueFieldUpdatePayload;

      let name = "Unknown Field";
      let oldValue = undefined;
      let newValue = undefined;

      switch (update.fieldId) {
        case IssueBuiltinFieldId.ASSIGNEE: {
          if (update.oldValue && update.newValue) {
            const oldName = store.getters["principal/principalById"](
              update.oldValue
            ).name;

            const newName = store.getters["principal/principalById"](
              update.newValue
            ).name;
            return [
              "activity.sentence.reassigned-issue",
              {
                oldName,
                newName,
              },
            ];
          } else if (!update.oldValue && update.newValue) {
            const newName = store.getters["principal/principalById"](
              update.newValue
            ).name;
            return [
              "activity.sentence.assigned-issue",
              {
                newName,
              },
            ];
          } else if (update.oldValue && !update.newValue) {
            const oldName = store.getters["principal/principalById"](
              update.oldValue
            ).name;
            return [
              "activity.sentence.unassigned-issue",
              {
                oldName,
              },
            ];
          } else {
            return ["activity.sentence.invalid-assignee-update", {}];
          }
        }
        // We don't display subscriber change for now
        case IssueBuiltinFieldId.SUBSCRIBER_LIST:
          break;
        case IssueBuiltinFieldId.DESCRIPTION:
          // Description could be very long, so we don't display it.
          return ["activity.sentence.changed-description", {}];
        case IssueBuiltinFieldId.NAME:
        case IssueBuiltinFieldId.PROJECT:
        case IssueBuiltinFieldId.SQL:
        case IssueBuiltinFieldId.ROLLBACK_SQL: {
          if (update.fieldId == IssueBuiltinFieldId.NAME) {
            name = "name";
          } else if (update.fieldId == IssueBuiltinFieldId.SQL) {
            name = "SQL";
          } else if (update.fieldId == IssueBuiltinFieldId.ROLLBACK_SQL) {
            name = "Rollback SQL";
          }

          oldValue = update.oldValue;
          newValue = update.newValue;
          if (oldValue && newValue) {
            return [
              "activity.sentence.changed-from-to",
              {
                name,
                oldValue,
                newValue,
              },
            ];
          } else if (oldValue) {
            return [
              "activity.sentence.unset-from",
              {
                name,
                oldValue,
              },
            ];
          } else if (newValue) {
            return [
              "activity.sentence.set-to",
              {
                name,
                newValue,
              },
            ];
          } else {
            return [
              "activity.sentence.changed-update",
              {
                name,
              },
            ];
          }
        }
      }

      return ["activity.sentence.updated", {}];
    }
    case "bb.issue.status.update": {
      const update = activity.payload as ActivityIssueStatusUpdatePayload;
      switch (update.newStatus) {
        case "OPEN":
          return ["activity.sentence.reopened-issue", {}];
        case "DONE":
          return ["activity.sentence.resolved-issue", {}];
        case "CANCELED":
          return ["activity.sentence.canceled-issue", {}];
      }
    }
  }
  return ["activity.sentence.empty", {}];
}
