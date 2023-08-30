<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="columnList"
    class="border"
    :row-clickable="rowClickable"
    @click-row="clickRow"
  >
    <template #item="{ item, row }: { item: SensitiveColumn, row: number }">
      <div v-if="rowSelectable" class="bb-grid-cell">
        <input
          type="checkbox"
          class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
          :checked="checkedColumnIndex.has(row)"
          @input.stop="
            toggleColumnChecked(
              row,
              ($event.target as HTMLInputElement).checked,
              $event
            )
          "
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
      <div class="bb-grid-cell gap-x-1">
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
      <div
        v-if="showOperation"
        class="bb-grid-cell justify-center !px-2 gap-x-1"
      >
        <button
          :disabled="!allowAdmin"
          class="w-5 h-5 p-0.5 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
          @click.stop="$emit('click', item, 'EDIT')"
        >
          <heroicons-outline:pencil class="w-4 h-4" />
        </button>
        <NPopconfirm @positive-click="$emit('click', item, 'DELETE')">
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
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { NPopconfirm } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, type BBGridColumn } from "@/bbkit";
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
}>();

const emit = defineEmits<{
  (
    event: "click",
    item: SensitiveColumn,
    action: "VIEW" | "DELETE" | "EDIT" | "SELECT"
  ): void;
  (event: "checked:update", list: SensitiveColumn[]): void;
}>();

const { t } = useI18n();
const checkedColumnIndex = ref<Set<number>>(new Set());

watch(
  () => checkedColumnIndex.value,
  (checkedIndexList) => {
    emit(
      "checked:update",
      [...checkedIndexList].map((i) => props.columnList[i])
    );
  }
);

const currentUserV1 = useCurrentUserV1();
const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});

const COLUMN_LIST = computed((): BBGridColumn[] => {
  const list: BBGridColumn[] = [
    {
      title: t("settings.sensitive-data.masking-level.self"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("database.column"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.table"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.database"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.instance"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.environment"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.project"),
      width: "minmax(auto, 1fr)",
    },
  ];
  if (props.showOperation) {
    list.push({
      title: t("common.operation"),
      width: "minmax(auto, 6rem)",
      class: "justify-center !px-2",
    });
  }
  if (props.rowSelectable) {
    list.unshift({
      title: "",
      width: "minmax(auto, 3rem)",
      // class: "w-[1%]",
    });
  }
  return list;
});

const clickRow = (
  item: SensitiveColumn,
  section: number,
  row: number,
  e: MouseEvent
) => {
  emit("click", item, "VIEW");
};

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
};
</script>
