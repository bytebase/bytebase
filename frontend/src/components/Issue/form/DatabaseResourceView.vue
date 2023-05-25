<template>
  <div
    class="flex flex-row justify-start items-center flex-nowrap whitespace-nowrap"
  >
    <Prefix class="flex-shrink-0 mr-1" />
    {{ database.name }}
    <template v-if="props.databaseResource.schema">
      /
      {{ props.databaseResource.schema }}
    </template>
    <template v-if="props.databaseResource.table">
      /
      {{ props.databaseResource.table }}
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, h } from "vue";
import { DatabaseResource } from "./SelectDatabaseResourceForm/common";
import { useDatabaseStore } from "@/store";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import TableIcon from "~icons/heroicons-outline/table-cells";

const props = defineProps<{
  databaseResource: DatabaseResource;
}>();

const databaseStore = useDatabaseStore();

const database = computed(() => {
  return databaseStore.getDatabaseById(props.databaseResource.databaseId);
});

const Prefix = () => {
  if (props.databaseResource.table !== undefined) {
    return h(TableIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  } else if (props.databaseResource.schema !== undefined) {
    return h(SchemaIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  }
  return h(DatabaseIcon, {
    class: "w-4 h-auto text-gray-400",
  });
};
</script>
