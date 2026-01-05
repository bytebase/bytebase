<template>
  <div class="px-4 py-2 flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <span class="textlabel">
        {{ title }}
      </span>

      <div
        v-if="!isCreating && allowEditIssue"
        class="flex items-center gap-x-2"
      >
        <NButton v-if="!state.isEditing" size="tiny" @click.prevent="beginEdit">
          {{ $t("common.edit") }}
        </NButton>
        <NButton
          v-if="state.isEditing"
          size="tiny"
          :disabled="state.description === issue.description"
          :loading="state.isUpdating"
          @click.prevent="saveEdit"
        >
          {{ $t("common.save") }}
        </NButton>
        <NButton
          v-if="state.isEditing"
          size="tiny"
          quaternary
          @click.prevent="cancelEdit"
        >
          {{ $t("common.cancel") }}
        </NButton>
      </div>
    </div>

    <div class="">
      <NInput
        v-if="isCreating || state.isEditing"
        ref="inputRef"
        v-model:value="state.description"
        :placeholder="$t('issue.add-some-description')"
        :autosize="{ minRows: 3, maxRows: 10 }"
        :disabled="state.isUpdating"
        :loading="state.isUpdating"
        :maxlength="10000"
        style="
          width: 100%;
          --n-placeholder-color: rgb(var(--color-control-placeholder));
        "
        type="textarea"
        size="small"
        @update:value="onDescriptionChange"
      />
      <div
        v-else
        class="min-h-12 max-h-64 whitespace-pre-wrap px-4 py-3 text-sm bg-white rounded-lg border border-gray-200 overflow-y-auto"
      >
        <template v-if="issue.description">
          <iframe
            v-if="issue.description"
            ref="contentPreviewArea"
            :srcdoc="renderedContent"
            class="rounded-md w-full overflow-hidden"
          />
        </template>
        <span v-else class="text-gray-400">
          {{ $t("issue.add-some-description") }}
        </span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton, NInput } from "naive-ui";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRenderMarkdown } from "@/components/MarkdownEditor";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import {
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { isGrantRequestIssue } from "@/utils";
import { useIssueContext } from "../../logic";

type LocalState = {
  isEditing: boolean;
  isUpdating: boolean;
  description: string;
};

const { t } = useI18n();
const {
  isCreating,
  issue,
  allowChange: allowEditIssue,
  events,
} = useIssueContext();
const { project } = useCurrentProjectV1();
const contentPreviewArea = ref<HTMLIFrameElement>();

const state = reactive<LocalState>({
  isEditing: false,
  isUpdating: false,
  description: issue.value.description,
});

const inputRef = ref<InstanceType<typeof NInput>>();

const title = computed(() => {
  return isGrantRequestIssue(issue.value)
    ? t("common.reason")
    : t("common.description");
});

const onDescriptionChange = (description: string) => {
  if (isCreating.value) {
    issue.value.description = description;
  }
};

const beginEdit = () => {
  state.description = issue.value.description;
  state.isEditing = true;
  nextTick(() => {
    inputRef.value?.focus();
  });
};

const saveEdit = async () => {
  try {
    state.isUpdating = true;
    if (issue.value.plan) {
      const request = create(UpdatePlanRequestSchema, {
        plan: create(PlanSchema, {
          name: issue.value.plan,
          description: state.description,
        }),
        updateMask: { paths: ["description"] },
      });
      await planServiceClientConnect.updatePlan(request);
    } else {
      const request = create(UpdateIssueRequestSchema, {
        issue: create(IssueSchema, {
          name: issue.value.name,
          description: state.description,
        }),
        updateMask: { paths: ["description"] },
      });
      await issueServiceClientConnect.updateIssue(request);
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    events.emit("status-changed", { eager: true });
    state.isEditing = false;
  } finally {
    state.isUpdating = false;
  }
};

const cancelEdit = () => {
  state.description = issue.value.description;
  state.isEditing = false;
};

const { renderedContent } = useRenderMarkdown(
  computed(() => issue.value.description),
  contentPreviewArea,
  project
);

// Reset the edit state after creating the issue.
watch(isCreating, (curr, prev) => {
  if (!curr && prev) {
    state.isEditing = false;
  }
});

watch(
  () => issue.value,
  (issue) => {
    if (state.isEditing) return;
    state.description = issue.description;
  },
  { immediate: true }
);
</script>
