<template>
  <div class="mt-2 flex flex-col gap-y-4">
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
    <div class="py-1 flex justify-end gap-x-3">
      <NButton @click="$emit('cancel')">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton
        v-bind="issueStatusActionButtonProps(action)"
        @click="$emit('confirm', action, comment)"
      >
        {{ issueStatusActionDisplayName(action) }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import {
  IssueStatusAction,
  issueStatusActionButtonProps,
  issueStatusActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";

defineProps<{
  action: IssueStatusAction;
}>();

defineEmits<{
  (event: "cancel"): void;
  (event: "confirm", action: IssueStatusAction, comment?: string): void;
}>();

const { issue } = useIssueContext();
const comment = ref("");
</script>
