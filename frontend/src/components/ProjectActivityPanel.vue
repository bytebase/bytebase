<template>
  <div>
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">
        {{ $t("common.activity") }}
      </p>
      <ActivityTable :activity-list="state.activityList" />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, onBeforeMount, PropType, reactive } from "vue";
import { Activity, Project } from "../types";
import ActivityTable from "../components/ActivityTable.vue";
import { useActivityStore } from "@/store";

interface LocalState {
  activityList: Activity[];
}

export default defineComponent({
  name: "ProjectActivityPanel",
  components: { ActivityTable },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  setup(props) {
    const state = reactive<LocalState>({
      activityList: [],
    });
    const activityStore = useActivityStore();

    const prepareActivityList = () => {
      const requests = [
        activityStore.fetchActivityListForDatabaseByProjectId({
          projectId: props.project.id,
        }),
        activityStore.fetchActivityListForProject({
          projectId: props.project.id,
        }),
      ];

      Promise.all(requests).then((lists) => {
        const flattenList = lists.flatMap((list) => list);
        flattenList.sort((a, b) => -(a.createdTs - b.createdTs)); // by createdTs DESC
        state.activityList = flattenList;
      });
    };

    onBeforeMount(prepareActivityList);

    return {
      state,
    };
  },
});
</script>
