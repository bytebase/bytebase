<template>
  <BBTable
    ref="tableRef"
    :column-list="tableHeaderList"
    :data-source="algorithmList"
    :show-header="true"
    :custom-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :top-bordered="true"
    :bottom-bordered="true"
    :compact-section="true"
    :row-clickable="rowClickable"
    @click-row="(section: number, index: number, _) => $emit('on-select', algorithmList[index].id)"
  >
    <template #header>
      <BBTableHeaderCell
        v-for="header in tableHeaderList"
        :key="header.title"
        :title="header.title"
      />
    </template>
    <template
      #body="{
        rowData,
        row,
      }: {
        rowData: MaskingAlgorithmSetting_Algorithm,
        row: number,
      }"
    >
      <BBTableCell class="bb-grid-cell">
        <h3>
          {{ rowData.title }}
        </h3>
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <h3>
          {{ rowData.description }}
        </h3>
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <h3>
          {{ getAlgorithmMaskingType(rowData) }}
        </h3>
      </BBTableCell>
      <BBTableCell class="bb-grid-cell w-6">
        <div v-if="rowData.id" class="flex justify-end items-center space-x-2">
          <button
            class="p-1 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
            @click.stop="$emit('on-edit', rowData)"
          >
            <heroicons-outline:pencil class="w-4 h-4" />
          </button>

          <NPopconfirm v-if="!readonly" @positive-click="onRemove(row)">
            <template #trigger>
              <button
                class="p-1 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                @click.stop=""
              >
                <heroicons-outline:trash class="w-4 h-4" />
              </button>
            </template>

            <div class="whitespace-nowrap">
              {{ $t("settings.sensitive-data.algorithms.table.delete") }}
            </div>
          </NPopconfirm>
        </div>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { NPopconfirm } from "naive-ui";
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn } from "@/bbkit/types";
import { useSettingV1Store, pushNotification } from "@/store";
import { MaskingAlgorithmSetting_Algorithm } from "@/types/proto/v1/setting_service";
import { getMaskingType } from "./utils";

defineProps<{
  readonly: boolean;
  rowClickable: boolean;
}>();

defineEmits<{
  (event: "on-select", id: string): void;
  (event: "on-edit", item: MaskingAlgorithmSetting_Algorithm): void;
}>();

const { t } = useI18n();
const settingStore = useSettingV1Store();

onMounted(async () => {
  await settingStore.getOrFetchSettingByName(
    "bb.workspace.masking-algorithm",
    true
  );
});

const tableHeaderList = computed(() => {
  const list: BBTableColumn[] = [
    {
      title: t("settings.sensitive-data.algorithms.table.title"),
    },
    {
      title: t("settings.sensitive-data.algorithms.table.description"),
    },
    {
      title: t("settings.sensitive-data.algorithms.table.masking-type"),
    },
    {
      // operations
      title: "",
    },
  ];
  return list;
});

const rawAlgorithmList = computed((): MaskingAlgorithmSetting_Algorithm[] => {
  return (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  );
});

const algorithmList = computed((): MaskingAlgorithmSetting_Algorithm[] => {
  return [
    MaskingAlgorithmSetting_Algorithm.fromPartial({
      title: t("settings.sensitive-data.algorithms.default"),
      description: t("settings.sensitive-data.algorithms.default-desc"),
      category: "MASK",
    }),
    ...rawAlgorithmList.value,
  ];
});

const getAlgorithmMaskingType = (
  algorithm: MaskingAlgorithmSetting_Algorithm
) => {
  const maskingType = getMaskingType(algorithm);
  if (maskingType) {
    return t(`settings.sensitive-data.algorithms.${maskingType}.self`);
  }

  return t(
    `settings.sensitive-data.algorithms.${algorithm.category.toLowerCase()}`
  );
};

const onRemove = async (index: number) => {
  const item = rawAlgorithmList.value[index];
  if (!item) {
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
