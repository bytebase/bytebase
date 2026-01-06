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
          <div class="rounded-lg border border-gray-200 bg-white overflow-hidden">
            <div class="px-3 py-2 bg-gray-50">
              <div class="flex items-center justify-between">
                <div
                  class="flex items-center gap-x-2 text-sm min-w-0 flex-wrap"
                >
                  <ActionCreator :creator="creatorName" />
                  <span class="text-gray-600 wrap-break-word min-w-0">{{
                    $t("activity.sentence.created-issue")
                  }}</span>
                  <HumanizeTs
                    :ts="getTimeForPbTimestampProtoEs(createTime, 0) / 1000"
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
              <EditableMarkdownContent
                :content="description"
                :edit-content="state.editContent"
                :project="project"
                :is-editing="state.isEditing"
                :allow-save="allowSave"
                :is-saving="state.isSaving"
                :placeholder="$t('issue.no-description-provided')"
                @update:edit-content="state.editContent = $event"
                @save="saveEdit"
                @cancel="cancelEdit"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </li>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { PencilIcon, PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import HumanizeTs from "@/components/misc/HumanizeTs.vue";
import { usePlanContext } from "@/components/Plan/logic";
import UserAvatar from "@/components/User/UserAvatar.vue";
import { issueServiceClientConnect } from "@/connect";
import { useCurrentProjectV1, useCurrentUserV1, useUserStore } from "@/store";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import ActionCreator from "./ActionCreator.vue";
import EditableMarkdownContent from "./EditableMarkdownContent.vue";

defineProps<{
  issueComments: IssueComment[];
}>();

const { issue } = usePlanContext();
const currentUser = useCurrentUserV1();
const userStore = useUserStore();
const { project } = useCurrentProjectV1();

const state = reactive({
  isEditing: false,
  editContent: "",
  isSaving: false,
});

const creatorName = computed(() => {
  return issue.value?.creator || "";
});

const createTime = computed((): Timestamp | undefined => {
  return issue.value?.createTime;
});

const creator = computed(() => {
  return (
    userStore.getUserByIdentifier(creatorName.value) ??
    unknownUser(creatorName.value)
  );
});

const description = computed(() => {
  return issue.value?.description || "";
});

const allowEdit = computed(() => {
  if (currentUser.value.name === creator.value?.name) {
    return true;
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
  if (!allowSave.value || !issue.value) return;

  try {
    state.isSaving = true;

    const issuePatch = create(IssueSchema, {
      name: issue.value.name,
      description: state.editContent,
    });
    const request = create(UpdateIssueRequestSchema, {
      issue: issuePatch,
      updateMask: { paths: ["description"] },
    });
    const response = await issueServiceClientConnect.updateIssue(request);
    Object.assign(issue.value, response);

    cancelEdit();
  } catch (error) {
    console.error("Failed to update description:", error);
  } finally {
    state.isSaving = false;
  }
};
</script>
