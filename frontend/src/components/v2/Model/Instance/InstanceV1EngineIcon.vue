<template>
  <NTooltip v-if="icon">
    <template #trigger>
      <div class="relative w-4" v-bind="$attrs">
        <img class="h-4 w-auto mx-auto" :src="icon" />
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

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { NTooltip } from "naive-ui";

import { Engine } from "@/types/proto/v1/common";
import { Instance } from "@/types/proto/v1/instance_service";

const props = defineProps({
  instance: {
    required: true,
    type: Object as PropType<Instance>,
  },
  showStatus: {
    type: Boolean,
    default: false,
  },
});

const ICON_PATH_MAP = new Map([
  [Engine.MYSQL, new URL("@/assets/db-mysql.png", import.meta.url).href],
  [Engine.POSTGRES, new URL("@/assets/db-postgres.png", import.meta.url).href],
  [Engine.TIDB, new URL("@/assets/db-tidb.png", import.meta.url).href],
  [
    Engine.SNOWFLAKE,
    new URL("@/assets/db-snowflake.png", import.meta.url).href,
  ],
  [
    Engine.CLICKHOUSE,
    new URL("@/assets/db-clickhouse.png", import.meta.url).href,
  ],
  [Engine.MONGODB, new URL("@/assets/db-mongodb.png", import.meta.url).href],
  [Engine.SPANNER, new URL("@/assets/db-spanner.png", import.meta.url).href],
  [Engine.REDIS, new URL("@/assets/db-redis.png", import.meta.url).href],
  [Engine.ORACLE, new URL("@/assets/db-oracle.svg", import.meta.url).href],
  [Engine.MSSQL, new URL("@/assets/db-mssql.svg", import.meta.url).href],
  [Engine.REDSHIFT, new URL("@/assets/db-redshift.svg", import.meta.url).href],
  [Engine.MARIADB, new URL("@/assets/db-mariadb.png", import.meta.url).href],
  [
    Engine.OCEANBASE,
    new URL("@/assets/db-oceanbase.png", import.meta.url).href,
  ],
]);
const icon = computed(() => {
  return ICON_PATH_MAP.get(props.instance.engine);
});
</script>
