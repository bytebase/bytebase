<template>
  <div
    class="w-full h-auto flex flex-col justify-start items-center"
    v-bind="$attrs"
  >
    <div class="w-full flex flex-row justify-between items-center gap-x-2 mb-2">
      <div class="textlabel flex-1 truncate">
        {{ resourceName }}
      </div>
      <div class="flex flex-row justify-end items-center">
        <NTooltip>
          <template #trigger>
            <NButton
              quaternary
              size="tiny"
              class="!px-1"
              @click="handleCopySchemaString"
            >
              <ClipboardIcon class="w-4 h-4" />
            </NButton>
          </template>
          {{ $t("common.copy") }}
        </NTooltip>
      </div>
    </div>
    <MonacoEditor
      :content="schemaString || ''"
      :readonly="true"
      class="border w-full h-auto grow"
    />
  </div>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import { ClipboardIcon } from "lucide-vue-next";
import { NButton, NTooltip } from "naive-ui";
import { onMounted, ref, computed } from "vue";
import { nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { MonacoEditor } from "@/components/MonacoEditor";
import { databaseServiceClient } from "@/grpcweb";
import {
  pushNotification,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import {
  GetSchemaStringRequest_ObjectType,
} from "@/types/proto/v1/database_service";
import { hasSchemaProperty } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
  schema?: string;
  object?: string;
  type?: GetSchemaStringRequest_ObjectType;
}>();

const { t } = useI18n();

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});
const schemaString = ref<string | null>(null);

const engine = computed(() => {
  return props.database.instanceResource.engine;
});

const resourceName = computed(() => {
  if (props.object) {
    if (hasSchemaProperty(engine.value)) {
      return `${props.schema}.${props.object}`;
    } else {
      return props.object;
    }
  }
  if (props.schema) {
    return props.schema;
  }
  return props.database.name;
});

onMounted(async () => {
  nextTick(async () => {
    const response = await databaseServiceClient.getSchemaString({
      name: props.database.name,
      type: props.type,
      schema: props.schema,
      object: props.object,
    })
    schemaString.value = response.schemaString.trim();
  });
});

const handleCopySchemaString = () => {
  if (!isSupported.value) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Copy to clipboard is not enabled in your browser.",
    });
    return;
  }

  copyTextToClipboard(schemaString.value || "");
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-editor.copy-success"),
  });
};
</script>
