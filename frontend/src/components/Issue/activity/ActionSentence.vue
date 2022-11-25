<template>
  <renderer />
</template>

<script lang="ts" setup>
import { h, PropType } from "vue";
import {
  Activity,
  ActivityTaskEarliestAllowedTimeUpdatePayload,
  ActivityTaskFileCommitPayload,
  ActivityTaskStatementUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  ActivityIssueExternalApprovalRejectPayload,
  Issue,
  SYSTEM_BOT_ID,
} from "@/types";
import { findTaskById, issueActivityActionSentence } from "@/utils";
import { Translation, useI18n } from "vue-i18n";
import dayjs from "dayjs";
import SQLPreviewPopover from "@/components/misc/SQLPreviewPopover.vue";

const props = defineProps({
  activity: {
    type: Object as PropType<Activity>,
    required: true,
  },
  issue: {
    type: Object as PropType<Issue>,
    required: true,
  },
});

const { t } = useI18n();

const renderActionSentence = () => {
  const { activity, issue } = props;
  if (activity.type.startsWith("bb.issue.")) {
    if (activity.type === "bb.issue.external-approval.reject") {
      const payload =
        activity.payload as ActivityIssueExternalApprovalRejectPayload;

      switch (payload.externalApprovalType) {
        case "bb.plugin.im.feishu":
          return t("activity.sentence.external-approval-rejected", {
            stageName: payload.stageName,
            imName: t("common.feishu"),
          });
        default:
          return t("activity.sentence.external-approval-rejected", {
            stageName: payload.stageName,
            imName: "",
          });
      }
    }
    const [tid, params] = issueActivityActionSentence(activity);
    return t(tid, params);
  }
  switch (activity.type) {
    case "bb.pipeline.task.status.update": {
      const payload = activity.payload as ActivityTaskStatusUpdatePayload;
      if (payload.newStatus === "PENDING_APPROVAL") {
        // stale approval dismissed.

        const task = findTaskById(issue.pipeline, payload.taskId);
        const taskName = t("activity.sentence.task-name", {
          name: task.name,
        });
        return t("activity.sentence.dismissed-stale-approval", {
          task: taskName,
        });
      }

      let str = t("activity.sentence.changed");
      switch (payload.newStatus) {
        case "PENDING": {
          if (payload.oldStatus == "RUNNING") {
            str = t("activity.sentence.canceled");
          } else if (payload.oldStatus == "PENDING_APPROVAL") {
            str = t("activity.sentence.approved");
          }
          break;
        }
        case "RUNNING": {
          str = t("activity.sentence.started");
          break;
        }
        case "DONE": {
          str = t("activity.sentence.completed");
          break;
        }
        case "FAILED": {
          str = t("activity.sentence.failed");
          break;
        }
        case "CANCELED": {
          str = t("activity.sentence.canceled");
          break;
        }
      }
      if (activity.creator.id != SYSTEM_BOT_ID) {
        // If creator is not the robot (which means we do NOT use task name in the subject),
        // then we append the task name here.
        const task = findTaskById(issue.pipeline, payload.taskId);
        str += t("activity.sentence.task-name", { name: task.name });
      }
      return str;
    }
    case "bb.pipeline.task.file.commit": {
      const payload = activity.payload as ActivityTaskFileCommitPayload;
      // return `committed ${payload.filePath} to ${payload.branch}@${payload.repositoryFullPath}`;
      return t("activity.sentence.committed-to-at", {
        file: payload.filePath,
        branch: payload.branch,
        repo: payload.repositoryFullPath,
      });
    }
    case "bb.pipeline.task.statement.update": {
      const payload = activity.payload as ActivityTaskStatementUpdatePayload;

      return h(
        Translation,
        {
          keypath: "activity.sentence.changed-from-to",
        },
        {
          name: () => "SQL",
          oldValue: () => renderStatement(payload.oldStatement),
          newValue: () => renderStatement(payload.newStatement),
        }
      );
    }
    case "bb.pipeline.task.general.earliest-allowed-time.update": {
      const payload =
        activity.payload as ActivityTaskEarliestAllowedTimeUpdatePayload;
      const newVal = payload.newEarliestAllowedTs;
      const oldVal = payload.oldEarliestAllowedTs;
      return t("activity.sentence.changed-from-to", {
        name: "earliest allowed time",
        oldValue: oldVal ? dayjs(oldVal * 1000) : "Unset",
        newValue: newVal ? dayjs(newVal * 1000) : "Unset",
      });
    }
  }
  return "";
};

const renderer = {
  render: renderActionSentence,
};

const renderStatement = (statement: string) => {
  return h(SQLPreviewPopover, {
    statement,
    maxLength: 50,
    width: 400,
    statementClass: "text-main",
  });
};
</script>
