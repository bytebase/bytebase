<template>
  <div v-if="databaseList.length > 0" class="mt-4">
    <div class="px-0.5">
      <NCheckbox v-model:checked="expand">
        {{ $t("database.show-schemaless-databases") }}
      </NCheckbox>
    </div>
    <DatabaseTable
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
import { ref } from "vue";
import { NCheckbox } from "naive-ui";

import { Database } from "@/types";

defineProps<{
  databaseList: Database[];
  mode: "PROJECT" | "ALL";
}>();

const expand = ref(false);
</script>
