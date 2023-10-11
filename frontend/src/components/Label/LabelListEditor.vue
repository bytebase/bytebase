<template>
  <div class="flex flex-col gap-y-2">
    <div
      class="grid gap-x-1 gap-y-2"
      style="grid-template-columns: 1fr 1fr auto"
    >
      <LabelEditorRow
        v-for="(kv, i) in kvList"
        :key="i"
        :kv="kv"
        :index="i"
        :readonly="readonly"
        :errors="errorsForKV(i)"
        @update-key="updateKey(i, $event)"
        @update-value="updateValue(i, $event)"
        @remove="handleRemove(i)"
      />
    </div>

    <div v-if="!readonly">
      <NButton size="small" :disabled="!allowAddLabel" @click="handleAdd">
        <template #icon>
          <heroicons:plus />
        </template>
        {{ $t("label.add-label") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MAX_LABEL_VALUE_LENGTH, isVirtualLabelKey } from "@/utils";
import LabelEditorRow from "./LabelEditorRow.vue";
import { Label } from "./types";

const props = defineProps<{
  kvList: Label[];
  readonly: boolean;
  showErrors: boolean;
}>();

const emit = defineEmits<{
  (event: "update:kvList", kvList: Label[]): void;
}>();

const { t } = useI18n();

const errorList = computed(() => {
  const { kvList } = props;
  const list: { key: string[]; value: string[] }[] = [];
  for (let i = 0; i < kvList.length; i++) {
    const kv = kvList[i];

    const { key, value } = kv;
    const errors = {
      key: [] as string[],
      value: [] as string[],
    };
    if (!key) {
      errors.key.push(t("label.error.key-necessary"));
    } else {
      if (isVirtualLabelKey(key)) {
        errors.key.push(t("label.error.x-is-reserved-key", { key }));
      }
      if (kvList.filter((kv) => kv.key === key).length > 1) {
        errors.key.push(t("label.error.key-duplicated"));
      }
    }
    if (!value) {
      if (!kv.allowEmpty) {
        errors.value.push(t("label.error.value-necessary"));
      }
    } else if (value.length > MAX_LABEL_VALUE_LENGTH) {
      errors.value.push(
        t("label.error.max-value-length-exceeded", {
          length: MAX_LABEL_VALUE_LENGTH,
        })
      );
    }
    list.push(errors);
  }

  return list;
});

const errorsForKV = (index: number) => {
  if (!props.showErrors) {
    return { key: [], value: [] };
  }
  return errorList.value[index] ?? { key: [], value: [] };
};

const flattenErrorsForKV = (index: number) => {
  const errors = errorsForKV(index);
  return [...errors.key, ...errors.value];
};

const flattenErrors = computed(() => {
  const flattenErrors: { key: string; errors: string[] }[] = [];
  props.kvList.forEach((kv, i) => {
    const { key } = kv;
    const errors = flattenErrorsForKV(i);
    if (errors.length > 0) {
      flattenErrors.push({ key, errors });
    }
  });
  return flattenErrors;
});

const allowAddLabel = computed(() => {
  return flattenErrors.value.length === 0;
});

const updateKey = (index: number, key: string) => {
  const list = cloneDeep(props.kvList);
  const kv = list[index];
  if (!kv) return;
  kv.key = key;
  emit("update:kvList", list);
};

const updateValue = (index: number, value: string) => {
  const list = cloneDeep(props.kvList);
  const kv = list[index];
  if (!kv) return;
  kv.value = value;
  emit("update:kvList", list);
};

const handleRemove = (index: number) => {
  const list = cloneDeep(props.kvList);
  list.splice(index, 1);
  emit("update:kvList", list);
};

const handleAdd = () => {
  const list = cloneDeep(props.kvList);
  emit("update:kvList", [...list, { key: "", value: "" }]);
};

defineExpose({
  errorList,
  flattenErrors,
});
</script>
