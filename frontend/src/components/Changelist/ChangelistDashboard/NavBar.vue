<template>
  <div class="flex flex-row items-center gap-x-2">
    <SearchBox
      v-model:value="filter.keyword"
      style="max-width: 100%"
      :placeholder="$t('common.filter-by-name')"
    />
    <NButton v-if="allowCreate" type="primary" @click="showCreatePanel = true">
      <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
      {{ $t("changelist.new") }}
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { SearchBox } from "@/components/v2";
import { useChangelistDashboardContext } from "./context";

defineProps<{
  allowCreate: boolean;
}>();

const { filter, showCreatePanel } = useChangelistDashboardContext();
</script>
