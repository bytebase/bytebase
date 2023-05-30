<template>
  <div class="flex items-center gap-x-1">
    <span v-if="environment" class="text-gray-400">
      (<EnvironmentV1Name :environment="environment" :link="false" />)
    </span>
    <Prefix />
    <!-- eslint-disable-next-line vue/no-v-html -->
    <span class="truncate" v-html="optionName" />
    <span v-if="database" class="ml-1 text-gray-500 flex flex-row items-center">
      (<InstanceName
        :instance="database.instance"
        :link="false"
        class="whitespace-nowrap"
      />)
    </span>
  </div>
</template>

<script setup lang="ts">
import { escape } from "lodash-es";
import { computed, h } from "vue";
import { DatabaseTreeOption } from "./common";
import { useDatabaseStore, useEnvironmentV1Store } from "@/store";
import { getHighlightHTMLByRegExp } from "@/utils";
import { EnvironmentV1Name, InstanceName } from "@/components/v2";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
import TableIcon from "~icons/heroicons-outline/table-cells";

const props = defineProps<{
  option: DatabaseTreeOption;
  keyword?: string;
}>();

const option = computed(() => props.option);

const database = computed(() => {
  const { option } = props;
  if (option.level !== "database") return undefined;
  const databaseId = option.value.replace("d-", "");
  return useDatabaseStore().getDatabaseById(databaseId);
});

const environment = computed(() => {
  const { option } = props;
  if (option.level !== "database") return undefined;
  return useEnvironmentV1Store().getEnvironmentByUID(
    database.value?.instance.environment.id as string
  );
});

const Prefix = () => {
  if (option.value.level === "database") {
    return h(DatabaseIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  } else if (option.value.level === "schema") {
    return h(SchemaIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  } else if (option.value.level === "table") {
    return h(TableIcon, {
      class: "w-4 h-auto text-gray-400",
    });
  }
  return null;
};

const optionName = computed(() => {
  const name = option.value?.label ?? "";
  const keyword = (props.keyword ?? "").trim();

  return getHighlightHTMLByRegExp(
    escape(name),
    escape(keyword),
    false /* !caseSensitive */
  );
});
</script>
