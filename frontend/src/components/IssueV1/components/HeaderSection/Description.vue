<template>
  <div class="flex items-center gap-x-4 py-1">
    <div v-if="showProjectInfo" class="flex items-center gap-x-1">
      <div class="textlabel">{{ $t("common.project") }}</div>
      <div>-</div>
      <ProjectV1Name :project="project" />
    </div>

    <i18n-t
      v-if="!isCreating && creator"
      keypath="issue.opened-by-at"
      tag="div"
      class="text-control-light"
    >
      <template #creator>
        <router-link
          :to="`/users/${creator.email}`"
          class="font-medium text-control hover:underline"
          >{{ creator.title }}</router-link
        >
      </template>
      <template #time>
        <HumanizeDate :date="issue.createTime" />
      </template>
    </i18n-t>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import HumanizeDate from "@/components/misc/HumanizeDate.vue";
import { usePageMode, useUserStore } from "@/store";
import { extractUserResourceName } from "@/utils";
import { useIssueContext } from "../../logic";

const pageMode = usePageMode();
const { isCreating, issue } = useIssueContext();

const creator = computed(() => {
  const email = extractUserResourceName(issue.value.creator);
  return useUserStore().getUserByEmail(email);
});

const project = computed(() => issue.value.projectEntity);

const showProjectInfo = computed(() => pageMode.value === "BUNDLED");
</script>
