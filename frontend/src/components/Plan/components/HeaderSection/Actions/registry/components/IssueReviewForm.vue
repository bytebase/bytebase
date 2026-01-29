<template>
  <div class="flex flex-col gap-y-4">
    <MarkdownEditor
      mode="editor"
      :content="comment"
      :project="project"
      :compact="compact"
      :maxlength="65536"
      @change="(val: string) => (comment = val)"
    />

    <NRadioGroup
      v-model:value="selectedAction"
      :disabled="loading"
      class="flex! flex-col gap-y-2"
    >
      <NRadio value="COMMENT">
        <div class="flex items-start gap-2">
          <MessageCircleIcon class="w-4 h-4 mt-0.5 text-gray-600 shrink-0" />
          <div class="flex flex-col">
            <span class="font-medium">{{ $t("common.comment") }}</span>
            <span class="text-control-light text-xs">
              {{ $t("issue.review.comment-description") }}
            </span>
          </div>
        </div>
      </NRadio>

      <NRadio v-if="canApprove" value="APPROVE">
        <div class="flex items-start gap-2">
          <CheckIcon class="w-4 h-4 mt-0.5 text-green-600 shrink-0" />
          <div class="flex flex-col">
            <span class="font-medium">{{ $t("common.approve") }}</span>
            <span class="text-control-light text-xs">
              {{ $t("issue.review.approve-description") }}
            </span>
          </div>
        </div>
      </NRadio>

      <NRadio v-if="canReject" value="REJECT">
        <div class="flex items-start gap-2">
          <XIcon class="w-4 h-4 mt-0.5 text-red-600 shrink-0" />
          <div class="flex flex-col">
            <span class="font-medium">{{ $t("common.reject") }}</span>
            <span class="text-control-light text-xs">
              {{ $t("issue.review.reject-description") }}
            </span>
          </div>
        </div>
      </NRadio>
    </NRadioGroup>

    <NAlert
      v-if="selectedAction === 'APPROVE' && planCheckWarnings.length > 0"
      type="warning"
      size="small"
    >
      <ul class="text-sm">
        <li
          v-for="(warning, index) in planCheckWarnings"
          :key="index"
          class="list-disc list-inside"
        >
          {{ warning }}
        </li>
      </ul>
      <NCheckbox
        v-model:checked="performActionAnyway"
        class="mt-2"
        size="small"
      >
        {{
          $t("issue.action-anyway", {
            action: $t("common.approve"),
          })
        }}
      </NCheckbox>
    </NAlert>

    <div class="flex justify-end gap-x-2">
      <NButton quaternary @click="$emit('cancel')">
        {{ $t("common.cancel") }}
      </NButton>
      <NTooltip :disabled="confirmErrors.length === 0" placement="top">
        <template #trigger>
          <NButton
            type="primary"
            :disabled="confirmErrors.length > 0 || loading"
            :loading="loading"
            @click="
              $emit('submit', {
                action: selectedAction,
                comment,
              })
            "
          >
            {{ $t("common.submit") }}
          </NButton>
        </template>
        <template #default>
          <ErrorList :errors="confirmErrors" />
        </template>
      </NTooltip>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CheckIcon, MessageCircleIcon, XIcon } from "lucide-vue-next";
import {
  NAlert,
  NButton,
  NCheckbox,
  NRadio,
  NRadioGroup,
  NTooltip,
} from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import MarkdownEditor from "@/components/MarkdownEditor";
import { ErrorList } from "@/components/Plan/components/common";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

export type ReviewAction = "APPROVE" | "REJECT" | "COMMENT";

const props = defineProps<{
  canApprove: boolean;
  canReject: boolean;
  loading: boolean;
  compact?: boolean;
  planCheckWarnings: string[];
  project: Project;
}>();

defineEmits<{
  (event: "submit", payload: { action: ReviewAction; comment: string }): void;
  (event: "cancel"): void;
}>();

const { t } = useI18n();

const comment = ref("");
const selectedAction = ref<ReviewAction>("COMMENT");
const performActionAnyway = ref(false);

const confirmErrors = computed(() => {
  const list: string[] = [];
  if (selectedAction.value === "COMMENT" && !comment.value.trim()) {
    list.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.x-field-is-required",
        { field: t("common.comment") }
      )
    );
  }
  if (
    selectedAction.value === "APPROVE" &&
    props.planCheckWarnings.length > 0 &&
    !performActionAnyway.value
  ) {
    list.push(...props.planCheckWarnings);
  }
  return list;
});
</script>
