<template>
  <div>
    <div class="space-y-2">
      <PagedActivityTable
        :activity-find="{
          resource: project.name,
          order: 'desc',
        }"
        session-key="bb.page-activity-table.project-activity-panel"
        :page-size="10"
      >
        <template #table="{ list }">
          <ActivityTable :activity-list="list" />
        </template>
      </PagedActivityTable>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { Project } from "@/types/proto/v1/project_service";

defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
});
</script>
