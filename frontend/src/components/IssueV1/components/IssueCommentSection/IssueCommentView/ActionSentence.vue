<template>
  <Renderer />
</template>

<script lang="tsx" setup>
import dayjs from "dayjs";
import type { h } from "vue";
import { defineComponent } from "vue";
import { Translation, useI18n } from "vue-i18n";
import {
  useUserStore,
  IssueCommentType,
  type ComposedIssueComment,
} from "@/store";
import { extractUserId } from "@/store";
import { getDateForPbTimestamp, type ComposedIssue } from "@/types";
import {
  IssueComment_Approval,
  IssueComment_Approval_Status,
  IssueComment_IssueUpdate,
  IssueComment_StageEnd,
  IssueComment_TaskPriorBackup,
  IssueComment_TaskUpdate,
  IssueComment_TaskUpdate_Status,
  IssueStatus,
} from "@/types/proto/v1/issue_service";
import { findStageByName, findTaskByName } from "@/utils";
import StageName from "./StageName.vue";
import StatementUpdate from "./StatementUpdate.vue";
import TaskName from "./TaskName.vue";

type RenderedContent = string | ReturnType<typeof h>;

const props = defineProps<{
  issue: ComposedIssue;
  issueComment: ComposedIssueComment;
}>();

const { t } = useI18n();
const userStore = useUserStore();

const renderActionSentence = () => {
  const { issueComment, issue } = props;
  if (issueComment.type === IssueCommentType.APPROVAL) {
    const { status } = IssueComment_Approval.fromPartial(
      issueComment.approval || {}
    );
    let verb = "";
    if (status === IssueComment_Approval_Status.APPROVED) {
      verb = t("custom-approval.issue-review.approved-issue");
    } else if (status === IssueComment_Approval_Status.REJECTED) {
      verb = t("custom-approval.issue-review.sent-back-issue");
    } else if (status === IssueComment_Approval_Status.PENDING) {
      verb = t("custom-approval.issue-review.re-requested-review");
    }
    if (verb) {
      return maybeAutomaticallyVerb(issueComment, verb);
    }
  } else if (issueComment.type === IssueCommentType.ISSUE_UPDATE) {
    const {
      fromTitle,
      toTitle,
      fromDescription,
      toDescription,
      fromStatus,
      toStatus,
      fromLabels,
      toLabels,
    } = IssueComment_IssueUpdate.fromPartial(issueComment.issueUpdate || {});
    if (fromTitle !== undefined && toTitle !== undefined) {
      return t("activity.sentence.changed-from-to", {
        name: t("issue.issue-name").toLowerCase(),
        oldValue: fromTitle,
        newValue: toTitle,
      });
    } else if (fromDescription !== undefined && toDescription !== undefined) {
      // Description could be very long, so we don't display it.
      return t("activity.sentence.changed-description");
    } else if (fromStatus !== undefined && toStatus !== undefined) {
      if (toStatus === IssueStatus.DONE) {
        return t("activity.sentence.resolved-issue");
      } else if (toStatus === IssueStatus.CANCELED) {
        return t("activity.sentence.canceled-issue");
      } else if (toStatus === IssueStatus.OPEN) {
        return t("activity.sentence.reopened-issue");
      }
    } else if (fromLabels.length !== 0 || toLabels.length !== 0) {
      return t("activity.sentence.changed-labels");
    }
  } else if (issueComment.type === IssueCommentType.STAGE_END) {
    const { stage } = IssueComment_StageEnd.fromPartial(
      issueComment.stageEnd || {}
    );
    const stageEntity = findStageByName(issue.rolloutEntity, stage);
    const params: VerbTypeTarget = {
      issueComment,
      type: t("common.stage"),
      target: <StageName stage={stageEntity} issue={issue} />,
      verb: t("activity.sentence.completed"),
    };
    return renderVerbTypeTarget(params);
  } else if (issueComment.type === IssueCommentType.TASK_UPDATE) {
    const {
      tasks,
      fromSheet,
      toSheet,
      fromEarliestAllowedTime,
      toEarliestAllowedTime,
      toStatus,
    } = IssueComment_TaskUpdate.fromPartial(issueComment.taskUpdate || {});

    if (toStatus !== undefined) {
      const params: VerbTypeTarget = {
        issueComment,
        verb: "",
        type: t("common.task"),
        target: "",
      };
      switch (toStatus) {
        case IssueComment_TaskUpdate_Status.PENDING: {
          params.verb = maybeAutomaticallyVerb(
            issueComment,
            t("activity.sentence.rolled-out")
          );
          break;
        }
        case IssueComment_TaskUpdate_Status.RUNNING: {
          params.verb = t("activity.sentence.started");
          break;
        }
        case IssueComment_TaskUpdate_Status.DONE: {
          params.verb = t("activity.sentence.completed");
          break;
        }
        case IssueComment_TaskUpdate_Status.FAILED: {
          params.verb = t("activity.sentence.failed");
          break;
        }
        case IssueComment_TaskUpdate_Status.CANCELED: {
          params.verb = t("activity.sentence.canceled");
          break;
        }
        case IssueComment_TaskUpdate_Status.SKIPPED: {
          params.verb = t("activity.sentence.skipped");
          break;
        }
        default:
          params.verb = t("activity.sentence.changed");
      }
      const taskEntities = tasks.map((task) =>
        findTaskByName(issue.rolloutEntity, task)
      );
      if (taskEntities.length > 0) {
        params.target = <TaskName issue={issue} task={taskEntities[0]} />;
      }
      return renderVerbTypeTarget(params);
    } else if (fromSheet !== undefined && toSheet !== undefined) {
      return (
        <Translation keypath="dynamic.activity.sentence.changed-x-link">
          {{
            name: () => "SQL",
            link: () => (
              <StatementUpdate oldSheet={fromSheet} newSheet={toSheet} />
            ),
          }}
        </Translation>
      );
    } else if (
      fromEarliestAllowedTime !== undefined ||
      toEarliestAllowedTime !== undefined
    ) {
      const oldVal = fromEarliestAllowedTime;
      const newVal = toEarliestAllowedTime;
      const timeFormat = "YYYY-MM-DD HH:mm:ss UTCZZ";
      return t("activity.sentence.changed-from-to", {
        name: t("task.earliest-allowed-time"),
        oldValue: oldVal
          ? dayjs(getDateForPbTimestamp(oldVal)).format(timeFormat)
          : "Unset",
        newValue: newVal
          ? dayjs(getDateForPbTimestamp(newVal)).format(timeFormat)
          : "Unset",
      });
    }
  } else if (issueComment.type === IssueCommentType.TASK_PRIOR_BACKUP) {
    const { task, tables, originalLine, database, error } =
      IssueComment_TaskPriorBackup.fromPartial(
        issueComment.taskPriorBackup || {}
      );
    if (error) {
      return t("activity.sentence.failed-to-backup", { error });
    }

    const taskEntity = findTaskByName(issue.rolloutEntity, task);
    let verb = t("activity.sentence.prior-back-table", {
      database: database.length > 0 ? database : "bbdataarchive",
      tables: tables
        .map((table) =>
          table.schema ? `${table.schema}.${table.table}` : table.table
        )
        .join(", "),
    });
    if (originalLine) {
      verb = t("activity.sentence.prior-back-table-for-line", {
        database: database.length > 0 ? database : "bbdataarchive",
        tables: tables
          .map((table) =>
            table.schema ? `${table.schema}.${table.table}` : table.table
          )
          .join(", "),
        line: originalLine,
      });
    }
    const params: VerbTypeTarget = {
      issueComment,
      verb: verb,
      type: t("common.task"),
      target: "",
    };
    if (task) {
      params.target = <TaskName issue={issue} task={taskEntity} />;
    }
    return renderVerbTypeTarget(params);
  }
  return "";
};

const maybeAutomaticallyVerb = (
  issueComment: ComposedIssueComment,
  verb: string
): string => {
  if (extractUserId(issueComment.creator) !== userStore.systemBotUser?.email) {
    return verb;
  }
  return t("activity.sentence.xxx-automatically", {
    verb,
  });
};

type VerbTypeTarget = {
  issueComment: ComposedIssueComment;
  verb: RenderedContent;
  type: RenderedContent;
  target?: RenderedContent;
};

const renderVerbTypeTarget = (params: VerbTypeTarget, props: object = {}) => {
  const keypath =
    extractUserId(params.issueComment.creator) ===
    userStore.systemBotUser?.email
      ? "dynamic.activity.sentence.verb-type-target-by-system-bot"
      : "dynamic.activity.sentence.verb-type-target-by-people";
  return (
    <Translation {...props} keypath={keypath}>
      {{
        verb: () => params.verb,
        type: () => params.type,
        target: () => params.target,
      }}
    </Translation>
  );
};

const Renderer = defineComponent({
  name: "ActionSentenceRenderer",
  render: () => <span>{renderActionSentence()}</span>,
});
</script>
