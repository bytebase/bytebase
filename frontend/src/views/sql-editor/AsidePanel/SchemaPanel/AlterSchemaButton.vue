<template>
  <NTooltip trigger="hover" :delay="500" :animated="false">
    <template #trigger>
      <NButton
        quaternary
        size="tiny"
        class="!px-1"
        v-bind="$attrs"
        @click="emit('click')"
      >
        <heroicons-outline:pencil-alt class="w-4 h-4" />
      </NButton>
    </template>
    {{ $t("database.alter-schema") }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import type { ComposedDatabase } from "@/types";
import type {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/store/database";

defineProps<{
  database: ComposedDatabase;
  schema?: SchemaMetadata;
  table?: TableMetadata;
}>();

const emit = defineEmits<{
  (event: "click"): void;
}>();
</script>
