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

      <template v-if="!create">
        <IssueReviewSidebarSection />
      </template>
    </div>

    <div
      class="mt-6 border-t border-block-border pt-6 grid gap-y-6 gap-x-1 grid-cols-3"
    >
      <h2 class="textlabel flex items-center col-span-1 col-start-1">
        {{ $t("common.project") }}
      </h2>
      <ProjectV1Name
        :project="project"
        :link="true"
        :plain="true"
        class="col-span-2 text-sm font-medium text-main hover:underline"
      />

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
  </aside>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
import { computed } from "vue";
import { useProjectV1Store } from "@/store";
import { Issue, IssueCreate } from "@/types";
import PrincipalAvatar from "../PrincipalAvatar.vue";
import { ProjectV1Name } from "../v2";
import IssueStatusIcon from "./IssueStatusIcon.vue";
import IssueSubscriberPanel from "./IssueSubscriberPanel.vue";
import { useExtraIssueLogic, useIssueLogic } from "./logic";
import { IssueReviewSidebarSection } from "./review";

dayjs.extend(isSameOrAfter);

const projectV1Store = useProjectV1Store();

const { create, issue } = useIssueLogic();
const { addSubscriberId, removeSubscriberId } = useExtraIssueLogic();

const project = computed(() => {
  const projectId = create.value
    ? (issue.value as IssueCreate).projectId
    : (issue.value as Issue).project.id;
  return projectV1Store.getProjectByUID(String(projectId));
});
</script>
