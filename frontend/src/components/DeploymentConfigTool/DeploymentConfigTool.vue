<template>
  <div v-if="!!schedule" class="bb-deployment-config divide-y">
    <DeploymentStage
      v-for="(deployment, i) in schedule.deployments"
      :key="getKey(deployment)"
      :index="i"
      :max="schedule.deployments.length"
      :deployment="deployment"
      :allow-edit="allowEdit"
      :show-header="true"
      :database-list="databaseList"
      @remove="removeStage(deployment)"
      @prev="reorder(i, -1)"
      @next="reorder(i, 1)"
    >
    </DeploymentStage>
  </div>
</template>

<script lang="ts">
/* eslint-disable vue/no-mutating-props */
import { defineComponent, PropType } from "vue";
import { ComposedDatabase } from "@/types";
import { Schedule, ScheduleDeployment } from "@/types/proto/v1/project_service";
import DeploymentStage from "./DeploymentStage.vue";

export default defineComponent({
  name: "DeploymentConfigTool",
  components: { DeploymentStage },
  props: {
    allowEdit: {
      type: Boolean,
      default: false,
    },
    schedule: {
      type: Object as PropType<Schedule>,
      required: true,
    },
    databaseList: {
      type: Array as PropType<ComposedDatabase[]>,
      default: () => [],
    },
  },
  setup(props) {
    const keyMap = new WeakMap<ScheduleDeployment, number>();
    // Map each Deployment object to an unique key to keep it being "moved"
    // rather than "replaced" when re-ordering. (BYT-758)
    const getKey = (dep: ScheduleDeployment): number => {
      let key = keyMap.get(dep);
      if (!key) {
        key = Math.random();
        keyMap.set(dep, key);
      }
      return key;
    };

    const removeStage = (deployment: ScheduleDeployment) => {
      const array = props.schedule.deployments;
      const index = array.indexOf(deployment);
      if (index >= 0) {
        array.splice(index, 1);
      }
    };

    const reorder = (index: number, diff: number) => {
      const swap = (i: number, j: number) => {
        const array = props.schedule.deployments;
        const tmp = array[i];
        array[i] = array[j];
        array[j] = tmp;
      };
      const target = index + diff;
      swap(index, target);
    };

    return {
      removeStage,
      reorder,
      getKey,
    };
  },
});
</script>
