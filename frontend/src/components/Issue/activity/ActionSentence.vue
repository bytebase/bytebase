<template>
  <Renderer />
</template>

<script lang="ts" setup>
import { defineComponent, h, PropType, watch } from "vue";
import dayjs from "dayjs";
import { Translation, useI18n } from "vue-i18n";

import {
  Activity,
  ActivityIssueCommentCreatePayload,
  ActivityStageStatusUpdatePayload,
  ActivityTaskEarliestAllowedTimeUpdatePayload,
  ActivityTaskFileCommitPayload,
  ActivityTaskStatementUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  Issue,
  SYSTEM_BOT_ID,
} from "@/types";
import {
  findStageById,
  findTaskById,
  issueActivityActionSentence,
} from "@/utils";
import { useSheetV1Store, useSheetStatementByUid } from "@/store";
import TaskName from "./TaskName.vue";
import SQLPreviewPopover from "@/components/misc/SQLPreviewPopover.vue";
import StageName from "./StageName.vue";

type RenderedContent = string | ReturnType<typeof h>;

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
const sheetV1Store = useSheetV1Store();

const renderActionSentence = () => {
  const renderSpan = (content: string, props?: object) => {
    return h("span", props, content);
  };

  const { activity, issue } = props;
  if (activity.type.startsWith("bb.issue.")) {
    if (activity.type === "bb.issue.comment.create") {
      const payload = activity.payload as ActivityIssueCommentCreatePayload;
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
      const task = findTaskById(issue.pipeline, payload.taskId);
      if (task) {
        params.target = h(TaskName, { issue, task });
      }
      return renderVerbTypeTarget(params, {
        tag: "span",
      });
    }
    case "bb.pipeline.stage.status.update": {
      const payload = activity.payload as ActivityStageStatusUpdatePayload;
      const stage = findStageById(issue.pipeline, payload.stageId);
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
      const oldStatement =
        useSheetStatementByUid(payload.oldSheetId).value ||
        payload.oldStatement;
      const newStatement =
        useSheetStatementByUid(payload.newSheetId).value ||
        payload.newStatement;
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

const maybeAutomaticallyVerb = (activity: Activity, verb: string): string => {
  if (activity.creator.id !== SYSTEM_BOT_ID) {
    return verb;
  }
  return t("activity.sentence.xxx-automatically", {
    verb,
  });
};

type VerbTypeTarget = {
  activity: Activity;
  verb: RenderedContent;
  type: RenderedContent;
  target?: RenderedContent;
};

const renderVerbTypeTarget = (params: VerbTypeTarget, props: object = {}) => {
  const keypath =
    params.activity.creator.id === SYSTEM_BOT_ID
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

const renderStatement = (statement: string) => {
  return h(SQLPreviewPopover, {
    statement,
    maxLength: 50,
    width: 400,
    statementClass: "text-main",
  });
};

watch(
  () => props.activity,
  async () => {
    const activity = props.activity;
    // Prepare sheet data for renderering.
    if (activity.type === "bb.pipeline.task.statement.update") {
      sheetV1Store.getOrFetchSheetByUid(
        (activity.payload as ActivityTaskStatementUpdatePayload).newSheetId
      );
      sheetV1Store.getOrFetchSheetByUid(
        (activity.payload as ActivityTaskStatementUpdatePayload).oldSheetId
      );
    }
  },
  {
    immediate: true,
  }
);
</script>
