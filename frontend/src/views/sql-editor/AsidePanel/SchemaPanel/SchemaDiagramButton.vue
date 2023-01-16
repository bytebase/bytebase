<template>
  <NTooltip trigger="hover">
    <template #trigger>
      <NButton text v-bind="$attrs" @click="state.show = true">
        <SchemaDiagramIcon />
      </NButton>
    </template>
    {{ $t("schema-diagram.self") }}
  </NTooltip>

  <BBModal
    v-if="state.show"
    :title="$t('schema-diagram.self')"
    class="h-[calc(100vh-40px)] !max-h-[calc(100vh-40px)]"
    header-class="!border-0"
    container-class="flex-1 !pt-0"
    @close="state.show = false"
  >
    <div class="w-[80vw] h-full">
      <SchemaDiagram
        :database="database"
        :database-metadata="databaseMetadata"
      />
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive } from "vue";

import type { DatabaseMetadata } from "@/types/proto/store/database";
import type { Database } from "@/types";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";

type LocalState = {
  show: boolean;
};

defineProps<{
  database: Database;
  databaseMetadata: DatabaseMetadata;
}>();

const state = reactive<LocalState>({
  show: false,
});
</script>
