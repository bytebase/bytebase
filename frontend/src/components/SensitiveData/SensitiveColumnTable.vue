<template>
  <BBTable
    ref="tableRef"
    :column-list="tableHeaderList"
    :section-data-source="datasource"
    :show-header="true"
    :custom-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :top-bordered="true"
    :bottom-bordered="true"
    :compact-section="true"
    :row-clickable="rowClickable"
    @click-row="clickTableRow"
  >
    <template #header>
      <th
        v-for="(column, index) in tableHeaderList"
        :key="index"
        scope="col"
        class="pl-2 py-2 text-left text-xs font-medium text-gray-500 tracking-wider capitalize"
        :class="[column.center && 'text-center pr-2']"
      >
        <template v-if="index === 0 && rowSelectable">
          <input
            v-if="columnList.length > 0"
            type="checkbox"
            class="ml-2 h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
            :checked="allSelectionState.checked"
            :indeterminate="allSelectionState.indeterminate"
            :disabled="false"
            @input="selectAll(($event.target as HTMLInputElement).checked)"
          />
        </template>
        <template v-else>{{ $t(column.title) }}</template>
      </th>
    </template>
    <template
      #body="{ rowData: item, row }: { rowData: SensitiveColumn, row: number }"
    >
      <BBTableCell v-if="rowSelectable" class="w-[1%]">
        <!-- width: 1% means as narrow as possible -->
        <input
          type="checkbox"
          class="ml-2 h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          :checked="checkedColumnIndex.has(row)"
          @click.stop=""
          @input="
            toggleColumnChecked(
              row,
              ($event.target as HTMLInputElement).checked,
              $event
            )
          "
        />
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{ getMaskingLevelText(item.maskData.maskingLevel) }}
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{ item.maskData.column }}
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        {{
          item.maskData.schema
            ? `${item.maskData.schema}.${item.maskData.table}`
            : item.maskData.table
        }}
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <DatabaseV1Name :database="item.database" :link="false" />
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <InstanceV1Name
          :instance="item.database.instanceEntity"
          :link="false"
        />
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <EnvironmentV1Name
          :environment="item.database.effectiveEnvironmentEntity"
          :link="false"
        />
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <ProjectV1Name :project="item.database.projectEntity" :link="false" />
      </BBTableCell>
      <BBTableCell
        v-if="showOperation"
        class="bb-grid-cell justify-center !px-2 space-x-2"
      >
        <button
          :disabled="!allowAdmin"
          class="w-5 h-5 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
          @click.stop="$emit('click', item, row, 'EDIT')"
        >
          <heroicons-outline:pencil class="w-4 h-4" />
        </button>
        <NPopconfirm @positive-click="$emit('click', item, row, 'DELETE')">
          <template #trigger>
            <button
              :disabled="!allowAdmin"
              class="w-5 h-5 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
              @click.stop=""
            >
              <heroicons-outline:trash />
            </button>
          </template>

          <div class="whitespace-nowrap">
            {{ $t("settings.sensitive-data.remove-sensitive-column-tips") }}
          </div>
        </NPopconfirm>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { NPopconfirm } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn, BBTableSectionDataSource } from "@/bbkit/types";
import {
  DatabaseV1Name,
  EnvironmentV1Name,
  InstanceV1Name,
  ProjectV1Name,
} from "@/components/v2";
import { useCurrentUserV1 } from "@/store";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import { hasWorkspacePermissionV1 } from "@/utils";
import { SensitiveColumn } from "./types";

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
    action: "VIEW" | "DELETE" | "EDIT" | "SELECT"
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

const currentUserV1 = useCurrentUserV1();
const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});

const getMaskingLevelText = (maskingLevel: MaskingLevel) => {
  let level = maskingLevelToJSON(maskingLevel);
  if (maskingLevel === MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
    level = maskingLevelToJSON(MaskingLevel.FULL);
  }
  return t(`settings.sensitive-data.masking-level.${level.toLowerCase()}`);
};

const toggleColumnChecked = (index: number, on: boolean, e: Event) => {
  e.preventDefault();
  e.stopPropagation();

  if (on) {
    checkedColumnIndex.value.add(index);
  } else {
    checkedColumnIndex.value.delete(index);
  }

  emit("checked:update", [...checkedColumnIndex.value]);
};

const tableHeaderList = computed(() => {
  const list: BBTableColumn[] = [
    {
      title: t("settings.sensitive-data.masking-level.self"),
    },
    {
      title: t("database.column"),
    },
    {
      title: t("common.table"),
    },
    {
      title: t("common.database"),
    },
    {
      title: t("common.instance"),
    },
    {
      title: t("common.environment"),
    },
    {
      title: t("common.project"),
    },
  ];
  if (props.showOperation) {
    list.push({
      title: t("common.operation"),
    });
  }
  if (props.rowSelectable) {
    list.unshift({
      title: "",
    });
  }
  return list;
});

const datasource = computed((): BBTableSectionDataSource<SensitiveColumn>[] => {
  return [
    {
      title: "",
      list: props.columnList,
    },
  ];
});

const allSelectionState = computed(() => {
  const checked = checkedColumnIndex.value.size === props.columnList.length;
  const indeterminate =
    !checked &&
    props.columnList.some((_, i) => checkedColumnIndex.value.has(i));

  return {
    checked,
    indeterminate,
  };
});

const selectAll = (check: boolean): void => {
  if (!check) {
    checkedColumnIndex.value = new Set([]);
  } else {
    checkedColumnIndex.value = new Set(props.columnList.map((_, i) => i));
  }
  emit("checked:update", [...checkedColumnIndex.value]);
};

const clickTableRow = function (_: number, row: number) {
  emit("click", props.columnList[row], row, "VIEW");
};
</script>
