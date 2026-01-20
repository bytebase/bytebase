<template>
  <NTooltip>
    <template #trigger>
      <span
        class="flex items-center justify-center rounded-full select-none overflow-hidden shrink-0"
        :class="issueIconClass()"
      >
        <template v-if="issueStatus === IssueStatus.OPEN">
          <span
            class="h-1.5 w-1.5 bg-info rounded-full"
            aria-hidden="true"
          ></span>
        </template>
        <template v-else-if="issueStatus === IssueStatus.DONE">
          <heroicons-solid:check class="w-4 h-4" />
        </template>
        <template v-else-if="issueStatus === IssueStatus.CANCELED">
          <heroicons-solid:minus class="w-5 h-5" />
        </template>
      </span>
    </template>

    <template #default>
      <div class="max-w-[24rem]">
        {{
          isUnfinishedResolvedIssue
            ? $t("issue.unfinished-resolved-issue-tips")
            : stringifyIssueStatus(props.issueStatus)
        }}
      </div>
    </template>
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import type { PropType } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { isUnfinishedResolvedTask as checkUnfinishedResolvedTask } from "@/utils";

export type SizeType = "small" | "normal";

const props = defineProps({
  issueStatus: {
    required: true,
    type: Number as PropType<IssueStatus>,
  },
  size: {
    type: String as PropType<SizeType>,
    default: "normal",
  },
  issue: {
    type: Object as PropType<Issue>,
    default: undefined,
  },
});

const { t } = useI18n();

const issueIconClass = () => {
  const iconClass = props.size === "normal" ? "w-5 h-5" : "w-4 h-4";
  switch (props.issueStatus) {
    case IssueStatus.OPEN:
      return iconClass + " bg-white border-2 border-info text-info";
    case IssueStatus.CANCELED:
      return iconClass + " bg-white border-2 text-gray-400 border-gray-400";
    case IssueStatus.DONE:
      return (
        iconClass +
        " text-white" +
        (isUnfinishedResolvedIssue.value ? " bg-warning" : " bg-success")
      );
  }
};

const isUnfinishedResolvedIssue = computed(() => {
  // In list contexts, we don't have rollout data, so pass undefined
  return checkUnfinishedResolvedTask(props.issue, undefined);
});

const stringifyIssueStatus = (issueStatus: IssueStatus): string => {
  if (issueStatus === IssueStatus.OPEN) {
    return t("issue.table.open");
  } else if (issueStatus === IssueStatus.DONE) {
    return t("common.done");
  } else if (issueStatus === IssueStatus.CANCELED) {
    return t("common.closed");
  }
  return "";
};
</script>
