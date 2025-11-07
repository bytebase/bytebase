<template>
  <div class="flex items-center gap-x-2">
    <DatabaseV1Name
      :database="database"
      :link="false"
      tag="span"
      :keyword="keyword"
    />
    <NTooltip v-if="database.drifted">
      <template #trigger>
        <GitPullRequestClosedIcon :size="16" class="text-warning-hover" />
      </template>
      {{ $t("database.drifted.schema-drift-detected.self") }}
    </NTooltip>
  </div>
</template>

<script setup lang="ts">
import { GitPullRequestClosedIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { DatabaseV1Name } from "@/components/v2";
import type { ComposedDatabase } from "@/types";

withDefaults(
  defineProps<{
    database: ComposedDatabase;
    keyword?: string;
  }>(),
  {
    keyword: undefined,
  }
);
</script>
