<template>
  <div v-if="!!schedule" class="bb-deployment-config divide-y py-2">
    <DeploymentStage
      v-for="(deployment, i) in schedule.deployments"
      :key="i"
      :index="i"
      :max="schedule.deployments.length"
      :deployment="deployment"
      :allow-edit="allowEdit"
      :show-header="true"
      :database-list="databaseList"
      :label-list="labelList"
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
import {
  AvailableLabel,
  Database,
  Deployment,
  DeploymentSchedule,
} from "../../types";
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
      type: Object as PropType<DeploymentSchedule>,
      required: true,
    },
    databaseList: {
      type: Array as PropType<Database[]>,
      default: () => [],
    },
    labelList: {
      type: Array as PropType<AvailableLabel[]>,
      default: () => [],
    },
  },
  setup(props) {
    const removeStage = (deployment: Deployment) => {
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
    };
  },
});
</script>
