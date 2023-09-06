<template>
  <BBTooltipButton
    type="primary"
    :disabled="!isTransitionApplicableForAllIssues('RESOLVE')"
    tooltip-mode="DISABLED-ONLY"
    @click="startBatchIssueTransition('RESOLVE')"
  >
    {{ $t("issue.batch-transition.resolve") }}
    <template #tooltip>
      <div class="whitespace-nowrap">
        {{
          $t("issue.batch-transition.not-allowed-tips", {
            operation: $t("issue.batch-transition.resolved"),
          })
        }}
      </div>
    </template>
  </BBTooltipButton>

  <BBTooltipButton
    type="normal"
    :disabled="!isTransitionApplicableForAllIssues('CANCEL')"
    tooltip-mode="DISABLED-ONLY"
    @click="startBatchIssueTransition('CANCEL')"
  >
    {{ $t("issue.batch-transition.cancel") }}
    <template #tooltip>
      <div class="whitespace-nowrap">
        {{
          $t("issue.batch-transition.not-allowed-tips", {
            operation: $t("issue.batch-transition.cancelled"),
          })
        }}
      </div>
    </template>
  </BBTooltipButton>

  <BBTooltipButton
    type="normal"
    :disabled="!isTransitionApplicableForAllIssues('REOPEN')"
    tooltip-mode="DISABLED-ONLY"
    @click="startBatchIssueTransition('REOPEN')"
  >
    {{ $t("issue.batch-transition.reopen") }}
    <template #tooltip>
      <div class="whitespace-nowrap">
        {{
          $t("issue.batch-transition.not-allowed-tips", {
            operation: $t("issue.batch-transition.reopened"),
          })
        }}
      </div>
    </template>
  </BBTooltipButton>

  <BBModal
    v-if="state.modal.show"
    :title="state.modal.title"
    class="relative overflow-hidden"
    @close="state.modal.show = false"
  >
    <div
      v-if="state.isRequesting"
      class="absolute inset-0 flex flex-col items-center justify-center bg-white/70"
    >
      <BBSpin />
      <div class="flex items-center textlabel">
        <span>{{ $t("common.updating") }}</span>
        <span v-if="state.stats"
          >({{ state.stats.success }} / {{ state.stats.total }})</span
        >
      </div>
    </div>

    <div>
      <div class="sm:col-span-4 w-112 min-w-full">
        <label for="about" class="textlabel">
          {{ $t("issue.status-transition.form.note") }}
        </label>
        <div class="mt-1">
          <textarea
            ref="commentTextArea"
            v-model="state.modal.comment"
            rows="3"
            class="textarea block w-full resize-none mt-1 text-sm text-control rounded-md whitespace-pre-wrap"
            :placeholder="$t('issue.status-transition.form.placeholder')"
            @input="
              (e) => {
                sizeToFit(e.target as HTMLTextAreaElement);
              }
            "
            @focus="
              (e) => {
                sizeToFit(e.target as HTMLTextAreaElement);
              }
            "
          ></textarea>
        </div>
      </div>
    </div>

    <!-- Update button group -->
    <div class="flex justify-end items-center pt-5">
      <button
        type="button"
        class="btn-normal mt-3 px-4 py-2 sm:mt-0 sm:w-auto"
        @click.prevent="state.modal.show = false"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        type="button"
        class="ml-3 px-4 py-2"
        :class="state.modal.okButtonClass"
        @click="doBatchIssueTransition(state.modal.transition!)"
      >
        {{ state.modal.okButtonText }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, refreshIssueList, useIssueStore } from "@/store";
import type {
  ComposedIssue,
  IssueStatusPatch,
  IssueStatusTransition,
  IssueStatusTransitionType,
} from "@/types";
import { ISSUE_STATUS_TRANSITION_LIST } from "@/types";
import { extractIssueUID } from "@/utils";
import { calcApplicableIssueStatusTransitionList } from "../logic/transition";

type RequestStats = {
  total: number;
  success: number;
  failed: number;
};

type ModalProps = {
  show: boolean;
  title: string;
  comment: string;
  transition?: IssueStatusTransition;
  okButtonText: string;
  okButtonClass?: string;
};

type LocalState = {
  isRequesting: boolean;
  modal: ModalProps;
  stats?: RequestStats;
};

const props = defineProps({
  issueList: {
    type: Array as PropType<ComposedIssue[]>,
    default: () => [],
  },
});

const state = reactive<LocalState>({
  isRequesting: false,
  modal: {
    show: false,
    title: "",
    comment: "",
    okButtonText: "",
  },
});

const issueStore = useIssueStore();
const { t } = useI18n();

const issueTransitionList = computed(() => {
  return props.issueList.map((issue) => {
    const transitions = calcApplicableIssueStatusTransitionList(issue);
    return { issue, transitions };
  });
});

const isTransitionApplicableForAllIssues = (
  type: IssueStatusTransitionType
): boolean => {
  return issueTransitionList.value.every((item) => {
    return (
      item.transitions.findIndex((transition) => transition.type === type) >= 0
    );
  });
};

const startBatchIssueTransition = (type: IssueStatusTransitionType) => {
  const { modal } = state;
  modal.show = true;
  const transition = ISSUE_STATUS_TRANSITION_LIST.get(type)!;
  modal.transition = transition;
  modal.comment = "";
  modal.okButtonClass = transition.buttonClass;
  modal.okButtonText = t(transition.buttonName);
  modal.title = t("issue.batch-transition.action-n-issues", {
    action: t(`issue.batch-transition.${type.toLowerCase()}`),
    n: props.issueList.length,
  });
};

const doBatchIssueTransition = async (transition: IssueStatusTransition) => {
  const issueStatusPatch: IssueStatusPatch = {
    status: transition.to,
    comment: state.modal.comment,
  };

  const stats = reactive<RequestStats>({
    total: props.issueList.length,
    success: 0,
    failed: 0,
  });

  const doSingleIssueTransition = async (
    issue: ComposedIssue,
    index: number
  ) => {
    const request = issueStore.updateIssueStatus({
      issueId: extractIssueUID(issue.name),
      issueStatusPatch,
    });
    request.then(
      () => stats.success++,
      () => stats.failed++
    );

    return request;
  };

  state.isRequesting = true;
  state.stats = stats;
  try {
    const requestList = props.issueList.map(doSingleIssueTransition);
    await Promise.allSettled(requestList);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("issue.batch-transition.successfully-updated-n-issues", {
        n: stats.success,
      }),
    });
  } finally {
    state.isRequesting = false;
    refreshIssueList();
  }
};
</script>
