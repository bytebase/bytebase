<template>
  <div
    v-if="!hidePreview"
    v-show="layoutReady"
    class="overflow-hidden flex flex-col items-stretch shrink-0 transition-all"
    :style="{
      height: expanded ? `${panelHeight}px` : '1rem',
    }"
    :data-parent-height="parentHeight"
  >
    <div
      class="cursor-pointer flex items-center justify-start px-2 gap-1 overflow-y-visible bg-control-bg hover:bg-control-bg-hover"
      :style="{
        height: '1rem',
      }"
      @click="expanded = !expanded"
    >
      <ChevronDownIcon
        class="w-4 h-4 transition-transform"
        :class="expanded ? '' : 'rotate-180'"
      />
      <span v-if="true || expanded" class="text-xs origin-left">
        {{ title }}
      </span>
    </div>
    <div class="relative flex-1 overflow-hidden">
      <MonacoEditor
        v-if="status === 'pending' || status === 'success'"
        :readonly="true"
        :content="data?.trim() ?? ''"
        :options="{
          fontSize: '12px',
          lineHeight: '18px',
        }"
        class="w-full h-full"
      />
      <pre v-if="expanded && status === 'error'" class="text-sm text-error">{{
        error?.message ?? "Unknown error"
      }}</pre>
      <MaskSpinner v-if="status === 'pending'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import {
  refDebounced,
  useElementSize,
  useLocalStorage,
  useParentElement,
} from "@vueuse/core";
import { ChevronDownIcon } from "lucide-vue-next";
import { computed, toRef, ref, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import MaskSpinner from "@/components/misc/MaskSpinner.vue";
import { databaseServiceClientConnect } from "@/grpcweb";
import type { ComposedDatabase } from "@/types";
import type { DatabaseCatalog } from "@/types/proto-es/v1/database_catalog_service_pb";
import type {
  DatabaseMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { GetSchemaStringRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { minmax } from "@/utils";
import { extractGrpcErrorMessage } from "@/utils/grpcweb";
import { useSchemaEditorContext } from "../context";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  title: string;
  mocked: { metadata: DatabaseMetadata; catalog: DatabaseCatalog } | undefined;
}>();

const { hidePreview } = useSchemaEditorContext();
const expanded = useLocalStorage("bb.schema-editor.preview.expanded", true);
const parentElement = useParentElement();
const { height: parentHeight } = useElementSize(parentElement);
const layoutReady = computed(() => parentHeight.value > 0);
const panelHeight = computed(() => {
  const min = 8 * 16; // 8rem ~= 6 lines
  const max = 16 * 16; // 16rem ~= 13 lines
  const flexible = parentHeight.value * 0.4;
  return minmax(flexible, min, max);
});

const mocked = toRef(props, "mocked");
const debouncedMocked = refDebounced(mocked, 500);

// Simple replacement for useQuery to avoid @tanstack/vue-query dependency
const status = ref<"pending" | "success" | "error">("pending");
const data = ref<string>("");
const error = ref<Error | null>(null);

const fetchSchemaString = async (currentMocked = debouncedMocked.value) => {
  if (!expanded.value) {
    data.value = "";
    status.value = "success";
    return;
  }

  if (!currentMocked) {
    data.value = "";
    status.value = "success";
    return;
  }

  status.value = "pending";
  error.value = null;

  try {
    const { metadata } = currentMocked;
    const request = create(GetSchemaStringRequestSchema, {
      name: props.db.name,
      metadata: metadata,
    });
    const response =
      await databaseServiceClientConnect.getSchemaString(request);

    // Only update if this is still the latest request (avoid race conditions)
    if (currentMocked === debouncedMocked.value) {
      data.value = response.schemaString;
      status.value = "success";
    }
  } catch (err) {
    // Only update error if this is still the latest request
    if (currentMocked === debouncedMocked.value) {
      error.value = new Error(extractGrpcErrorMessage(err));
      status.value = "error";
    }
  }
};

// Watch for changes and refetch
watch(
  [debouncedMocked, expanded],
  ([newMocked]) => {
    fetchSchemaString(newMocked);
  },
  { immediate: true }
);
</script>
