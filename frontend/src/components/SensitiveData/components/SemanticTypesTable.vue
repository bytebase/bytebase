<template>
  <BBTable
    ref="tableRef"
    :column-list="tableHeaderList"
    :data-source="semanticItemList"
    :show-header="true"
    :custom-header="true"
    :left-bordered="true"
    :right-bordered="true"
    :top-bordered="true"
    :bottom-bordered="true"
    :compact-section="true"
    :row-clickable="rowClickable"
    @click-row="(section: number, index: number, _) => $emit('on-select', semanticItemList[index].item.id)"
  >
    <template #header>
      <BBTableHeaderCell
        v-for="header in tableHeaderList"
        :key="header.title"
        :title="header.title"
      />
    </template>
    <template #body="{ rowData, row }: { rowData: SemanticItem, row: number }">
      <BBTableCell class="bb-grid-cell">
        <h3 v-if="rowData.mode === 'NORMAL'">
          {{ rowData.item.title }}
        </h3>
        <NInput
          v-else
          :value="rowData.item.title"
          size="small"
          type="text"
          :placeholder="
            $t('settings.sensitive-data.semantic-types.table.semantic-type')
          "
          @input="(val: string) => onInput(row, (data) => data.item.title = val)"
        />
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <h3 v-if="rowData.mode === 'NORMAL'">
          {{ rowData.item.description }}
        </h3>
        <NInput
          v-else
          :value="rowData.item.description"
          size="small"
          type="text"
          :placeholder="
            $t('settings.sensitive-data.semantic-types.table.description')
          "
          @input="(val: string) => onInput(row, (data) => data.item.description = val)"
        />
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <h3 v-if="rowData.mode === 'NORMAL'">
          {{
            getAlgorithmById(rowData.item.fullMaskAlgorithmId)?.label ??
            $t("settings.sensitive-data.algorithms.default")
          }}
        </h3>
        <NSelect
          v-else
          :value="rowData.item.fullMaskAlgorithmId"
          clearable
          :options="algorithmList"
          :consistent-menu-width="false"
          :placeholder="$t('custom-approval.risk-rule.condition.select-value')"
          size="small"
          :fallback-option="(_: string) => ({ label: $t('settings.sensitive-data.algorithms.default'), value: '' })"
          style="min-width: 7rem; width: auto; overflow-x: hidden"
          @update:value="(val: string) => onInput(row, (data) => data.item.fullMaskAlgorithmId = val)"
        />
      </BBTableCell>
      <BBTableCell class="bb-grid-cell">
        <h3 v-if="rowData.mode === 'NORMAL'">
          {{
            getAlgorithmById(rowData.item.partialMaskAlgorithmId)?.label ??
            $t("settings.sensitive-data.algorithms.default")
          }}
        </h3>
        <NSelect
          v-else
          :value="rowData.item.partialMaskAlgorithmId"
          clearable
          :options="algorithmList"
          :consistent-menu-width="false"
          :placeholder="$t('custom-approval.risk-rule.condition.select-value')"
          :fallback-option="(_: string) => ({ label: $t('settings.sensitive-data.algorithms.default'), value: '' })"
          size="small"
          style="min-width: 7rem; width: auto; overflow-x: hidden"
          @update:value="(val: string) => onInput(row, (data) => data.item.partialMaskAlgorithmId = val)"
        />
      </BBTableCell>
      <BBTableCell v-if="!props.readonly" class="bb-grid-cell w-6">
        <div class="flex justify-end items-center space-x-2">
          <NPopconfirm
            v-if="rowData.mode === 'EDIT'"
            @positive-click="$emit('on-remove', row)"
          >
            <template #trigger>
              <button
                class="p-1 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                @click.stop=""
              >
                <heroicons-outline:trash class="w-4 h-4" />
              </button>
            </template>

            <div class="whitespace-nowrap">
              {{ $t("settings.sensitive-data.semantic-types.table.delete") }}
            </div>
          </NPopconfirm>

          <NButton
            v-if="rowData.mode !== 'NORMAL'"
            size="small"
            @click="$emit('on-cancel', row)"
          >
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            v-if="rowData.mode !== 'NORMAL'"
            type="primary"
            :disabled="isConfirmDisabled(rowData)"
            size="small"
            @click.stop="$emit('on-confirm', row)"
          >
            {{ $t("common.confirm") }}
          </NButton>

          <button
            v-if="rowData.mode === 'NORMAL'"
            class="p-1 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
            @click.stop="rowData.mode = 'EDIT'"
          >
            <heroicons-outline:pencil class="w-4 h-4" />
          </button>
        </div>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { NPopconfirm, NButton, NSelect, NInput } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn } from "@/bbkit/types";
import { useSettingV1Store } from "@/store";
import { SemanticTypeSetting_SemanticType } from "@/types/proto/v1/setting_service";

type SemanticItemMode = "NORMAL" | "CREATE" | "EDIT";

export interface SemanticItem {
  mode: SemanticItemMode;
  dirty: boolean;
  item: SemanticTypeSetting_SemanticType;
}

const props = defineProps<{
  readonly: boolean;
  semanticItemList: SemanticItem[];
  rowClickable: boolean;
}>();

defineEmits<{
  (event: "on-select", id: string): void;
  (event: "on-remove", index: number): void;
  (event: "on-cancel", index: number): void;
  (event: "on-confirm", index: number): void;
}>();

const { t } = useI18n();
const settingStore = useSettingV1Store();

const tableHeaderList = computed(() => {
  const list: BBTableColumn[] = [
    {
      title: t("settings.sensitive-data.semantic-types.table.semantic-type"),
    },
    {
      title: t("settings.sensitive-data.semantic-types.table.description"),
    },
    {
      title: t(
        "settings.sensitive-data.semantic-types.table.full-masking-algorithm"
      ),
    },
    {
      title: t(
        "settings.sensitive-data.semantic-types.table.partial-masking-algorithm"
      ),
    },
  ];
  if (!props.readonly) {
    // operation.
    list.push({
      title: "",
    });
  }
  return list;
});

const onInput = (index: number, callback: (item: SemanticItem) => void) => {
  const item = props.semanticItemList[index];
  if (!item) {
    return;
  }
  callback(item);
  item.dirty = true;
};

const algorithmList = computed((): SelectOption[] => {
  return (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  ).map((algorithm) => ({
    label: algorithm.title,
    value: algorithm.id,
  }));
});

const getAlgorithmById = (algorithmId: string) => {
  return algorithmList.value.find((a) => a.value === algorithmId);
};

const isConfirmDisabled = (data: SemanticItem): boolean => {
  if (!data.item.title) {
    return true;
  }
  if (data.mode === "EDIT" && !data.dirty) {
    return true;
  }
  return false;
};
</script>
