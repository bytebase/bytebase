<template>
  <li>
    <div class="relative pb-4">
      <span
        v-if="issueComments.length > 0"
        class="absolute left-4 -ml-px h-full w-0.5 bg-block-border"
        aria-hidden="true"
      ></span>
      <div class="relative flex items-start">
        <div class="relative">
          <div class="pt-1.5 bg-white"></div>
          <UserAvatar override-class="w-7 h-7 font-medium" :user="creator" />
          <div
            class="absolute -bottom-1 -right-1 w-4 h-4 bg-control-bg rounded-full ring-2 ring-white flex items-center justify-center"
          >
            <PlusIcon class="w-4 h-4 text-control" />
          </div>
        </div>

        <div class="ml-3 min-w-0 flex-1">
          <div class="rounded-lg border border-gray-200 bg-white">
            <div class="px-3 py-2 bg-gray-50">
              <div class="flex items-center justify-between">
                <div
                  class="flex items-center gap-x-2 text-sm min-w-0 flex-wrap"
                >
                  <ActionCreator :creator="issue.creator" />
                  <span class="text-gray-600 wrap-break-word min-w-0">{{
                    $t("activity.sentence.created-issue")
                  }}</span>
                  <HumanizeTs
                    :ts="
                      getTimeForPbTimestampProtoEs(issue.createTime, 0) / 1000
                    "
                    class="text-gray-500"
                  />
                </div>
                <NButton
                  v-if="allowEdit && !state.isEditing"
                  quaternary
                  size="tiny"
                  @click.prevent="startEdit"
                >
                  <PencilIcon class="w-3.5 h-3.5" />
                </NButton>
              </div>
            </div>
            <div
              class="px-4 py-3 border-t border-gray-200 text-sm text-gray-700"
            >
              <p v-if="!state.isEditing && !description">
                <i class="text-gray-400 italic">{{
                  $t("issue.no-description-provided")
                }}</i>
              </p>
              <MarkdownEditor
                v-else
                :mode="state.isEditing ? 'editor' : 'preview'"
                :content="state.isEditing ? state.editContent : description"
                :project="project"
                @change="(val: string) => (state.editContent = val)"
                @submit="saveEdit"
              />
              <div
                v-if="state.isEditing"
                class="flex gap-x-2 mt-2 items-center justify-end"
              >
                <NButton quaternary size="small" @click.prevent="cancelEdit">
                  {{ $t("common.cancel") }}
                </NButton>
                <NButton
                  size="small"
                  :disabled="!allowSave"
                  :loading="state.isSaving"
                  @click.prevent="saveEdit"
                >
                  {{ $t("common.save") }}
                </NButton>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </li>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { PencilIcon, PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import MarkdownEditor from "@/components/MarkdownEditor";
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import { useCurrentProjectV1, useCurrentUserV1, useUserStore } from "@/store";
import { type ComposedIssue, getTimeForPbTimestampProtoEs } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { UpdateIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import ActionCreator from "./ActionCreator.vue";

const props = defineProps<{
  issue: ComposedIssue;
  issueComments: IssueComment[];
}>();

const emit = defineEmits<{
  (event: "update-issue", issue: ComposedIssue): void;
}>();

const currentUser = useCurrentUserV1();
const userStore = useUserStore();
const { project } = useCurrentProjectV1();

const state = reactive({
  isEditing: false,
  editContent: "",
  isSaving: false,
});

const creator = computed(() => {
  return userStore.getUserByIdentifier(props.issue.creator);
});

const description = computed(() => {
  // Try to get description from plan first, then fallback to issue description
  return (
    (props.issue.plan
      ? props.issue.planEntity?.description
      : props.issue.description) || ""
  );
});

const allowEdit = computed(() => {
  // Always allow if the current user is the creator.
  if (currentUser.value.name === creator.value?.name) {
    return true;
  }
  // Check for plan update permission if we have a plan, otherwise issue update.
  if (props.issue.plan) {
    return hasProjectPermissionV2(project.value, "bb.plans.update");
  }
  return hasProjectPermissionV2(project.value, "bb.issues.update");
});

const allowSave = computed(() => {
  return state.editContent !== description.value;
});

const startEdit = () => {
  state.editContent = description.value;
  state.isEditing = true;
};

const cancelEdit = () => {
  state.isEditing = false;
  state.editContent = "";
};

const saveEdit = async () => {
  if (!allowSave.value) return;

  try {
    state.isSaving = true;

    if (props.issue.plan) {
      // Update plan description
      const planPatch = create(PlanSchema, {
        name: props.issue.plan,
        description: state.editContent,
      });
      const request = create(UpdatePlanRequestSchema, {
        plan: planPatch,
        updateMask: { paths: ["description"] },
      });
      const response = await planServiceClientConnect.updatePlan(request);
      if (props.issue.planEntity) {
        const updatedIssue = {
          ...props.issue,
          planEntity: response,
        };
        emit("update-issue", updatedIssue);
      }
    } else {
      // Update issue description
      const request = create(UpdateIssueRequestSchema, {
        issue: {
          ...props.issue,
          description: state.editContent,
        },
        updateMask: { paths: ["description"] },
      });
      const response = await issueServiceClientConnect.updateIssue(request);
      const updatedIssue = {
        ...props.issue,
        ...response,
      };
      emit("update-issue", updatedIssue);
    }

    cancelEdit();
  } catch (error) {
    console.error("Failed to update description:", error);
  } finally {
    state.isSaving = false;
  }
};
</script>
