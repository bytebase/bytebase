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
        <CopyButton :content="schemaString || ''" />
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
import { create } from "@bufbuild/protobuf";
import { computed, nextTick, onMounted, ref } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { CopyButton } from "@/components/v2";
import { databaseServiceClientConnect } from "@/connect";
import type { ComposedDatabase } from "@/types";
import type { GetSchemaStringRequest_ObjectType } from "@/types/proto-es/v1/database_service_pb";
import { GetSchemaStringRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { hasSchemaProperty } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
  schema?: string;
  object?: string;
  type?: GetSchemaStringRequest_ObjectType;
}>();

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
    const request = create(GetSchemaStringRequestSchema, {
      name: props.database.name,
      type: props.type,
      schema: props.schema,
      object: props.object,
    });
    const response =
      await databaseServiceClientConnect.getSchemaString(request);
    schemaString.value = response.schemaString.trim();
  });
});
</script>
