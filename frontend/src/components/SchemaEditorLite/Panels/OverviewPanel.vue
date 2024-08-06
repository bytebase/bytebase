<template>
  <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-3 text-sm">
    <div
      v-if="hasTableEngineProperty(instanceEngine)"
      class="flex flex-col gap-y-1 items-start"
    >
      <dt class="font-medium text-control-light">
        {{ $t("database.engine") }}
      </dt>
      <dd class="text-semibold">
        {{ table.engine }}
      </dd>
    </div>

    <div class="flex flex-col gap-y-1 items-start">
      <dt class="font-medium text-control-light">
        {{ $t("database.row-count-estimate") }}
      </dt>
      <dd class="text-semibold">
        {{ table.rowCount }}
      </dd>
    </div>

    <div class="flex flex-col gap-y-1 items-start">
      <dt class="font-medium text-control-light">
        {{ $t("database.data-size") }}
      </dt>
      <dd class="text-semibold">
        {{ bytesToString(table.dataSize.toNumber()) }}
      </dd>
    </div>

    <div
      v-if="hasIndexSizeProperty(instanceEngine)"
      class="flex flex-col gap-y-1 items-start"
    >
      <dt class="font-medium text-control-light">
        {{ $t("database.index-size") }}
      </dt>
      <dd class="text-semibold">
        {{ bytesToString(table.indexSize.toNumber()) }}
      </dd>
    </div>

    <div
      v-if="hasCollationProperty(instanceEngine)"
      class="flex flex-col gap-y-1 items-start"
    >
      <dt class="font-medium text-control-light">
        {{ $t("db.collation") }}
      </dt>
      <dd class="text-semibold">
        {{ table.collation }}
      </dd>
    </div>
  </dl>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import {
  bytesToString,
  hasCollationProperty,
  hasIndexSizeProperty,
  hasTableEngineProperty,
} from "@/utils";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}>();

const instanceEngine = computed(() => {
  return props.db.instanceResource.engine;
});
</script>
