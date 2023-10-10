<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent :title="$t('common.labels')">
      <LabelListEditor
        ref="labelListEditorRef"
        v-model:kv-list="state.kvList"
        :readonly="readonly"
        :show-errors="dirty"
        class="max-w-[30rem]"
      />
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-3">
            <NButton @click.prevent="$emit('dismiss')">
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              v-if="!readonly"
              :disabled="!allowSave"
              type="primary"
              @click="$emit('apply', convertKVListToLabels(state.kvList))"
            >
              {{ $t("common.save") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { computed, reactive, watch, ref } from "vue";
import LabelListEditor from "@/components/Label/LabelListEditor.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { type ComposedDatabase } from "@/types";
import { convertKVListToLabels, convertLabelsToKVList } from "@/utils";

const props = defineProps<{
  show: boolean;
  readonly: boolean;
  database: ComposedDatabase;
  labels: {
    [key: string]: string;
  };
}>();
defineEmits<{
  (event: "dismiss"): void;
  (
    event: "apply",
    labels: {
      [key: string]: string;
    }
  ): void;
}>();

type LocalState = {
  kvList: { key: string; value: string }[];
};

const state = reactive<LocalState>({
  kvList: [],
});
const labelListEditorRef = ref<InstanceType<typeof LabelListEditor>>();

const convert = () => {
  const labels = cloneDeep(props.labels);
  return convertLabelsToKVList(labels, true /* sort */);
};

const dirty = computed(() => {
  const original = convert();
  const local = state.kvList;
  return !isEqual(original, local);
});

const allowSave = computed(() => {
  if (!dirty.value) return false;
  const errors = labelListEditorRef.value?.flattenErrors ?? [];
  return errors.length === 0;
});

watch(
  () => props.labels,
  () => {
    state.kvList = convert();
  },
  {
    immediate: true,
    deep: true,
  }
);
</script>
