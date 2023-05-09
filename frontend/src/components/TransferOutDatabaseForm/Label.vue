<template>
  <EnvironmentName
    v-if="environment"
    :environment="environment"
    :link="false"
  />
  <template v-if="database">
    <div class="flex items-center gap-x-1">
      <InstanceName
        :instance="database.instance"
        :link="false"
        class="whitespace-nowrap"
      />
      <heroicons:chevron-right class="shrink-0 h-3 w-3 opacity-70" />
      <!-- eslint-disable-next-line vue/no-v-html -->
      <span class="truncate" v-html="databaseName" />
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { escape } from "lodash-es";

import { DatabaseTreeOption } from "./common";
import { useDatabaseStore, useEnvironmentStore } from "@/store";
import { EnvironmentName, InstanceName } from "@/components/v2";
import { getHighlightHTMLByRegExp } from "@/utils";

const props = defineProps<{
  option: DatabaseTreeOption;
  keyword?: string;
}>();

const id = computed(() => {
  return parseInt(props.option.value.split("-").pop()!, 10);
});

const environment = computed(() => {
  const { option } = props;
  if (option.level === "database") return undefined;
  return useEnvironmentStore().getEnvironmentById(id.value);
});

const database = computed(() => {
  const { option } = props;
  if (option.level === "environment") return undefined;
  return useDatabaseStore().getDatabaseById(id.value);
});

const databaseName = computed(() => {
  const name = database.value?.name ?? "";
  const keyword = (props.keyword ?? "").trim();

  return getHighlightHTMLByRegExp(
    escape(name),
    escape(keyword),
    false /* !caseSensitive */
  );
});
</script>
