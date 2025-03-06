<template>
  <div v-if="databaseList.length > 0" class="mt-4">
    <div class="px-0.5 mb-2">
      <NCheckbox v-model:checked="expand">
        {{ $t("database.show-schemaless-databases") }}
      </NCheckbox>
    </div>
    <DatabaseV1Table
      v-if="expand"
      :mode="`${mode}_SHORT`"
      :schemaless="true"
      :database-list="databaseList"
      :show-selection="false"
    />
  </div>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { ref } from "vue";
import { DatabaseV1Table } from "@/components/v2/Model/DatabaseV1Table";
import type { ComposedDatabase } from "@/types";

defineProps<{
  databaseList: ComposedDatabase[];
  mode: "PROJECT" | "ALL";
}>();

const expand = ref(false);
</script>
