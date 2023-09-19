<template>
  <NTooltip trigger="hover" :delay="500" :animated="false">
    <template #trigger>
      <NButton
        quaternary
        size="tiny"
        class="!px-1"
        v-bind="$attrs"
        @click="state.show = true"
      >
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
import { NButton } from "naive-ui";
import { reactive } from "vue";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";
import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto/v1/database_service";

type LocalState = {
  show: boolean;
};

defineProps<{
  database: ComposedDatabase;
  databaseMetadata: DatabaseMetadata;
}>();

const state = reactive<LocalState>({
  show: false,
});
</script>
