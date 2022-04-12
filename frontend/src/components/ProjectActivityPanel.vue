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
import { defineComponent, PropType, reactive, watchEffect } from "vue";
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

    const prepareActivityList = () => {
      useActivityStore()
        .fetchActivityListForProject({
          projectId: props.project.id,
        })
        .then((list) => {
          state.activityList = list;
        });
    };

    watchEffect(prepareActivityList);

    return {
      state,
    };
  },
});
</script>
