<template>
  <Renderer />
</template>

<script lang="ts" setup>
import { defineComponent, h, watch } from "vue";
import dayjs from "dayjs";
import { Translation, useI18n } from "vue-i18n";

import {
  ActivityIssueCommentCreatePayload,
  ActivityStageStatusUpdatePayload,
  ActivityTaskEarliestAllowedTimeUpdatePayload,
  ActivityTaskFileCommitPayload,
  ActivityTaskStatementUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  ComposedIssue,
  SYSTEM_BOT_EMAIL,
  UNKNOWN_ID,
} from "@/types";
import {
  findStageByUID,
  findTaskByUID,
  issueActivityActionSentence,
} from "@/utils";
import { useSheetV1Store, useSheetStatementByUID } from "@/store";
import TaskName from "./TaskName.vue";
import TextOverflowPopover from "@/components/misc/TextOverflowPopover.vue";
import StageName from "./StageName.vue";
import { LogEntity, LogEntity_Action } from "@/types/proto/v1/logging_service";
import { extractUserResourceName } from "@/utils";

type RenderedContent = string | ReturnType<typeof h>;

const props = defineProps<{
  activity: LogEntity;
  issue: ComposedIssue;
}>();

const { t } = useI18n();
const sheetV1Store = useSheetV1Store();

const renderActionSentence = () => {
  const renderSpan = (content: string, props?: object) => {
    return h("span", props, content);
  };

  const { activity, issue } = props;
  if (activity.resource.startsWith("issues")) {
    if (activity.action === LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE) {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityIssueCommentCreatePayload;
      if (payload.externalApprovalEvent) {
        if (payload.externalApprovalEvent.action == "REJECT") {
          let imName = "";
          switch (payload.externalApprovalEvent.type) {
            case "bb.plugin.app.feishu":
              imName = t("common.feishu");
              break;
          }
          return renderSpan(
            t("activity.sentence.external-approval-rejected", {
              stageName: payload.externalApprovalEvent.stageName,
              imName: imName,
            })
          );
        }
      }
      if (payload.approvalEvent) {
        if (payload.approvalEvent) {
          const { status } = payload.approvalEvent;
          const dict: Record<typeof status, string> = {
            APPROVED: t("custom-approval.issue-review.approved-issue"),
            REJECTED: t("custom-approval.issue-review.sent-back-issue"),
            PENDING: t("custom-approval.issue-review.re-requested-review"),
          };
          const verb = dict[status];
          if (verb) {
            return renderSpan(maybeAutomaticallyVerb(activity, verb));
          }
        }
      }
    }

    const [tid, params] = issueActivityActionSentence(activity);
    return renderSpan(t(tid, params));
  }
  switch (activity.action) {
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityTaskStatusUpdatePayload;
      if (payload.newStatus === "PENDING_APPROVAL") {
        // stale approval dismissed.

        const task = findTaskByUID(issue.rolloutEntity, String(payload.taskId));
        const taskName = t("activity.sentence.task-name", {
          name: task.title,
        });
        return t("activity.sentence.dismissed-stale-approval", {
          task: taskName,
        });
      }
      const params: VerbTypeTarget = {
        activity,
        verb: "",
        type: t("common.task"),
        target: "",
      };
      switch (payload.newStatus) {
        case "PENDING": {
          if (payload.oldStatus == "RUNNING") {
            params.verb = t("activity.sentence.canceled");
          } else if (payload.oldStatus == "PENDING_APPROVAL") {
            params.verb = maybeAutomaticallyVerb(
              activity,
              t("activity.sentence.rolled-out")
            );
          }
          break;
        }
        case "RUNNING": {
          params.verb = t("activity.sentence.started");
          break;
        }
        case "DONE": {
          if (payload.oldStatus === "RUNNING") {
            params.verb = t("activity.sentence.completed");
          } else {
            params.verb = t("activity.sentence.skipped");
          }
          break;
        }
        case "FAILED": {
          params.verb = t("activity.sentence.failed");
          break;
        }
        case "CANCELED": {
          params.verb = t("activity.sentence.canceled");
          break;
        }
        default:
          params.verb = t("activity.sentence.changed");
      }
      const task = findTaskByUID(issue.rolloutEntity, String(payload.taskId));
      if (task) {
        params.target = h(TaskName, { issue, task });
      }
      return renderVerbTypeTarget(params, {
        tag: "span",
      });
    }
    case LogEntity_Action.ACTION_PIPELINE_STAGE_STATUS_UPDATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityStageStatusUpdatePayload;
      const stage = findStageByUID(
        issue.rolloutEntity,
        String(payload.stageId)
      );
      const params: VerbTypeTarget = {
        activity,
        verb: "",
        type: t("common.stage"),
        target: h(StageName, { stage, issue }),
      };
      switch (payload.stageStatusUpdateType) {
        case "BEGIN":
          params.verb = maybeAutomaticallyVerb(
            activity,
            t("activity.sentence.started")
          );
          break;
        case "END":
          params.verb = t("activity.sentence.completed");
          break;
        default:
          params.verb = t("activity.sentence.changed");
          break;
      }
      return renderVerbTypeTarget(params, {
        tag: "span",
      });
    }
    case LogEntity_Action.ACTION_PIPELINE_TASK_FILE_COMMIT: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityTaskFileCommitPayload;
      // return `committed ${payload.filePath} to ${payload.branch}@${payload.repositoryFullPath}`;
      return t("activity.sentence.committed-to-at", {
        file: payload.filePath,
        branch: payload.branch,
        repo: payload.repositoryFullPath,
      });
    }
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityTaskStatementUpdatePayload;
      const oldStatement =
        useSheetStatementByUID(String(payload.oldSheetId || UNKNOWN_ID))
          .value || payload.oldStatement;
      const newStatement =
        useSheetStatementByUID(String(payload.newSheetId || UNKNOWN_ID))
          .value || payload.newStatement;
      return h(
        "span",
        {},
        h(
          Translation,
          {
            keypath: "activity.sentence.changed-from-to",
          },
          {
            name: () => "SQL",
            oldValue: () => renderStatement(oldStatement),
            newValue: () => renderStatement(newStatement),
          }
        )
      );
    }
    case LogEntity_Action.ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityTaskEarliestAllowedTimeUpdatePayload;
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

const maybeAutomaticallyVerb = (activity: LogEntity, verb: string): string => {
  if (extractUserResourceName(activity.creator) !== SYSTEM_BOT_EMAIL) {
    return verb;
  }
  return t("activity.sentence.xxx-automatically", {
    verb,
  });
};

type VerbTypeTarget = {
  activity: LogEntity;
  verb: RenderedContent;
  type: RenderedContent;
  target?: RenderedContent;
};

const renderVerbTypeTarget = (params: VerbTypeTarget, props: object = {}) => {
  const keypath =
    extractUserResourceName(params.activity.creator) === SYSTEM_BOT_EMAIL
      ? "activity.sentence.verb-type-target-by-system-bot"
      : "activity.sentence.verb-type-target-by-people";
  return h(
    Translation,
    {
      ...props,
      keypath,
    },
    {
      verb: () => params.verb,
      type: () => params.type,
      target: () => params.target,
    }
  );
};

const Renderer = defineComponent({
  name: "ActionSentenceRenderer",
  render: renderActionSentence,
});

const renderStatement = (content: string) => {
  return h(TextOverflowPopover, {
    statement: content,
    maxLength: 50,
    width: 400,
    contentClass: "text-main",
  });
};

watch(
  () => props.activity,
  async () => {
    const activity = props.activity;
    // Prepare sheet data for rendering.
    if (
      activity.action === LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE
    ) {
      sheetV1Store.getOrFetchSheetByUID(
        String(
          (JSON.parse(activity.payload) as ActivityTaskStatementUpdatePayload)
            .newSheetId || UNKNOWN_ID
        )
      );
      sheetV1Store.getOrFetchSheetByUID(
        String(
          (JSON.parse(activity.payload) as ActivityTaskStatementUpdatePayload)
            .oldSheetId || UNKNOWN_ID
        )
      );
    }
  },
  {
    immediate: true,
  }
);
</script>
