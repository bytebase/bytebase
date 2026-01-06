<template>
  <div class="flex-1">
    <NInput
      v-model:value="state.title"
      :style="style"
      :loading="state.isUpdating"
      :disabled="!allowEdit || state.isUpdating"
      :maxlength="200"
      size="medium"
      required
      class="bb-plan-title-input"
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
import {
  extractUserId,
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import {
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContext } from "../../logic";

type ViewMode = "EDIT" | "VIEW";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const {
  isCreating,
  plan,
  issue,
  readonly,
  allowEdit: hasPermission,
} = usePlanContext();

const state = reactive({
  isEditing: false,
  isUpdating: false,
  title: "",
});

// Watch for changes in issue/plan to update the title
watch(
  () => [plan.value, issue.value],
  () => {
    state.title = issue.value ? issue.value.title : plan.value.title;
  },
  { immediate: true }
);

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

const allowEdit = computed(() => {
  if (readonly.value) {
    return false;
  }
  if (isCreating.value) {
    return true;
  }
  // Plans with rollout should have readonly title
  if (!issue.value && plan.value.hasRollout) {
    return false;
  }

  // If issue exists, check issue permissions
  if (issue.value) {
    // Allowed if current user is the creator.
    if (extractUserId(issue.value.creator) === currentUser.value.email) {
      return true;
    }
    // Allowed if current user has related permission.
    if (hasProjectPermissionV2(project.value, "bb.issues.update")) {
      return true;
    }
    return false;
  }

  return hasPermission.value;
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

  // If issue exists, update issue title
  if (issue.value) {
    if (state.title === issue.value.title) {
      cleanup();
      return;
    }
    try {
      state.isUpdating = true;
      const issuePatch = create(IssueSchema, {
        ...issue.value,
        title: state.title,
      });
      const request = create(UpdateIssueRequestSchema, {
        issue: issuePatch,
        updateMask: { paths: ["title"] },
      });
      const response = await issueServiceClientConnect.updateIssue(request);
      Object.assign(issue.value, response);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      console.error("Failed to update issue title:", error);
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.error"),
        description: String(error),
      });
      // Revert the title - use optional chaining in case issue becomes undefined
      state.title = issue.value?.title || "";
    } finally {
      cleanup();
    }
    return;
  }

  // Update plan title
  if (state.title === plan.value.title) {
    cleanup();
    return;
  }
  try {
    state.isUpdating = true;
    const planPatch = create(PlanSchema, {
      ...plan.value,
      title: state.title,
    });
    const request = create(UpdatePlanRequestSchema, {
      plan: planPatch,
      updateMask: { paths: ["title"] },
    });
    const response = await planServiceClientConnect.updatePlan(request);
    Object.assign(plan.value, response);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } catch (error) {
    console.error("Failed to update plan title:", error);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.error"),
      description: String(error),
    });
    // Revert the title
    state.title = plan.value.title;
  } finally {
    cleanup();
  }
};

const onEnter = (e: Event) => {
  const input = e.target as HTMLInputElement;
  input.blur();
};

const onUpdateValue = (value: string) => {
  if (!isCreating.value) {
    return;
  }
  // When creating, update issue title if issue exists, plan title otherwise
  if (issue.value) {
    issue.value.title = value;
  } else {
    plan.value.title = value;
  }
};
</script>

<style>
.bb-plan-title-input input {
  cursor: text !important;
  color: var(--n-text-color) !important;
  text-decoration-color: var(--n-text-color) !important;
}
</style>
