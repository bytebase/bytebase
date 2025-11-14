<template>
  <BBModal :title="$t('common.preview')" @close="$emit('cancel')">
    <div class="w-144 py-1 flex flex-col justify-start items-start gap-2">
      <div class="w-full flex flex-row justify-between items-center gap-4">
        <p class="font-medium textlabel text-nowrap">
          {{ $t("sql-editor.select-encoding") }}
        </p>
        <NSelect
          v-model:value="state.encoding"
          class="w-auto!"
          filterable
          :options="encodingOptions"
        />
      </div>
      <div class="w-full overflow-hidden relative h-80 shrink-0">
        <MonacoEditor
          class="border w-full h-full"
          :content="decodedText"
          :readonly="true"
        />
        <NSpin v-if="isLoading" class="absolute inset-0" />
      </div>
      <div class="w-full flex justify-end gap-x-2">
        <NButton @click="$emit('cancel')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowConfirm"
          :loading="isLoading"
          @click="$emit('confirm', decodedText)"
        >
          {{ $t("common.confirm") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NButton, NSelect, NSpin } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { BBModal } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";
import { pushNotification } from "@/store";
import { ENCODINGS, type Encoding, readFileAsArrayBuffer } from "@/utils";

interface LocalState {
  encoding: Encoding;
}

const props = defineProps<{
  file: File;
}>();

defineEmits<{
  (event: "cancel"): void;
  (event: "confirm", text: string): void;
}>();

const state = reactive<LocalState>({
  encoding: "utf-8",
});
const isLoading = ref(true);
const decodedText = ref<string>("");

const allowConfirm = computed(() => {
  return !isLoading.value;
});

const encodingOptions = computed(() =>
  ENCODINGS.map((encoding) => ({
    label: encoding,
    value: encoding,
  }))
);

watch(
  [() => props.file, state],
  async () => {
    isLoading.value = true;
    try {
      const { arrayBuffer } = await readFileAsArrayBuffer(props.file);
      const text = new TextDecoder(state.encoding).decode(arrayBuffer);
      decodedText.value = text;
    } catch (error) {
      console.error(error);
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Failed to read file",
      });
    }
    isLoading.value = false;
  },
  {
    immediate: true,
  }
);
</script>
