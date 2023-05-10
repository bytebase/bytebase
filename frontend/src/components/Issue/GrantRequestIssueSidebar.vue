<template>
  <aside class="pr-0.5">
    <h2 class="sr-only">Details</h2>
    <div class="grid gap-y-6 gap-x-1 grid-cols-3">
      <template v-if="!create">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.status") }}
        </h2>
        <div class="col-span-2">
          <span class="flex items-center space-x-2">
            <IssueStatusIcon
              :issue-status="(issue as Issue).status"
              :size="'normal'"
            />
            <span class="text-main capitalize">
              {{ (issue as Issue).status.toLowerCase() }}
            </span>
          </span>
        </div>
      </template>

      <h2 class="textlabel flex items-center col-span-1 col-start-1 gap-x-1">
        <span>{{ $t("common.assignee") }}</span>
        <span>
          <NTooltip>
            <template #trigger>
              <heroicons-outline:question-mark-circle />
            </template>
            <div>{{ $t("issue.assignee-tooltip") }}</div>
          </NTooltip>
        </span>
      </h2>

      <div class="col-span-2" data-label="bb-assignee-select-container">
        <MemberSelect
          class="w-full"
          :disabled="true"
          :selected-id="assigneeId as number"
          data-label="bb-assignee-select"
          @select-principal-id="
            (principalId: number) => {
              updateAssigneeId(principalId)
            }
          "
        />
      </div>
    </div>

    <div
      class="mt-6 border-t border-block-border pt-6 grid gap-y-6 gap-x-1 grid-cols-3"
    >
      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        {{ $t("common.project") }}
      </h2>
      <router-link
        :to="`/project/${projectSlug(project)}`"
        class="col-span-2 text-sm font-medium text-main hover:underline"
      >
        {{ projectName(project) }}
      </router-link>

      <template v-if="!create">
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.updated-at") }}
        </h2>
        <span class="textfield col-span-2">
          {{ dayjs((issue as Issue).updatedTs * 1000).format("LLL") }}</span
        >

        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.created-at") }}
        </h2>
        <span class="textfield col-span-2">
          {{ dayjs((issue as Issue).createdTs * 1000).format("LLL") }}</span
        >
        <h2 class="textlabel flex items-center col-span-1 col-start-1">
          {{ $t("common.creator") }}
        </h2>
        <ul class="col-span-2">
          <li class="flex justify-start items-center space-x-2">
            <div class="flex-shrink-0">
              <PrincipalAvatar
                :principal="(issue as Issue).creator"
                :size="'SMALL'"
              />
            </div>
            <router-link
              :to="`/u/${(issue as Issue).creator.id}`"
              class="text-sm font-medium text-main hover:underline"
            >
              {{ (issue as Issue).creator.name }}
            </router-link>
          </li>
        </ul>
      </template>
    </div>
    <IssueSubscriberPanel
      v-if="!create"
      :issue="(issue as Issue)"
      @add-subscriber-id="(subscriberId) => addSubscriberId(subscriberId)"
      @remove-subscriber-id="(subscriberId) => removeSubscriberId(subscriberId)"
    />
    <FeatureModal
      v-if="state.showFeatureModal"
      :feature="'bb.feature.task-schedule-time'"
      @cancel="state.showFeatureModal = false"
    />
  </aside>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
import IssueStatusIcon from "./IssueStatusIcon.vue";
import IssueSubscriberPanel from "./IssueSubscriberPanel.vue";
import PrincipalAvatar from "../PrincipalAvatar.vue";
import MemberSelect from "../MemberSelect.vue";
import FeatureModal from "../FeatureModal.vue";
import { Project, Issue, IssueCreate } from "@/types";
import { useProjectStore } from "@/store";
import { useExtraIssueLogic, useIssueLogic } from "./logic";

dayjs.extend(isSameOrAfter);

interface LocalState {
  showFeatureModal: boolean;
}

const projectStore = useProjectStore();

const { create, issue } = useIssueLogic();
const { updateAssigneeId, addSubscriberId, removeSubscriberId } =
  useExtraIssueLogic();

const state = reactive<LocalState>({
  showFeatureModal: false,
});

const project = computed((): Project => {
  if (create.value) {
    return projectStore.getProjectById((issue.value as IssueCreate).projectId);
  }
  return (issue.value as Issue).project;
});

const assigneeId = computed(() => {
  if (create.value) {
    return (issue.value as IssueCreate).assigneeId;
  }
  return (issue.value as Issue).assignee.id;
});
</script>
