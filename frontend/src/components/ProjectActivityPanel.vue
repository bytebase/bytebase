<template>
  <div>
    <div class="space-y-2">
      <p class="text-lg font-medium leading-7 text-main">Activities</p>
      <ActivityTable :activity-list="state.activityList" />
    </div>
  </div>
</template>

<script lang="ts">
import { PropType, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import { Activity, Project } from "../types";
import ActivityTable from "../components/ActivityTable.vue";

interface LocalState {
  activityList: Activity[];
}

export default {
  name: "ProjectActivityPanel",
  components: { ActivityTable },
  props: {
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  setup(props) {
    const store = useStore();

    const state = reactive<LocalState>({
      activityList: [],
    });

    const prepareActivityList = () => {
      store
        .dispatch("activity/fetchActivityListForProject", {
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
};
</script>
