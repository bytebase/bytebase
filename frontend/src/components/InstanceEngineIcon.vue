<template>
  <NTooltip :disabled="!instance.engineVersion">
    <template #trigger>
      <div class="relative w-4" v-bind="$attrs">
        <img class="h-4 w-auto mx-auto" :src="SelectedEngineIconPath" />
        <div
          v-if="showStatus"
          class="bg-green-400 border-surface-high rounded-full absolute border-2"
          style="bottom: -3px; height: 9px; right: -3px; width: 9px"
        />
      </div>
    </template>
    <span>{{ instance.engineVersion }}</span>
  </NTooltip>
</template>

<script lang="ts">
import { computed, PropType, defineComponent } from "vue";
import { NTooltip } from "naive-ui";

import { Instance } from "../types";

export default defineComponent({
  name: "InstanceEngineIcon",
  components: { NTooltip },
  props: {
    instance: {
      required: true,
      type: Object as PropType<Instance>,
    },
    showStatus: {
      type: Boolean,
      default: false,
    },
  },
  setup(props) {
    const EngineIconPath = {
      MYSQL: new URL("../assets/db-mysql.png", import.meta.url).href,
      POSTGRES: new URL("../assets/db-postgres.png", import.meta.url).href,
      TIDB: new URL("../assets/db-tidb.png", import.meta.url).href,
      SNOWFLAKE: new URL("../assets/db-snowflake.png", import.meta.url).href,
      CLICKHOUSE: new URL("../assets/db-clickhouse.png", import.meta.url).href,
      MONGODB: new URL("../assets/db-mongodb.png", import.meta.url).href,
      SPANNER: new URL("../assets/db-spanner.png", import.meta.url).href,
      REDIS: new URL("../assets/db-redis.png", import.meta.url).href,
      ORACLE: new URL("../assets/db-oracle.svg", import.meta.url).href,
      MSSQL: new URL("../assets/db-mssql.svg", import.meta.url).href,
      REDSHIFT: new URL("../assets/db-redshift.svg", import.meta.url).href,
      MARIADB: new URL("../assets/db-mariadb.png", import.meta.url).href,
      OCEANBASE: new URL("../assets/db-oceanbase.png", import.meta.url).href,
    };
    const SelectedEngineIconPath = computed(() => {
      return EngineIconPath[props.instance.engine];
    });
    return { SelectedEngineIconPath };
  },
});
</script>
