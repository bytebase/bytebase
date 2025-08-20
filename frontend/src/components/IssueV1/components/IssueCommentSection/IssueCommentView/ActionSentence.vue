<template>
  <Renderer />
</template>

<script lang="tsx" setup>
import type { h } from "vue";
import { defineComponent } from "vue";
import { Translation, useI18n } from "vue-i18n";
import { useUserStore, IssueCommentType, getIssueCommentType } from "@/store";
import { extractUserId } from "@/store";
import { type ComposedIssue } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueComment_Approval_Status,
  IssueComment_TaskUpdate_Status,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { findStageByName, findTaskByName } from "@/utils";
import StageName from "./StageName.vue";
import StatementUpdate from "./StatementUpdate.vue";
import TaskName from "./TaskName.vue";

type RenderedContent = string | ReturnType<typeof h>;

const props = defineProps<{
  issue: ComposedIssue;
  issueComment: IssueComment;
}>();

const { t } = useI18n();
const userStore = useUserStore();

const renderActionSentence = () => {
  const { issueComment, issue } = props;
  const commentType = getIssueCommentType(issueComment);
  if (
    commentType === IssueCommentType.APPROVAL &&
    issueComment.event?.case === "approval"
  ) {
    const { status } = issueComment.event.value;
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
  } else if (
    commentType === IssueCommentType.ISSUE_UPDATE &&
    issueComment.event?.case === "issueUpdate"
  ) {
    const {
      fromTitle,
      toTitle,
      fromDescription,
      toDescription,
      fromStatus,
      toStatus,
      fromLabels,
      toLabels,
    } = issueComment.event.value;
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
  } else if (
    commentType === IssueCommentType.STAGE_END &&
    issueComment.event?.case === "stageEnd"
  ) {
    const { stage } = issueComment.event.value;
    const stageEntity = findStageByName(issue.rolloutEntity, stage);
    const params: VerbTypeTarget = {
      issueComment,
      type: t("common.stage"),
      target: <StageName stage={stageEntity} />,
      verb: t("activity.sentence.completed"),
    };
    return renderVerbTypeTarget(params);
  } else if (
    commentType === IssueCommentType.TASK_UPDATE &&
    issueComment.event?.case === "taskUpdate"
  ) {
    const { tasks, fromSheet, toSheet, toStatus } = issueComment.event.value;

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
    }
  } else if (
    commentType === IssueCommentType.TASK_PRIOR_BACKUP &&
    issueComment.event?.case === "taskPriorBackup"
  ) {
    const { task, tables, originalLine, database, error } =
      issueComment.event.value;
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
  issueComment: IssueComment,
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
  issueComment: IssueComment;
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
