<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @close="$emit('close')"
  >
    <template #default>
      <div v-if="action" class="flex flex-col gap-y-4">
        <div class="text-sm">
          {{ issue.title }}
        </div>
        <div class="flex flex-col gap-y-1">
          <p class="textlabel">
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
      <div v-if="action" class="py-1 flex justify-end gap-x-3">
        <NButton @click="$emit('close')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          v-bind="issueStatusActionButtonProps(action) !== undefined"
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
  useIssueContext,
  issueStatusActionButtonProps,
  issueStatusActionDisplayName,
} from "@/components/IssueV1/logic";
import { pushNotification } from "@/store";
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

const handleConfirm = async (
  action: IssueStatusAction,
  comment: string | undefined
) => {
  state.loading = true;
  // TODO
  try {
    await new Promise((r) => setTimeout(r, 1000));

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.loading = false;
    emit("close");
  }

  // Trying to avoid some kind of concurrency and race condition, we fetch the
  // latest snapshot of issue from the server-side and check whether this
  // transition is applicable again.
  // const latestIssue = await useIssueStore().fetchIssueById(issue.value.id);

  // const { action: transition } = props;
  // const applicableList = getApplicableIssueStatusTransitionList(latestIssue);
  // if (!isApplicableTransition(transition, applicableList)) {
  //   return cleanup();
  // }

  // changeIssueStatus(transition.to, comment);
  // isTransiting.value = false;
  // emit("updated");
};
</script>
