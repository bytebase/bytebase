import { useUserStore } from "@/store";
import { IssueBuiltinFieldId } from "../plugins";
import { t } from "@/plugins/i18n";
import {
  Activity,
  ActivityIssueFieldUpdatePayload,
  ActivityIssueStatusUpdatePayload,
  unknownUser,
} from "../types";

export function issueActivityActionSentence(
  activity: Activity
): [string, Record<string, any>] {
  switch (activity.type) {
    case "bb.issue.create":
      return ["activity.sentence.created-issue", {}];
    case "bb.issue.comment.create":
      return ["activity.sentence.commented", {}];
    case "bb.issue.field.update": {
      const userStore = useUserStore();
      const update = activity.payload as ActivityIssueFieldUpdatePayload;

      switch (update.fieldId) {
        case IssueBuiltinFieldId.NAME: {
          const oldName = update.oldValue ?? "";
          const newName = update.newValue ?? "";
          return [
            "activity.sentence.changed-from-to",
            {
              name: t("issue.issue-name").toLowerCase(),
              oldValue: oldName,
              newValue: newName,
            },
          ];
        }
        case IssueBuiltinFieldId.ASSIGNEE: {
          if (update.oldValue && update.newValue) {
            const oldName = (
              userStore.getUserById(String(update.oldValue)) ?? unknownUser()
            ).name;
            const newName = (
              userStore.getUserById(String(update.newValue)) ?? unknownUser()
            ).name;
            return [
              "activity.sentence.reassigned-issue",
              {
                oldName,
                newName,
              },
            ];
          } else if (!update.oldValue && update.newValue) {
            const newName = (
              userStore.getUserById(String(update.newValue)) ?? unknownUser()
            ).name;
            return [
              "activity.sentence.assigned-issue",
              {
                newName,
              },
            ];
          } else if (update.oldValue && !update.newValue) {
            const oldName = (
              userStore.getUserById(String(update.oldValue)) ?? unknownUser()
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
        case IssueBuiltinFieldId.PROJECT:
        case IssueBuiltinFieldId.SQL:
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
