<template>
  <div class="flex flex-col space-y-4" v-bind="$attrs">
    <PagedRevisionTable
      :database="database"
      session-key="bb.paged-revision-table"
    >
      <template #table="{ list, loading }">
        <RevisionDataTable
          :key="`revision-table.${database.name}`"
          :loading="loading"
          :revisions="list"
          :custom-click="true"
          @row-click="state.selectedRevisionName = $event"
        />
      </template>
    </PagedRevisionTable>
  </div>

  <Drawer
    :show="!!state.selectedRevisionName"
    @close="state.selectedRevisionName = undefined"
  >
    <DrawerContent class="w-192 max-w-[80vw]" :title="'Revision'">
      <RevisionDetailPanel
        :database="database"
        :revision-name="state.selectedRevisionName!"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import { PagedRevisionTable, RevisionDataTable } from "@/components/Revision";
import type { ComposedDatabase } from "@/types";
import RevisionDetailPanel from "../Revision/RevisionDetailPanel.vue";
import { Drawer, DrawerContent } from "../v2";

interface LocalState {
  selectedRevisionName?: string;
}

defineProps<{
  database: ComposedDatabase;
}>();

const state: LocalState = reactive({});
</script>
