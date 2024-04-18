<template>
  <div v-if="databaseList.length > 0" class="mt-4">
    <div class="px-0.5">
      <NCheckbox v-model:checked="expand">
        {{ $t("database.show-schemaless-databases") }}
      </NCheckbox>
    </div>
    <DatabaseDataTable
      v-if="expand"
      :mode="`${mode}_SHORT`"
      :schemaless="true"
      :database-list="databaseList"
      :show-selection="false"
      :show-sql-editor-button="false"
    />
  </div>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { ref } from "vue";
import DatabaseDataTable from "@/components/DatabaseDataTable";
import type { ComposedDatabase } from "@/types";

defineProps<{
  databaseList: ComposedDatabase[];
  mode: "PROJECT" | "ALL";
}>();

const expand = ref(false);
</script>
