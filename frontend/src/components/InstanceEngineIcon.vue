<template>
  <div class="tooltip-wrapper">
    <span class="tooltip">{{ instance.engineVersion }}</span>
    <img class="h-4 w-auto" :src="SelectedEngineIconPath" />
  </div>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { Instance } from "../types";

export default {
  name: "InstanceEngineIcon",
  props: {
    instance: {
      required: true,
      type: Object as PropType<Instance>,
    },
  },
  setup(props) {
    const EngineIconPath = {
      MYSQL: new URL("../assets/db-mysql.png", import.meta.url).href,
      POSTGRES: new URL("../assets/db-postgres.png", import.meta.url).href,
      TIDB: new URL("../assets/db-tidb.png", import.meta.url).href,
      SNOWFLAKE: new URL("../assets/db-snowflake.png", import.meta.url).href,
      CLICKHOUSE: new URL("../assets/db-clickhouse.png", import.meta.url).href,
    };
    const SelectedEngineIconPath = computed(() => {
      return EngineIconPath[props.instance.engine];
    });
    return { SelectedEngineIconPath };
  },
};
</script>
