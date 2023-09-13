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
            {{
              issueList.length === 1 ? $t("common.issue") : $t("common.issues")
            }}
          </div>
          <div class="textinfolabel">
            <ul
              class="flex flex-col"
              :class="[issueList.length > 1 && 'list-disc pl-4']"
            >
              <li v-for="issue in issueList" :key="issue.uid">
                {{ issue.title }}
              </li>
            </ul>
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
        <NButton
          v-bind="issueStatusActionButtonProps(action)"
          @click="handleConfirm(action, comment)"
        >
          {{ issueStatusActionDisplayName(action) }}
        </NButton>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  IssueStatusAction,
  issueStatusActionButtonProps,
  issueStatusActionDisplayName,
  IssueStatusActionToIssueStatusMap,
} from "@/components/IssueV1/logic";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import { ComposedIssue } from "@/types";
import CommonDrawer from "./CommonDrawer.vue";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  issueList: ComposedIssue[];
  action?: IssueStatusAction;
}>();

const emit = defineEmits<{
  (event: "updating"): void;
  (event: "updated"): void;
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const comment = ref("");

const title = computed(() => {
  const { action } = props;
  if (!action) return "";

  return t("issue.batch-transition.action-n-issues", {
    action: issueStatusActionDisplayName(action),
    n: props.issueList.length,
  });
});

const handleConfirm = async (
  action: IssueStatusAction,
  comment: string | undefined
) => {
  state.loading = true;

  try {
    emit("updating");
    await issueServiceClient.batchUpdateIssuesStatus({
      parent: "projects/-",
      issues: props.issueList.map((issue) => issue.name),
      status: IssueStatusActionToIssueStatusMap[action],
      reason: comment ?? "",
    });

    // notify the parent component that issues updated
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    emit("updated");
  } finally {
    state.loading = false;
  }
};
</script>
