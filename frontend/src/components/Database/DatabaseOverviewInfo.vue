<template>
  <dl
    class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2"
    data-label="bb-database-overview-description-list"
  >
    <template v-if="instanceV1HasCollationAndCharacterSet(databaseEngine)">
      <div class="col-span-1 col-start-1">
        <dt class="text-sm font-medium text-control-light">
          {{
            databaseEngine === Engine.POSTGRES
              ? $t("db.encoding")
              : $t("db.character-set")
          }}
        </dt>
        <dd class="mt-1 text-sm text-main">
          {{ databaseSchemaMetadata.characterSet }}
        </dd>
      </div>

      <div class="col-span-1">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("db.collation") }}
        </dt>
        <dd class="mt-1 text-sm text-main">
          {{ databaseSchemaMetadata.collation }}
        </dd>
      </div>
    </template>

    <div class="col-span-1 col-start-1">
      <dt class="text-sm font-medium text-control-light">
        {{ $t("database.sync-status") }}
      </dt>
      <dd class="mt-1 text-sm text-main">
        <span>
          {{ database.state === State.ACTIVE ? "OK" : "NOT_FOUND" }}
        </span>
      </dd>
    </div>

    <div class="col-span-1">
      <dt class="text-sm font-medium text-control-light">
        {{ $t("database.last-successful-sync") }}
      </dt>
      <dd class="mt-1 text-sm text-main">
        {{
          humanizeDate(
            database.successfulSyncTime
              ? new Date(Number(database.successfulSyncTime.seconds) * 1000)
              : undefined
          )
        }}
      </dd>
    </div>
  </dl>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import { humanizeDate, instanceV1HasCollationAndCharacterSet } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const databaseEngine = computed(() => {
  return props.database.instanceResource.engine;
});

const databaseSchemaMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(props.database.name);
});
</script>
