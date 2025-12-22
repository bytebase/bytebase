<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="
        algorithm
          ? $t('settings.sensitive-data.algorithms.edit')
          : $t('settings.sensitive-data.algorithms.add')
      "
    >
      <div class="w-[40rem] max-w-[calc(100vw-5rem)]">
        <div class="flex flex-col gap-y-6 pb-6">
          <div class="w-full mb-6 flex flex-col gap-y-1">
            <label for="masking-type" class="textlabel">
              {{ $t("settings.sensitive-data.algorithms.table.masking-type") }}
              <RequiredStar />
            </label>
            <RadioGrid
              :value="state.maskingType"
              :options="maskingTypeList"
              :disabled="readonly"
              class="grid-cols-3 gap-2"
              @update:value="onMaskingTypeChange($event as MaskingType)"
            >
              <template #item="{ option }">
                {{ option.label }}
              </template>
            </RadioGrid>
          </div>
        </div>
        <div class="flex flex-col gap-y-6 border-t border-block-border pt-6">
          <template v-if="state.maskingType === 'full-mask'">
            <div class="sm:col-span-2 sm:col-start-1">
              <label for="substitution" class="textlabel">
                {{
                  $t(
                    "settings.sensitive-data.algorithms.full-mask.substitution"
                  )
                }}
                <RequiredStar />
              </label>
              <p class="textinfolabel">
                {{
                  $t(
                    "settings.sensitive-data.algorithms.full-mask.substitution-label"
                  )
                }}
              </p>
              <NInput
                v-model:value="state.fullMask.substitution"
                :placeholder="
                  t('settings.sensitive-data.algorithms.full-mask.substitution')
                "
                class="mt-2"
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
              class="flex gap-x-2 items-center"
            >
              <div class="flex-none flex flex-col gap-y-1">
                <label for="slice-start" class="textlabel flex">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.range-mask.slice-start"
                    )
                  }}
                  <RequiredStar />
                </label>
                <NInputNumber
                  :value="slice.start"
                  :placeholder="
                    t(
                      'settings.sensitive-data.algorithms.range-mask.slice-start'
                    )
                  "
                  :disabled="state.processing || readonly"
                  style="width: 5rem"
                  @update:value="onSliceStartChange(i, $event)"
                />
              </div>
              <div class="flex-none flex flex-col gap-y-1">
                <label for="slice-end" class="textlabel">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.range-mask.slice-end"
                    )
                  }}
                  <RequiredStar />
                </label>
                <NInputNumber
                  :value="slice.end"
                  :placeholder="
                    t('settings.sensitive-data.algorithms.range-mask.slice-end')
                  "
                  :disabled="state.processing || readonly"
                  style="width: 5rem"
                  @update:value="onSliceEndChange(i, $event)"
                />
              </div>
              <div class="flex-1 flex flex-col gap-y-1">
                <label for="substitution" class="textlabel">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.range-mask.substitution"
                    )
                  }}
                  <RequiredStar />
                </label>
                <NInput
                  v-model:value="slice.substitution"
                  :placeholder="
                    t(
                      'settings.sensitive-data.algorithms.range-mask.substitution'
                    )
                  "
                  :disabled="state.processing || readonly"
                />
              </div>
              <div class="h-[34px] flex flex-row items-center self-end">
                <MiniActionButton
                  :disabled="state.processing || readonly"
                  @click.stop="removeSlice(i)"
                >
                  <TrashIcon class="w-4 h-4" />
                </MiniActionButton>
              </div>
            </div>
            <div v-if="rangeMaskErrorMessage" class="text-red-600">
              {{ rangeMaskErrorMessage }}
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
                <RequiredStar />
              </label>
              <p class="textinfolabel">
                {{
                  $t("settings.sensitive-data.algorithms.md5-mask.salt-label")
                }}
              </p>
              <NInput
                v-model:value="state.md5Mask.salt"
                :placeholder="
                  t('settings.sensitive-data.algorithms.md5-mask.salt')
                "
                class="mt-2"
                :disabled="state.processing || readonly"
              />
            </div>
          </template>
          <template v-if="state.maskingType === 'inner-outer-mask'">
            <label class="textlabel">
              {{
                $t("settings.sensitive-data.algorithms.inner-outer-mask.type")
              }}
              <RequiredStar />
              <p class="textinfolabel">
                {{
                  state.innerOuterMask.type ==
                  Algorithm_InnerOuterMask_MaskType.INNER
                    ? $t(
                        "settings.sensitive-data.algorithms.inner-outer-mask.inner-label"
                      )
                    : $t(
                        "settings.sensitive-data.algorithms.inner-outer-mask.outer-label"
                      )
                }}
              </p>
            </label>
            <NRadioGroup
              v-model:value="state.innerOuterMask.type"
              :disabled="state.processing || readonly"
            >
              <NRadio :value="Algorithm_InnerOuterMask_MaskType.INNER">
                {{
                  $t(
                    "settings.sensitive-data.algorithms.inner-outer-mask.inner-mask"
                  )
                }}
              </NRadio>
              <NRadio :value="Algorithm_InnerOuterMask_MaskType.OUTER">
                {{
                  $t(
                    "settings.sensitive-data.algorithms.inner-outer-mask.outer-mask"
                  )
                }}
              </NRadio>
            </NRadioGroup>
            <div class="flex gap-x-2 items-center">
              <div class="flex-none flex flex-col gap-y-1">
                <label for="slice-start" class="textlabel flex">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.inner-outer-mask.prefix-length"
                    )
                  }}
                  <RequiredStar />
                </label>
                <NInputNumber
                  :value="state.innerOuterMask.prefixLen"
                  :placeholder="
                    t(
                      'settings.sensitive-data.algorithms.inner-outer-mask.prefix-length'
                    )
                  "
                  :disabled="state.processing || readonly"
                  style="width: 5.5rem"
                  @update:value="onPrefixChange($event as number)"
                />
              </div>
              <div class="flex-none flex flex-col gap-y-1">
                <label for="slice-end" class="textlabel">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.inner-outer-mask.suffix-length"
                    )
                  }}
                  <RequiredStar />
                </label>
                <NInputNumber
                  :value="state.innerOuterMask.suffixLen"
                  :placeholder="
                    t(
                      'settings.sensitive-data.algorithms.inner-outer-mask.suffix-length'
                    )
                  "
                  :disabled="state.processing || readonly"
                  style="width: 5.5rem"
                  @update:value="onSuffixChange($event as number)"
                />
              </div>
              <div class="flex-1 flex flex-col gap-y-1">
                <label for="substitution" class="textlabel">
                  {{
                    $t(
                      "settings.sensitive-data.algorithms.range-mask.substitution"
                    )
                  }}
                  <RequiredStar />
                </label>
                <NInput
                  v-model:value="state.innerOuterMask.substitution"
                  :placeholder="
                    t(
                      'settings.sensitive-data.algorithms.range-mask.substitution'
                    )
                  "
                  :disabled="state.processing || readonly"
                />
              </div>
            </div>
          </template>
        </div>
      </div>
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-2">
            <NButton @click.prevent="$emit('dismiss')">
              {{ $t("common.cancel") }}
            </NButton>
            <NTooltip :disabled="!errorMessage">
              <template #trigger>
                <NButton
                  v-if="!readonly"
                  :disabled="isSubmitDisabled"
                  type="primary"
                  @click.prevent="() => $emit('apply', maskingAlgorithm)"
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
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { TrashIcon } from "lucide-vue-next";
import {
  NButton,
  NInput,
  NInputNumber,
  NRadio,
  NRadioGroup,
  NTooltip,
} from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import RequiredStar from "@/components/RequiredStar.vue";
import type { RadioGridOption } from "@/components/v2";
import {
  Drawer,
  DrawerContent,
  MiniActionButton,
  RadioGrid,
} from "@/components/v2";
import type {
  Algorithm,
  Algorithm_InnerOuterMask,
  Algorithm_FullMask as FullMask,
  Algorithm_MD5Mask as MD5Mask,
  Algorithm_RangeMask as RangeMask,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  Algorithm_InnerOuterMask_MaskType,
  Algorithm_InnerOuterMaskSchema,
  Algorithm_RangeMask_SliceSchema,
  AlgorithmSchema,
  Algorithm_FullMaskSchema as FullMaskSchema,
  Algorithm_MD5MaskSchema as MD5MaskSchema,
  Algorithm_RangeMaskSchema as RangeMaskSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import type { MaskingType } from "./utils";
import { getMaskingType } from "./utils";

interface MaskingTypeOption extends RadioGridOption<MaskingType> {
  value: MaskingType;
  label: string;
}

interface LocalState {
  processing: boolean;
  maskingType: MaskingType;
  fullMask: FullMask;
  rangeMask: RangeMask;
  md5Mask: MD5Mask;
  innerOuterMask: Algorithm_InnerOuterMask;
}

const props = defineProps<{
  show: boolean;
  readonly: boolean;
  algorithm?: Algorithm;
}>();

defineEmits<{
  (event: "dismiss"): void;
  (event: "apply", algorithm: Algorithm): void;
}>();

const defaultRangeMask = computed(() =>
  create(RangeMaskSchema, {
    slices: [
      create(Algorithm_RangeMask_SliceSchema, {
        start: 0,
        end: 1,
        substitution: "*",
      }),
    ],
  })
);

const defaultInnerOuterMask = computed(() =>
  create(Algorithm_InnerOuterMaskSchema, {
    prefixLen: 0,
    suffixLen: 0,
    type: Algorithm_InnerOuterMask_MaskType.INNER,
    substitution: "*",
  })
);

const state = reactive<LocalState>({
  processing: false,
  maskingType: "full-mask",
  fullMask: create(FullMaskSchema, {}),
  rangeMask: cloneDeep(defaultRangeMask.value),
  md5Mask: create(MD5MaskSchema, {}),
  innerOuterMask: cloneDeep(defaultInnerOuterMask.value),
});

const { t } = useI18n();

const maskingTypeList = computed((): MaskingTypeOption[] => [
  {
    value: "full-mask",
    label: t("settings.sensitive-data.algorithms.full-mask.self"),
  },
  {
    value: "range-mask",
    label: t("settings.sensitive-data.algorithms.range-mask.self"),
  },
  {
    value: "md5-mask",
    label: t("settings.sensitive-data.algorithms.md5-mask.self"),
  },
  {
    value: "inner-outer-mask",
    label: t("settings.sensitive-data.algorithms.inner-outer-mask.self"),
  },
]);

watch(
  () => props.algorithm,
  (algorithm) => {
    state.maskingType = getMaskingType(algorithm) ?? "full-mask";
    state.fullMask =
      algorithm?.mask?.case === "fullMask"
        ? algorithm.mask.value
        : create(FullMaskSchema, {});
    state.rangeMask =
      algorithm?.mask?.case === "rangeMask"
        ? algorithm.mask.value
        : cloneDeep(defaultRangeMask.value);
    state.md5Mask =
      algorithm?.mask?.case === "md5Mask"
        ? algorithm.mask.value
        : create(MD5MaskSchema, {});
    state.innerOuterMask =
      algorithm?.mask?.case === "innerOuterMask"
        ? algorithm.mask.value
        : cloneDeep(defaultInnerOuterMask.value);
  }
);

const maskingAlgorithm = computed((): Algorithm => {
  switch (state.maskingType) {
    case "full-mask":
      return create(AlgorithmSchema, {
        mask: {
          case: "fullMask",
          value: state.fullMask,
        },
      });
    case "range-mask":
      return create(AlgorithmSchema, {
        mask: {
          case: "rangeMask",
          value: state.rangeMask,
        },
      });
    case "md5-mask":
      return create(AlgorithmSchema, {
        mask: {
          case: "md5Mask",
          value: state.md5Mask,
        },
      });
    case "inner-outer-mask":
      return create(AlgorithmSchema, {
        mask: {
          case: "innerOuterMask",
          value: state.innerOuterMask,
        },
      });
    default:
      return create(AlgorithmSchema, {
        mask: {
          case: "fullMask",
          value: create(FullMaskSchema, {}),
        },
      });
  }
});

const rangeMaskErrorMessage = computed(() => {
  if (state.rangeMask.slices.length === 0) {
    return t("settings.sensitive-data.algorithms.error.slice-required");
  }
  for (let i = 0; i < state.rangeMask.slices.length; i++) {
    const slice = state.rangeMask.slices[i];
    if (Number.isNaN(slice.start) || Number.isNaN(slice.end)) {
      return t("settings.sensitive-data.algorithms.error.slice-invalid-number");
    }

    for (let j = 0; j < i; j++) {
      const pre = state.rangeMask.slices[j];
      if (slice.start >= pre.end || pre.start >= slice.end) {
        continue;
      }
      return t("settings.sensitive-data.algorithms.error.slice-overlap");
    }

    if (!slice.substitution) {
      return t(
        "settings.sensitive-data.algorithms.error.substitution-required"
      );
    }
    if (slice.substitution.length > 16) {
      return t("settings.sensitive-data.algorithms.error.substitution-length");
    }
  }
  return "";
});

const errorMessage = computed(() => {
  switch (state.maskingType) {
    case "full-mask":
      if (!state.fullMask.substitution) {
        return t(
          "settings.sensitive-data.algorithms.error.substitution-required"
        );
      }
      if (state.fullMask.substitution.length > 16) {
        return t(
          "settings.sensitive-data.algorithms.error.substitution-length"
        );
      }
      return "";
    case "md5-mask":
      if (!state.md5Mask.salt) {
        return t("settings.sensitive-data.algorithms.error.salt-required");
      }
      return "";
    case "range-mask":
      return rangeMaskErrorMessage.value;
    case "inner-outer-mask":
      if (!state.innerOuterMask.substitution) {
        return t(
          "settings.sensitive-data.algorithms.error.substitution-required"
        );
      }
      if (state.innerOuterMask.substitution.length > 16) {
        return t(
          "settings.sensitive-data.algorithms.error.substitution-length"
        );
      }
      return "";
  }
  return "";
});

const isSubmitDisabled = computed(() => {
  if (props.readonly || state.processing) {
    return true;
  }

  return !!errorMessage.value;
});

const onMaskingTypeChange = (maskingType: MaskingType) => {
  if (state.processing || props.readonly) {
    return;
  }
  switch (maskingType) {
    case "full-mask":
      state.fullMask = create(FullMaskSchema, {});
      break;
    case "range-mask":
      state.rangeMask = cloneDeep(defaultRangeMask.value);
      break;
    case "md5-mask":
      state.md5Mask = create(MD5MaskSchema, {});
      break;
    case "inner-outer-mask":
      state.innerOuterMask = cloneDeep(defaultInnerOuterMask.value);
      break;
  }
  state.maskingType = maskingType;
};

const addSlice = () => {
  const last = state.rangeMask.slices[state.rangeMask.slices.length - 1];
  state.rangeMask.slices.push(
    create(Algorithm_RangeMask_SliceSchema, {
      start: (last?.start ?? -1) + 1,
      end: (last?.end ?? 0) + 1,
      substitution: "*",
    })
  );
};

const removeSlice = (i: number) => {
  state.rangeMask.slices.splice(i, 1);
};

const onSliceStartChange = (index: number, val: number | null) => {
  if (val === null || Number.isNaN(val)) {
    return;
  }
  const slice = state.rangeMask.slices[index];
  slice.start = val;
  if (slice.end <= slice.start) {
    slice.end = slice.start + 1;
  }
};

const onSliceEndChange = (index: number, val: number | null) => {
  if (val === null || Number.isNaN(val)) {
    return;
  }
  const slice = state.rangeMask.slices[index];
  slice.end = val;
};

const onPrefixChange = (val: number) => {
  if (val === null || Number.isNaN(val) || val < 0) {
    return;
  }
  state.innerOuterMask.prefixLen = val;
};

const onSuffixChange = (val: number) => {
  if (val === null || Number.isNaN(val) || val < 0) {
    return;
  }
  state.innerOuterMask.suffixLen = val;
};
</script>
