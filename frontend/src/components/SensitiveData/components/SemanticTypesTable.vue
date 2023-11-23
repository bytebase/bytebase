<template>
  <BBGrid
    :column-list="columnList"
    :data-source="semanticItemList"
    :row-clickable="rowClickable"
    class="border compact"
    @click-row="
      (item: SemanticItem) => $emit('select', item.item.id)
    "
  >
    <template #item="{ item, row }: SemanticItemRow">
      <div class="bb-grid-cell">
        <h3 v-if="item.mode === 'NORMAL'" class="break-normal">
          {{ item.item.title }}
        </h3>
        <NInput
          v-else
          :value="item.item.title"
          size="small"
          type="text"
          :placeholder="
            $t('settings.sensitive-data.semantic-types.table.semantic-type')
          "
          @input="(val: string) => onInput(row, (data) => data.item.title = val)"
        />
      </div>
      <div class="bb-grid-cell">
        <h3 v-if="item.mode === 'NORMAL'">
          {{ item.item.description }}
        </h3>
        <NInput
          v-else
          :value="item.item.description"
          size="small"
          type="text"
          :placeholder="
            $t('settings.sensitive-data.semantic-types.table.description')
          "
          @input="(val: string) => onInput(row, (data) => data.item.description = val)"
        />
      </div>
      <div class="bb-grid-cell">
        <h3 v-if="item.mode === 'NORMAL'">
          {{
            getAlgorithmById(item.item.fullMaskAlgorithmId)?.label ??
            $t("settings.sensitive-data.algorithms.default")
          }}
        </h3>
        <NSelect
          v-else
          :value="item.item.fullMaskAlgorithmId"
          :options="algorithmList"
          :consistent-menu-width="false"
          :placeholder="$t('settings.sensitive-data.algorithms.default')"
          :fallback-option="(_: string) => ({ label: $t('settings.sensitive-data.algorithms.default'), value: '' })"
          clearable
          size="small"
          style="min-width: 7rem; width: auto; overflow-x: hidden"
          @update:value="
            onInput(row, (data) => (data.item.fullMaskAlgorithmId = $event))
          "
        />
      </div>
      <div class="bb-grid-cell">
        <h3 v-if="item.mode === 'NORMAL'">
          {{
            getAlgorithmById(item.item.partialMaskAlgorithmId)?.label ??
            $t("settings.sensitive-data.algorithms.default")
          }}
        </h3>
        <NSelect
          v-else
          :value="item.item.partialMaskAlgorithmId"
          :options="algorithmList"
          :consistent-menu-width="false"
          :placeholder="$t('settings.sensitive-data.algorithms.default')"
          :fallback-option="(_: string) => ({ label: $t('settings.sensitive-data.algorithms.default'), value: '' })"
          clearable
          size="small"
          style="min-width: 7rem; width: auto; overflow-x: hidden"
          @update:value="
            onInput(row, (data) => (data.item.partialMaskAlgorithmId = $event))
          "
        />
      </div>
      <div v-if="!props.readonly" class="bb-grid-cell justify-end">
        <NPopconfirm
          v-if="item.mode === 'EDIT'"
          @positive-click="$emit('remove', row)"
        >
          <template #trigger>
            <MiniActionButton tag="div" @click.stop="">
              <TrashIcon class="w-4 h-4" />
            </MiniActionButton>
          </template>
          <div class="whitespace-nowrap">
            {{ $t("settings.sensitive-data.semantic-types.table.delete") }}
          </div>
        </NPopconfirm>

        <MiniActionButton
          v-if="item.mode !== 'NORMAL'"
          @click="$emit('cancel', row)"
        >
          <Undo2Icon class="w-4 h-4" />
        </MiniActionButton>
        <MiniActionButton
          v-if="item.mode !== 'NORMAL'"
          type="primary"
          :disabled="isConfirmDisabled(item)"
          @click.stop="$emit('confirm', row)"
        >
          <CheckIcon class="w-4 h-4" />
        </MiniActionButton>

        <MiniActionButton
          v-if="item.mode === 'NORMAL'"
          @click.stop="item.mode = 'EDIT'"
        >
          <PencilIcon class="w-4 h-4" />
        </MiniActionButton>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { CheckIcon, PencilIcon, TrashIcon, Undo2Icon } from "lucide-vue-next";
import { NPopconfirm, NSelect, NInput } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { BBGridColumn, BBGridRow } from "@/bbkit/types";
import { MiniActionButton } from "@/components/v2";
import { useSettingV1Store } from "@/store";
import { SemanticTypeSetting_SemanticType } from "@/types/proto/v1/setting_service";

type SemanticItemMode = "NORMAL" | "CREATE" | "EDIT";

export interface SemanticItem {
  mode: SemanticItemMode;
  dirty: boolean;
  item: SemanticTypeSetting_SemanticType;
}

type SemanticItemRow = BBGridRow<SemanticItem>;

const props = defineProps<{
  readonly: boolean;
  semanticItemList: SemanticItem[];
  rowClickable: boolean;
}>();

defineEmits<{
  (event: "select", id: string): void;
  (event: "remove", index: number): void;
  (event: "cancel", index: number): void;
  (event: "confirm", index: number): void;
}>();

const { t } = useI18n();
const settingStore = useSettingV1Store();

const columnList = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("settings.sensitive-data.semantic-types.table.semantic-type"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t("settings.sensitive-data.semantic-types.table.description"),
      width: "minmax(min-content, auto)",
    },
    {
      title: t(
        "settings.sensitive-data.semantic-types.table.full-masking-algorithm"
      ),
      width: "minmax(min-content, auto)",
    },
    {
      title: t(
        "settings.sensitive-data.semantic-types.table.partial-masking-algorithm"
      ),
      width: "minmax(min-content, auto)",
    },
  ];
  if (!props.readonly) {
    // operation.
    columns.push({
      title: "",
      width: "minmax(min-content, auto)",
    });
  }
  return columns;
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
  const list = (
    settingStore.getSettingByName("bb.workspace.masking-algorithm")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  ).map((algorithm) => ({
    label: algorithm.title,
    value: algorithm.id,
  }));

  list.unshift({
    label: t("settings.sensitive-data.algorithms.default"),
    value: "",
  });

  return list;
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
