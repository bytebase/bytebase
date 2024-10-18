<template>
  <div class="flex flex-col space-y-2" v-bind="$attrs">
    <PagedRevisionTable
      :key="pagedRevisionTableSessionKey"
      :database="database"
      :session-key="pagedRevisionTableSessionKey"
    >
      <template #table="{ list, loading }">
        <RevisionDataTable
          :key="`revision-table.${database.name}`"
          :loading="loading"
          :revisions="list"
          :custom-click="true"
          :show-selection="true"
          @row-click="state.selectedRevisionName = $event"
        />
      </template>
    </PagedRevisionTable>
  </div>

  <Drawer
    :show="!!state.selectedRevisionName"
    @close="state.selectedRevisionName = undefined"
  >
    <DrawerContent class="w-320 max-w-[80vw]" :title="'Revision'">
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
import { useDatabaseDetailContext } from "./context";

interface LocalState {
  selectedRevisionName?: string;
}

defineProps<{
  database: ComposedDatabase;
}>();

const { pagedRevisionTableSessionKey } = useDatabaseDetailContext();
const state: LocalState = reactive({});
</script>
