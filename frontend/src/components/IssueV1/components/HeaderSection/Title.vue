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
import { issueServiceClientConnect } from "@/grpcweb";
import { emitWindowEvent } from "@/plugins";
import { pushNotification } from "@/store";
import {
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { useIssueContext } from "../../logic";

type ViewMode = "EDIT" | "VIEW";

const { t } = useI18n();
const { isCreating, issue, allowChange: allowEditIssue } = useIssueContext();

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
    const request = create(UpdateIssueRequestSchema, {
      issue: create(IssueSchema, {
        name: issue.value.name,
        title: state.title,
      }),
      updateMask: { paths: ["title"] },
    });
    const updated = await issueServiceClientConnect.updateIssue(request);
    Object.assign(issue.value, updated);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    emitWindowEvent("bb.issue-field-update");
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
