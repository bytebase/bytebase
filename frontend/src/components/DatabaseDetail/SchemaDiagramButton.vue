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
    v-if="open"
    :title="$t('schema-diagram.self')"
    class="h-[calc(100vh-40px)] max-h-[calc(100vh-40px)]!"
    header-class="border-0!"
    container-class="flex-1 pt-0!"
    @close="open = false"
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
import { ref } from "vue";
import { BBModal } from "@/bbkit";
import { SchemaDiagram, SchemaDiagramIcon } from "@/components/SchemaDiagram";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const open = ref<boolean>(false);
const dbSchemaStore = useDBSchemaV1Store();

const openModal = async () => {
  await dbSchemaStore.getOrFetchDatabaseMetadata({
    database: props.database.name,
    skipCache: false,
  });
  open.value = true;
};
</script>
