<template>
  <div class="tooltip-wrapper">
    <span class="tooltip">{{ instance.engineVersion }}</span>
    <img
      v-if="instance.engine == 'CLICKHOUSE'"
      class="h-4 w-auto"
      src="../assets/db-clickhouse.png"
    />
    <img
      v-else-if="instance.engine == 'MYSQL'"
      class="h-4 w-auto"
      src="../assets/db-mysql.png"
    />
    <img
      v-else-if="instance.engine == 'POSTGRES'"
      class="h-4 w-auto"
      src="../assets/db-postgres.png"
    />
    <img
      v-else-if="instance.engine == 'SNOWFLAKE'"
      class="h-4 w-auto"
      src="../assets/db-snowflake.png"
    />
    <img
      v-else-if="instance.engine == 'TIDB'"
      class="h-4 w-auto"
      src="../assets/db-tidb.png"
    />
  </div>
</template>

<script lang="ts">
import { PropType } from "vue";
import { Instance } from "../types";

export default {
  name: "InstanceEngineIcon",
  props: {
    instance: {
      required: true,
      type: Object as PropType<Instance>,
    },
  },
};
</script>
