<template>
  <div>
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("common.activity") }}
      </p>
      <PagedActivityTableVue
        :activity-find="{
          typePrefix: ['bb.project.', 'bb.database.'],
          container: project.uid,
          order: 'DESC',
        }"
        session-key="project-activity-panel"
        :page-size="10"
      >
        <template #table="{ activityList }">
          <ActivityTable :activity-list="activityList" />
        </template>
      </PagedActivityTableVue>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import ActivityTable from "../components/ActivityTable.vue";
import PagedActivityTableVue from "@/components/PagedActivityTable.vue";
import { Project } from "@/types/proto/v1/project_service";

defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});
</script>
