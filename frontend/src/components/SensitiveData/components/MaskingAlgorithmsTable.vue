<template>
  <BBGrid
    :column-list="columnList"
    :data-source="algorithmList"
    :row-clickable="rowClickable"
    class="border compact"
    @click-row="(item: Algorithm) => $emit('select', item.id)"
  >
    <template #item="{ item }: AlgorithmRow">
      <div class="bb-grid-cell">
        {{ item.title }}
      </div>
      <div class="bb-grid-cell">
        {{ item.description }}
      </div>
      <div class="bb-grid-cell">
        {{ getAlgorithmMaskingType(item) }}
      </div>
      <div class="bb-grid-cell justify-end">
        <template v-if="item.id">
          <MiniActionButton @click.stop="$emit('edit', item)">
            <PencilIcon class="w-4 h-4" />
          </MiniActionButton>
          <NPopconfirm v-if="!readonly" @positive-click="onRemove(item.id)">
            <template #trigger>
              <MiniActionButton tag="div" @click.stop="">
                <TrashIcon class="w-4 h-4" />
              </MiniActionButton>
            </template>
            <div class="whitespace-nowrap">
              {{ $t("settings.sensitive-data.algorithms.table.delete") }}
            </div>
          </NPopconfirm>
        </template>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { PencilIcon, TrashIcon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import type { BBGridColumn, BBGridRow } from "@/bbkit/types";
import { MiniActionButton } from "@/components/v2";
import { useSettingV1Store, pushNotification } from "@/store";
import { MaskingAlgorithmSetting_Algorithm as Algorithm } from "@/types/proto/v1/setting_service";
import { getMaskingType } from "./utils";

type AlgorithmRow = BBGridRow<Algorithm>;

defineProps<{
  readonly: boolean;
  rowClickable: boolean;
}>();

defineEmits<{
  (event: "select", id: string): void;
  (event: "edit", item: Algorithm): void;
}>();

const { t } = useI18n();
const settingStore = useSettingV1Store();

onMounted(async () => {
  await settingStore.getOrFetchSettingByName(
    "bb.workspace.masking-algorithm",
    true
  );
});

const columnList = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("settings.sensitive-data.algorithms.table.title"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("settings.sensitive-data.algorithms.table.description"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("settings.sensitive-data.algorithms.table.masking-type"),
      width: "minmax(min-content, auto)",
    },
    {
      // operations
      title: "",
      width: "minmax(min-content, auto)",
    },
  ];
  return columns;
});

const rawAlgorithmList = computed((): Algorithm[] => {
  return (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  );
});

const algorithmList = computed((): Algorithm[] => {
  return [
    Algorithm.fromPartial({
      title: t("settings.sensitive-data.algorithms.default"),
      description: t("settings.sensitive-data.algorithms.default-desc"),
      category: "MASK",
    }),
    ...rawAlgorithmList.value,
  ];
});

const getAlgorithmMaskingType = (algorithm: Algorithm) => {
  const maskingType = getMaskingType(algorithm);
  if (maskingType) {
    return t(`settings.sensitive-data.algorithms.${maskingType}.self`);
  }

  return t(
    `settings.sensitive-data.algorithms.${algorithm.category.toLowerCase()}`
  );
};

const onRemove = async (id: string) => {
  const index = rawAlgorithmList.value.findIndex((item) => item.id === id);
  if (index < 0) {
    return;
  }
  const newList = [
    ...rawAlgorithmList.value.slice(0, index),
    ...rawAlgorithmList.value.slice(index + 1),
  ];

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
    title: t("common.deleted"),
  });
};
</script>
