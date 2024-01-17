<template>
  <NTooltip
    v-if="allowAlterSchema"
    trigger="hover"
    :delay="500"
    :animated="false"
  >
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
    {{ $t("database.edit-schema") }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useCurrentUserV1 } from "@/store";
import { UNKNOWN_ID, type ComposedDatabase } from "@/types";
import type {
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { hasProjectPermissionV2, instanceV1HasAlterSchema } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
  schema?: SchemaMetadata;
  table?: TableMetadata;
}>();

const emit = defineEmits<{
  (event: "click"): void;
}>();

const me = useCurrentUserV1();

const allowAlterSchema = computed(() => {
  if (props.database.uid === String(UNKNOWN_ID)) {
    return false;
  }
  return (
    instanceV1HasAlterSchema(props.database.instanceEntity) &&
    hasProjectPermissionV2(
      props.database.projectEntity,
      me.value,
      "bb.issues.create"
    )
  );
});
</script>
