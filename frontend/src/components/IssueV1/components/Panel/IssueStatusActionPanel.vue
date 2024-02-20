<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @show="comment = ''"
    @close="$emit('close')"
  >
    <template #default>
      <div v-if="action" class="flex flex-col gap-y-4 px-1">
        <div class="flex flex-col gap-y-1">
          <div class="font-medium text-control">
            {{ $t("common.issue") }}
          </div>
          <div class="textinfolabel">
            {{ issue.title }}
          </div>
        </div>

        <div v-if="issueStatusActionErrors.length > 0" class="flex flex-col">
          <ErrorList
            :errors="issueStatusActionErrors"
            bullets="none"
            class="text-sm"
          >
            <template #prefix>
              <heroicons:exclamation-triangle
                class="text-warning w-4 h-4 inline-block mr-1 mb-px"
              />
            </template>
          </ErrorList>
          <div>
            <NCheckbox v-model:checked="performActionAnyway">
              {{
                $t("issue.action-anyway", {
                  action: issueStatusActionDisplayName(action),
                })
              }}
            </NCheckbox>
          </div>
        </div>

        <div class="flex flex-col gap-y-1">
          <p class="font-medium text-control">
            {{ $t("common.comment") }}
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div v-if="action" class="flex justify-end gap-x-3">
        <NButton @click="$emit('close')">
          {{ $t("common.cancel") }}
        </NButton>
        <NTooltip :disabled="confirmErrors.length === 0" placement="top">
          <template #trigger>
            <NButton
              :disabled="confirmErrors.length > 0"
              v-bind="issueStatusActionButtonProps(action)"
              @click="handleConfirm(action, comment)"
            >
              {{ issueStatusActionDisplayName(action) }}
            </NButton>
          </template>
          <template #default>
            <ErrorList :errors="confirmErrors" />
          </template>
        </NTooltip>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  IssueStatusAction,
  useIssueContext,
  issueStatusActionButtonProps,
  issueStatusActionDisplayName,
  IssueStatusActionToIssueStatusMap,
} from "@/components/IssueV1/logic";
import ErrorList from "@/components/misc/ErrorList.vue";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { flattenTaskV1List } from "@/utils";
import CommonDrawer from "./CommonDrawer.vue";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: IssueStatusAction;
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const { events, issue } = useIssueContext();
const comment = ref("");
const performActionAnyway = ref(false);

const title = computed(() => {
  const { action } = props;

  switch (action) {
    case "RESOLVE":
      return t("issue.status-transition.modal.resolve");
    case "CANCEL":
      return t("issue.status-transition.modal.cancel");
    case "REOPEN":
      return t("issue.status-transition.modal.reopen");
  }
  return "";
});

const issueStatusActionErrors = computed(() => {
  const tasks = flattenTaskV1List(issue.value.rolloutEntity);
  if (tasks.some((task) => task.status === Task_Status.RUNNING)) {
    return [t("issue.status-transition.error.some-tasks-are-still-running")];
  }
  return [];
});

const confirmErrors = computed(() => {
  const errors: string[] = [];
  if (issueStatusActionErrors.value.length > 0 && !performActionAnyway.value) {
    errors.push(...issueStatusActionErrors.value);
  }
  return errors;
});

const handleConfirm = async (
  action: IssueStatusAction,
  comment: string | undefined
) => {
  state.loading = true;
  try {
    await issueServiceClient.batchUpdateIssuesStatus({
      parent: issue.value.project,
      issues: [issue.value.name],
      status: IssueStatusActionToIssueStatusMap[action],
      reason: comment ?? "",
    });

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    emit("close");
  } finally {
    state.loading = false;
  }
};
</script>
