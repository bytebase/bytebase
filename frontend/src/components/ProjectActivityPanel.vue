<template>
  <div>
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("common.activity") }}
      </p>
      <PagedActivityTableVue
        :activity-find="{
          typePrefix: ['bb.project.', 'bb.database.'],
          container: project.id,
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

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { Project } from "../types";
import ActivityTable from "../components/ActivityTable.vue";
import PagedActivityTableVue from "@/components/PagedActivityTable.vue";

export default defineComponent({
  name: "ProjectActivityPanel",
  components: { ActivityTable, PagedActivityTableVue },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
});
</script>
