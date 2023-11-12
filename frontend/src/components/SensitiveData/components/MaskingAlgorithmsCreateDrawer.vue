<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent :title="$t('settings.sensitive-data.algorithms.add')">
      <div
        class="w-[40rem] max-w-[calc(100vw-5rem)] space-y-6 divide-y divide-block-border"
      >
        <div class="space-y-6">
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="title" class="textlabel">
              {{ $t("settings.sensitive-data.algorithms.table.title") }}
              <span class="text-red-600 mr-2">*</span>
            </label>
            <input
              v-model="state.title"
              required
              name="title"
              type="text"
              placeholder="algorithm title"
              class="textfield mt-1 w-full"
              :disabled="state.processing || readonly"
            />
          </div>
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="description" class="textlabel">
              {{ $t("settings.sensitive-data.algorithms.table.description") }}
            </label>
            <input
              v-model="state.description"
              required
              name="title"
              type="text"
              placeholder="algorithm description"
              class="textfield mt-1 w-full"
              :disabled="state.processing || readonly"
            />
          </div>
          <div class="w-full mb-6 space-y-1">
            <label for="masking-type" class="textlabel">
              {{ $t("settings.sensitive-data.algorithms.table.masking-type") }}
              <span class="text-red-600 mr-2">*</span>
            </label>
            <div class="grid grid-cols-3 gap-2">
              <template v-for="(item, i) in maskingTypeList" :key="i">
                <div
                  class="flex relative justify-start p-2 border rounded"
                  :class="[
                    state.maskingType === item.id &&
                      'font-medium bg-control-bg-hover',
                    readonly
                      ? 'cursor-not-allowed'
                      : 'cursor-pointer hover:bg-control-bg-hover',
                  ]"
                  @click.capture="onMaskingTypeChange(item.id)"
                >
                  <div class="flex flex-row justify-start items-center">
                    <input
                      type="radio"
                      class="btn mr-2"
                      :checked="state.maskingType === item.id"
                      :disabled="state.processing || readonly"
                    />
                    <p class="text-center text-sm">
                      {{ item.title }}
                    </p>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </div>
        <div class="space-y-6 pt-6">
          <template v-if="state.maskingType === 'full-mask'">
            <div class="sm:col-span-2 sm:col-start-1">
              <label for="substitution" class="textlabel">
                {{
                  $t(
                    "settings.sensitive-data.algorithms.full-mask.substitution"
                  )
                }}
                <span class="text-red-600 mr-2">*</span>
              </label>
              <p class="textinfolabel">
                {{
                  $t(
                    "settings.sensitive-data.algorithms.full-mask.substitution-label"
                  )
                }}
              </p>
              <input
                v-model="state.fullMask.substitution"
                required
                name="title"
                type="text"
                placeholder="substitution"
                class="textfield mt-2 w-full"
                :disabled="state.processing || readonly"
              />
            </div>
          </template>
          <template v-if="state.maskingType === 'range-mask'">
            <p class="textinfolabel">
              {{ $t("settings.sensitive-data.algorithms.range-mask.label") }}
            </p>
            <div
              v-for="(slice, i) in state.rangeMask.slices"
              :key="i"
              class="flex space-x-2 items-center"
            >
              <div class="flex-none flex flex-col">
                <label for="slice-start" class="textlabel flex">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.range-mask.slice-start"
                    )
                  }}
                  <span class="text-red-600 mr-2">*</span>
                </label>
                <input
                  :value="slice.start"
                  required
                  name="slice-start"
                  type="number"
                  min="0"
                  placeholder="slice start"
                  class="textfield mt-1 w-20"
                  :disabled="state.processing || readonly"
                  @input="(e: any) => onSliceStartChange(i, Number(e.target.value))"
                />
              </div>
              <div class="flex-none flex flex-col">
                <label for="slice-end" class="textlabel">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.range-mask.slice-end"
                    )
                  }}
                  <span class="text-red-600 mr-2">*</span>
                </label>
                <input
                  :value="slice.end"
                  required
                  name="slice-end"
                  type="number"
                  min="1"
                  placeholder="slice end"
                  class="textfield mt-1 w-20"
                  :disabled="state.processing || readonly"
                  @input="(e: any) => onSliceEndChange(i, Number(e.target.value))"
                />
              </div>
              <div class="flex-1 flex flex-col">
                <label for="substitution" class="textlabel">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.range-mask.substitution"
                    )
                  }}
                  <span class="text-red-600 mr-2">*</span>
                </label>
                <input
                  v-model="slice.substitution"
                  required
                  name="substitution"
                  type="text"
                  placeholder="substitution"
                  class="textfield mt-1 w-full"
                  :disabled="state.processing || readonly"
                />
              </div>
              <div class="mt-5">
                <button
                  class="p-1 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                  :disabled="state.processing || readonly"
                  @click.stop="removeSlice(i)"
                >
                  <heroicons-outline:trash class="w-4 h-4" />
                </button>
              </div>
            </div>
            <NButton
              class="ml-auto"
              :disabled="state.processing || readonly"
              @click.prevent="addSlice"
            >
              {{ $t("common.add") }}
            </NButton>
          </template>
          <template v-if="state.maskingType === 'md5-mask'">
            <div class="sm:col-span-2 sm:col-start-1">
              <label for="salt" class="textlabel">
                {{ $t("settings.sensitive-data.algorithms.md5-mask.salt") }}
                <span class="text-red-600 mr-2">*</span>
              </label>
              <p class="textinfolabel">
                {{
                  $t("settings.sensitive-data.algorithms.md5-mask.salt-label")
                }}
              </p>
              <input
                v-model="state.md5Mask.salt"
                required
                name="title"
                type="text"
                placeholder="salt"
                class="textfield mt-2 w-full"
                :disabled="state.processing || readonly"
              />
            </div>
          </template>
        </div>
      </div>
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-3">
            <NButton @click.prevent="$emit('dismiss')">
              {{ $t("common.cancel") }}
            </NButton>
            <NTooltip :disabled="!errorMessage">
              <template #trigger>
                <NButton
                  v-if="!readonly"
                  :disabled="isSubmitDisabled"
                  type="primary"
                  @click.prevent="onUpsert"
                >
                  {{ $t("common.confirm") }}
                </NButton>
              </template>
              {{ errorMessage }}
            </NTooltip>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>
<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import { computed, watch, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import { pushNotification, useSettingV1Store } from "@/store";
import {
  MaskingAlgorithmSetting_Algorithm,
  MaskingAlgorithmSetting_Algorithm_FullMask,
  MaskingAlgorithmSetting_Algorithm_RangeMask,
  MaskingAlgorithmSetting_Algorithm_MD5Mask,
  MaskingAlgorithmSetting_Algorithm_RangeMask_Slice,
} from "@/types/proto/v1/setting_service";
import { MaskingType, getMaskingType } from "./utils";

interface MaskingTypeItem {
  id: MaskingType;
  title: string;
}

interface LocalState {
  processing: boolean;
  maskingType: MaskingType;
  title: string;
  description: string;
  fullMask: MaskingAlgorithmSetting_Algorithm_FullMask;
  rangeMask: MaskingAlgorithmSetting_Algorithm_RangeMask;
  md5Mask: MaskingAlgorithmSetting_Algorithm_MD5Mask;
}

const props = defineProps<{
  show: boolean;
  readonly: boolean;
  algorithm: MaskingAlgorithmSetting_Algorithm;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const defaultRangeMask = computed(() =>
  MaskingAlgorithmSetting_Algorithm_RangeMask.fromPartial({
    slices: [
      MaskingAlgorithmSetting_Algorithm_RangeMask_Slice.fromPartial({
        start: 0,
        end: 1,
      }),
    ],
  })
);

const state = reactive<LocalState>({
  processing: false,
  maskingType: "full-mask",
  title: "",
  description: "",
  fullMask: MaskingAlgorithmSetting_Algorithm_FullMask.fromPartial({}),
  rangeMask: cloneDeep(defaultRangeMask.value),
  md5Mask: MaskingAlgorithmSetting_Algorithm_MD5Mask.fromPartial({}),
});
const { t } = useI18n();
const settingStore = useSettingV1Store();

const maskingTypeList = computed((): MaskingTypeItem[] => [
  {
    id: "full-mask",
    title: t("settings.sensitive-data.algorithms.full-mask.self"),
  },
  {
    id: "range-mask",
    title: t("settings.sensitive-data.algorithms.range-mask.self"),
  },
  {
    id: "md5-mask",
    title: t("settings.sensitive-data.algorithms.md5-mask.self"),
  },
]);

const algorithmList = computed((): MaskingAlgorithmSetting_Algorithm[] => {
  return (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  );
});

watch(
  () => props.algorithm,
  (algorithm) => {
    state.title = algorithm.title;
    state.description = algorithm.description;
    state.maskingType = getMaskingType(algorithm) ?? "full-mask";
    state.fullMask =
      algorithm.fullMask ??
      MaskingAlgorithmSetting_Algorithm_FullMask.fromPartial({});
    state.rangeMask = algorithm.rangeMask ?? cloneDeep(defaultRangeMask.value);
    state.md5Mask =
      algorithm.md5Mask ??
      MaskingAlgorithmSetting_Algorithm_MD5Mask.fromPartial({});
  }
);

const maskingAlgorithm = computed((): MaskingAlgorithmSetting_Algorithm => {
  const result = MaskingAlgorithmSetting_Algorithm.fromPartial({
    id: props.algorithm.id,
    title: state.title,
    description: state.description,
    category: state.maskingType === "md5-mask" ? "HASH" : "MASK",
  });

  switch (state.maskingType) {
    case "full-mask":
      result.fullMask = state.fullMask;
      break;
    case "range-mask":
      result.rangeMask = state.rangeMask;
      break;
    case "md5-mask":
      result.md5Mask = state.md5Mask;
      break;
  }

  return result;
});

const errorMessage = computed(() => {
  if (!state.title) {
    return "Title is required";
  }

  switch (state.maskingType) {
    case "full-mask":
      if (!state.fullMask.substitution) {
        return "Substitution is required";
      }
      return "";
    case "md5-mask":
      if (!state.md5Mask.salt) {
        return "Salt is required";
      }
      return "";
    case "range-mask":
      if (state.rangeMask.slices.length === 0) {
        return "Slices is required";
      }
      for (let i = 0; i < state.rangeMask.slices.length; i++) {
        const slice = state.rangeMask.slices[i];
        if (!slice.substitution) {
          return "Slice substitution is required";
        }
        if (
          Number.isNaN(slice.start) ||
          Number.isNaN(slice.end) ||
          slice.start < 0 ||
          slice.end <= 0
        ) {
          return "Slice start or end is not valid number";
        }
        if (slice.start >= slice.end) {
          return "The slice end must smaller than the start";
        }

        for (let j = 0; j < i; j++) {
          const pre = state.rangeMask.slices[j];
          if (slice.start >= pre.end || pre.start >= slice.end) {
            continue;
          }
          return "The slice range cannot overlap";
        }
      }
  }
  return "";
});

const isSubmitDisabled = computed(() => {
  if (props.readonly || state.processing) {
    return true;
  }

  if (!state.title) {
    return true;
  }

  return !!errorMessage.value;
});

const onUpsert = async () => {
  state.processing = true;

  const index = algorithmList.value.findIndex(
    (item) => item.id === props.algorithm.id
  );
  const newList = [...algorithmList.value];
  if (index < 0) {
    newList.push(maskingAlgorithm.value);
  } else {
    newList[index] = maskingAlgorithm.value;
  }

  try {
    await settingStore.upsertSetting({
      name: "bb.workspace.masking-algorithm",
      value: {
        maskingAlgorithmSettingValue: {
          algorithms: newList,
        },
      },
    });

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    emit("dismiss");
  } finally {
    state.processing = false;
  }
};

const onMaskingTypeChange = (maskingType: MaskingType) => {
  if (state.processing || props.readonly) {
    return;
  }
  switch (maskingType) {
    case "full-mask":
      state.fullMask = MaskingAlgorithmSetting_Algorithm_FullMask.fromPartial(
        {}
      );
      break;
    case "range-mask":
      state.rangeMask = cloneDeep(defaultRangeMask.value);
      break;
    case "md5-mask":
      state.md5Mask = MaskingAlgorithmSetting_Algorithm_MD5Mask.fromPartial({});
      break;
  }
  state.maskingType = maskingType;
};

const addSlice = () => {
  state.rangeMask.slices.push(
    MaskingAlgorithmSetting_Algorithm_RangeMask_Slice.fromPartial({
      start: 0,
      end: 1,
    })
  );
};

const removeSlice = (i: number) => {
  state.rangeMask.slices.splice(i, 1);
};

const onSliceStartChange = (index: number, val: number) => {
  if (Number.isNaN(val)) {
    return;
  }
  const slice = state.rangeMask.slices[index];
  slice.start = Math.max(0, val);
  if (slice.end <= slice.start) {
    slice.end = slice.start + 1;
  }
};

const onSliceEndChange = (index: number, val: number) => {
  if (Number.isNaN(val)) {
    return;
  }
  const slice = state.rangeMask.slices[index];
  slice.end = Math.max(1, val);
  if (slice.start >= slice.end) {
    slice.start = slice.end - 1;
  }
};
</script>
