<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent :title="title">
      <BBAttention
        v-if="showWarning"
        class="mb-5"
        :style="'WARN'"
        :title="$t('database.mixed-label-values-warning')"
      />
      <LabelListEditor
        ref="labelListEditorRef"
        v-model:kv-list="state.kvList"
        :readonly="readonly"
        :show-errors="dirty"
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
              @click="onSave"
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
import { isEqual, cloneDeep } from "lodash-es";
import { computed, reactive, watch, ref } from "vue";
import { useI18n } from "vue-i18n";
import { LabelListEditor } from "@/components/Label";
import { Label } from "@/components/Label/types";
import { Drawer, DrawerContent } from "@/components/v2";
import { convertKVListToLabels } from "@/utils";

const props = defineProps<{
  show: boolean;
  readonly: boolean;
  title: string;
  labels: {
    [key: string]: string;
  }[];
}>();
const emit = defineEmits<{
  (event: "dismiss"): void;
  (
    event: "apply",
    labels: {
      [key: string]: string;
    }[]
  ): void;
}>();

type LocalState = {
  kvList: Label[];
};

const { t } = useI18n();
const state = reactive<LocalState>({
  kvList: [],
});
const labelListEditorRef = ref<InstanceType<typeof LabelListEditor>>();

const convertKVListToLabelsList = () => {
  return props.labels.map((oldLabel) => {
    const labels = convertKVListToLabels(state.kvList);
    for (const label of state.kvList) {
      if (!label.value) {
        if (oldLabel[label.key]) {
          labels[label.key] = oldLabel[label.key];
        }
      }
    }
    return labels;
  });
};

const convertLabelsToKVList = () => {
  interface TmpLabel {
    [key: string]: {
      value: string;
      message?: string;
      allowEmpty?: boolean;
    };
  }

  let tmp: TmpLabel = {};
  if (props.labels.length > 0) {
    tmp = Object.entries(props.labels[0]).reduce((resp, [key, value]) => {
      resp[key] = {
        value,
      };
      return resp;
    }, {} as TmpLabel);
  }
  for (let i = 1; i < props.labels.length; i++) {
    const label = cloneDeep(props.labels[i]);
    for (const key of Object.keys(tmp)) {
      if (!label[key] || label[key] !== tmp[key].value) {
        tmp[key] = {
          message: t("database.mixed-values-for-label"),
          value: "",
          allowEmpty: true,
        };
      }
      delete label[key];
    }
    for (const [key, value] of Object.entries(label)) {
      if (!tmp[key]) {
        tmp[key] = {
          value,
        };
      } else if (tmp[key].value !== value) {
        tmp[key] = {
          message: t("database.mixed-values-for-label"),
          value: "",
          allowEmpty: true,
        };
      }
    }
  }

  const list: Label[] = Object.entries(tmp).map(([key, obj]) => ({
    key,
    value: obj.value,
    message: obj.message,
    allowEmpty: obj.allowEmpty,
  }));

  return list;
};

const dirty = computed(() => {
  const original = convertLabelsToKVList();
  const local = state.kvList;
  return !isEqual(original, local);
});

const allowSave = computed(() => {
  if (!dirty.value) return false;
  const errors = labelListEditorRef.value?.flattenErrors ?? [];
  return errors.length === 0;
});

const showWarning = computed(() => {
  return state.kvList.some((kv) => !!kv.message);
});

const onSave = () => {
  emit("apply", convertKVListToLabelsList());
  emit("dismiss");
};

watch(
  () => props.labels,
  () => {
    state.kvList = convertLabelsToKVList();
  },
  {
    immediate: true,
    deep: true,
  }
);
</script>
