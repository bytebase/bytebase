<template>
  <Renderer />
</template>

<script lang="tsx" setup>
import { defineComponent } from "vue";
import { Translation, useI18n } from "vue-i18n";
import { usePlanContext } from "@/components/Plan/logic";
import { SpecLink } from "@/components/v2";
import {
  extractUserId,
  getIssueCommentType,
  IssueCommentType,
  useUserStore,
} from "@/store";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueComment_Approval_Status,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  extractPlanUID,
  extractProjectResourceName,
  getSpecDisplayInfo,
} from "@/utils";
import StatementUpdate from "./StatementUpdate.vue";

const props = defineProps<{
  issueComment: IssueComment;
}>();

const { t } = useI18n();
const userStore = useUserStore();
const { plan } = usePlanContext();

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

const renderActionSentence = () => {
  const { issueComment } = props;
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
    commentType === IssueCommentType.PLAN_SPEC_UPDATE &&
    issueComment.event?.case === "planSpecUpdate"
  ) {
    const { spec, fromSheet, toSheet } = issueComment.event.value;
    if (fromSheet !== undefined && toSheet !== undefined) {
      const specs = plan.value.specs ?? [];
      const specInfo = getSpecDisplayInfo(specs, spec);
      const planName = plan.value.name;
      const projectName = extractProjectResourceName(planName);
      const planUID = extractPlanUID(planName);

      return (
        <Translation keypath="dynamic.activity.sentence.modified-sql-of-spec-link">
          {{
            spec: () => (
              <SpecLink
                projectName={projectName}
                planUID={planUID}
                specId={specInfo?.specId ?? ""}
                displayIndex={specInfo?.displayIndex ?? 0}
              />
            ),
            link: () => (
              <StatementUpdate oldSheet={fromSheet} newSheet={toSheet} />
            ),
          }}
        </Translation>
      );
    }
  }
  return "";
};

const Renderer = defineComponent({
  name: "ActionSentenceRenderer",
  render: () => <span>{renderActionSentence()}</span>,
});
</script>
