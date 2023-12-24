<template>
  <div
    class="flex items-center text-sm textlabel cursor-pointer hover:text-accent"
    v-bind="$attrs"
    @click.stop.prevent="openModal"
  >
    <span class="mr-1">{{ $t("schema-diagram.self") }}</span>
    <SchemaDiagramIcon />
  </div>

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
        :database-metadata="dbSchemaStore.getDatabaseMetadata(database.name)"
      />
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { reactive } from "vue";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";

type LocalState = {
  show: boolean;
  fetchLast: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
}>();

const state = reactive<LocalState>({
  show: false,
  fetchLast: false,
});

const dbSchemaStore = useDBSchemaV1Store();

const openModal = async () => {
  if (!state.fetchLast) {
    await dbSchemaStore.getOrFetchDatabaseMetadata({
      database: props.database.name,
      skipCache: true,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    });
    state.fetchLast = true;
  }
  state.show = true;
};
</script>
