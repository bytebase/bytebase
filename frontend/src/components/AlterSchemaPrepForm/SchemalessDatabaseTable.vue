<template>
  <div v-if="databaseList.length > 0" class="mt-4">
    <div class="px-0.5">
      <NCheckbox v-model:checked="expand">
        {{ $t("database.show-schemaless-databases") }}
      </NCheckbox>
    </div>
    <DatabaseV1Table
      v-if="expand"
      :mode="`${mode}_SHORT`"
      class="overflow-y-auto mt-2"
      table-class="border"
      :schemaless="true"
      :row-clickable="false"
      :database-list="databaseList"
    />
  </div>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { ref } from "vue";
import { ComposedDatabase } from "@/types";
import { DatabaseV1Table } from "../v2/";

defineProps<{
  databaseList: ComposedDatabase[];
  mode: "PROJECT" | "ALL";
}>();

const expand = ref(false);
</script>
