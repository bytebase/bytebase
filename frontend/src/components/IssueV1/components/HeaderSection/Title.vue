<template>
  <div class="flex-1">
    <NInput
      :value="state.title"
      :style="style"
      :loading="state.isUpdating"
      :disabled="!allowEditIssue || state.isUpdating"
      size="medium"
      required
      class="bb-issue-title-input"
      @focus="state.isEditing = true"
      @blur="onBlur"
      @keyup.enter="onEnter"
      @update:value="onUpdateValue"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NInput } from "naive-ui";
import type { CSSProperties } from "vue";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import { pushNotification } from "@/store";
import {
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { useIssueContext } from "../../logic";

type ViewMode = "EDIT" | "VIEW";

const { t } = useI18n();
const {
  isCreating,
  issue,
  allowChange: allowEditIssue,
  events,
} = useIssueContext();

const state = reactive({
  isEditing: false,
  isUpdating: false,
  title: issue.value.title,
});

const viewMode = computed((): ViewMode => {
  if (isCreating.value) return "EDIT";
  return state.isEditing ? "EDIT" : "VIEW";
});

const style = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    "--n-color-disabled": "transparent",
    "--n-font-size": "18px",
    "font-weight": "bold",
  };
  const border =
    viewMode.value === "EDIT"
      ? "1px solid rgb(var(--color-control-border))"
      : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

const onBlur = async () => {
  const cleanup = () => {
    state.isEditing = false;
    state.isUpdating = false;
  };

  if (isCreating.value) {
    cleanup();
    return;
  }
  if (state.title === issue.value.title) {
    cleanup();
    return;
  }
  try {
    state.isUpdating = true;
    if (issue.value.plan) {
      const request = create(UpdatePlanRequestSchema, {
        plan: create(PlanSchema, {
          name: issue.value.plan,
          title: state.title,
        }),
        updateMask: { paths: ["title"] },
      });
      await planServiceClientConnect.updatePlan(request);
    } else {
      const request = create(UpdateIssueRequestSchema, {
        issue: create(IssueSchema, {
          name: issue.value.name,
          title: state.title,
        }),
        updateMask: { paths: ["title"] },
      });
      await issueServiceClientConnect.updateIssue(request);
    }
    events.emit("status-changed", { eager: true });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    cleanup();
  }
};

const onEnter = (e: Event) => {
  const input = e.target as HTMLInputElement;
  input.blur();
};

const onUpdateValue = (title: string) => {
  state.title = title;
  if (isCreating.value) {
    issue.value.title = title;
  }
};

watch(
  () => issue.value.title,
  (title) => {
    state.title = title;
  }
);
</script>

<style>
.bb-issue-title-input input {
  cursor: text !important;
  color: var(--n-text-color) !important;
  text-decoration-color: var(--n-text-color) !important;
}
</style>
