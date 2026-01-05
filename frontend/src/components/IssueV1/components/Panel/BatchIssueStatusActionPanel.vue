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
              <li v-for="issue in issueList" :key="issue.name">
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
          v-bind="confirmButtonProps"
          @click="handleConfirm(action, comment)"
        >
          {{ $t("common.confirm") }}
        </NButton>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton, NInput } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { IssueStatusAction } from "@/components/IssueV1/logic";
import {
  IssueStatusActionToIssueStatusMap,
  issueStatusActionButtonProps,
  issueStatusActionDisplayName,
} from "@/components/IssueV1/logic";
import { issueServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store";
import type { ComposedIssue } from "@/types";
import { BatchUpdateIssuesStatusRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
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

  return issueStatusActionDisplayName(action, props.issueList.length);
});

const confirmButtonProps = computed(() => {
  if (!props.action) return {};
  const p = issueStatusActionButtonProps(props.action);
  if (p.type === "default") {
    p.type = "primary";
  }
  return p;
});

const handleConfirm = async (
  action: IssueStatusAction,
  comment: string | undefined
) => {
  state.loading = true;

  try {
    emit("updating");
    const request = create(BatchUpdateIssuesStatusRequestSchema, {
      parent: "projects/-",
      issues: props.issueList.map((issue) => issue.name),
      status: IssueStatusActionToIssueStatusMap[action],
      reason: comment ?? "",
    });
    await issueServiceClientConnect.batchUpdateIssuesStatus(request);

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
