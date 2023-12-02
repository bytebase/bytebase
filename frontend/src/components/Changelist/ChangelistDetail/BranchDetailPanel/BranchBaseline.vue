<template>
  <div class="flex flex-row justify-start items-center opacity-80">
    <span class="mr-4 shrink-0"
      >{{ $t("schema-designer.baseline-version") }}:</span
    >
    <DatabaseInfo
      class="flex-nowrap mr-4 shrink-0"
      :database="baselineDatabase"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { useDatabaseV1Store } from "@/store";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

const props = defineProps<{
  branch: SchemaDesign;
}>();

const baselineDatabase = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(props.branch.baselineDatabase);
});
</script>
