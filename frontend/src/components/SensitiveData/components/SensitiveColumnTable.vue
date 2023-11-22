<template>
  <BBGrid
    :column-list="gridColumnList"
    :data-source="columnList"
    :row-clickable="rowClickable"
    :custom-header="true"
    class="border compact"
    @click-row="clickTableRow"
  >
    <template #header>
      <div role="table-row" class="bb-grid-row bb-grid-header-row group">
        <div
          v-for="(column, index) in gridColumnList"
          :key="index"
          role="table-cell"
          class="bb-grid-header-cell capitalize"
          :class="[column.class]"
        >
          <template v-if="index === 0 && rowSelectable">
            <NCheckbox
              v-if="columnList.length > 0"
              :checked="allSelectionState.checked"
              :indeterminate="allSelectionState.indeterminate"
              @update:checked="toggleSelectAll"
            />
          </template>
          <template v-else>{{ column.title }}</template>
        </div>
      </div>
    </template>
    <template #item="{ item, row }: SensitiveColumnRow">
      <div v-if="rowSelectable" class="bb-grid-cell" @click.stop.prevent>
        <NCheckbox
          :checked="checkedColumnIndex.has(row)"
          @update:checked="toggleColumnChecked(row, $event)"
        />
      </div>
      <div class="bb-grid-cell">
        {{ getMaskingLevelText(item.maskData.maskingLevel) }}
      </div>
      <div class="bb-grid-cell">
        {{ item.maskData.column }}
      </div>
      <div class="bb-grid-cell">
        {{
          item.maskData.schema
            ? `${item.maskData.schema}.${item.maskData.table}`
            : item.maskData.table
        }}
      </div>
      <div class="bb-grid-cell">
        <DatabaseV1Name :database="item.database" :link="false" />
      </div>
      <div class="bb-grid-cell">
        <InstanceV1Name
          :instance="item.database.instanceEntity"
          :link="false"
        />
      </div>
      <div class="bb-grid-cell">
        <EnvironmentV1Name
          :environment="item.database.effectiveEnvironmentEntity"
          :link="false"
        />
      </div>
      <div class="bb-grid-cell">
        <ProjectV1Name :project="item.database.projectEntity" :link="false" />
      </div>
      <div v-if="showOperation" class="bb-grid-cell" @click.stop.prevent>
        <MiniActionButton
          @click.stop.prevent="$emit('click', item, row, 'EDIT')"
        >
          <PencilIcon class="w-4 h-4" />
        </MiniActionButton>
        <NPopconfirm @positive-click="$emit('click', item, row, 'DELETE')">
          <template #trigger>
            <MiniActionButton tag="div" @click.stop.prevent>
              <TrashIcon class="w-4 h-4" />
            </MiniActionButton>
          </template>

          <div class="whitespace-nowrap">
            {{ $t("settings.sensitive-data.remove-sensitive-column-tips") }}
          </div>
        </NPopconfirm>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { TrashIcon } from "lucide-vue-next";
import { NCheckbox, NPopconfirm } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid } from "@/bbkit";
import type { BBGridColumn, BBGridRow } from "@/bbkit/types";
import {
  DatabaseV1Name,
  EnvironmentV1Name,
  InstanceV1Name,
  MiniActionButton,
  ProjectV1Name,
} from "@/components/v2";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import { SensitiveColumn } from "../types";

type SensitiveColumnRow = BBGridRow<SensitiveColumn>;

const props = defineProps<{
  showOperation: boolean;
  rowClickable: boolean;
  rowSelectable: boolean;
  columnList: SensitiveColumn[];
  checkedColumnIndexList: number[];
}>();

const emit = defineEmits<{
  (
    event: "click",
    item: SensitiveColumn,
    row: number,
    action: "VIEW" | "DELETE" | "EDIT"
  ): void;
  (event: "checked:update", list: number[]): void;
}>();

const { t } = useI18n();
const checkedColumnIndex = ref<Set<number>>(
  new Set(props.checkedColumnIndexList)
);

watch(
  () => props.columnList,
  () => (checkedColumnIndex.value = new Set()),
  { deep: true }
);
watch(
  () => props.checkedColumnIndexList,
  (val) => (checkedColumnIndex.value = new Set(val)),
  { deep: true }
);

const getMaskingLevelText = (maskingLevel: MaskingLevel) => {
  const level = maskingLevelToJSON(maskingLevel);
  return t(`settings.sensitive-data.masking-level.${level.toLowerCase()}`);
};

const toggleColumnChecked = (index: number, on: boolean) => {
  if (on) {
    checkedColumnIndex.value.add(index);
  } else {
    checkedColumnIndex.value.delete(index);
  }

  emit("checked:update", [...checkedColumnIndex.value]);
};

const gridColumnList = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("settings.sensitive-data.masking-level.self"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("database.column"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("common.table"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("common.database"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("common.instance"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("common.environment"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("common.project"),
      width: "minmax(min-content, auto)",
    },
  ];
  if (props.showOperation) {
    columns.push({
      title: t("common.operation"),
      width: "minmax(min-content, auto)",
    });
  }
  if (props.rowSelectable) {
    columns.unshift({
      title: "",
      width: "minmax(auto, 3rem)",
    });
  }

  return columns;
});

const allSelectionState = computed(() => {
  const checked =
    checkedColumnIndex.value.size > 0 &&
    checkedColumnIndex.value.size === props.columnList.length;
  const indeterminate =
    !checked &&
    props.columnList.some((_, i) => checkedColumnIndex.value.has(i));

  return {
    checked,
    indeterminate,
  };
});

const toggleSelectAll = (check: boolean): void => {
  if (!check) {
    checkedColumnIndex.value = new Set([]);
  } else {
    checkedColumnIndex.value = new Set(props.columnList.map((_, i) => i));
  }
  emit("checked:update", [...checkedColumnIndex.value]);
};

const clickTableRow = function (item: SensitiveColumn, _: number, row: number) {
  emit("click", item, row, "VIEW");
};
</script>
