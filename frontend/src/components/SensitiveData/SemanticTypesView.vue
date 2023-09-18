<template>
  <div class="w-full mt-4 space-y-4">
    <div class="flex items-center justify-end">
      <NButton
        type="primary"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="onAdd"
      >
        {{ $t("settings.sensitive-data.semantic-types.add-type") }}
      </NButton>
    </div>
    <div class="space-y-5 divide-y-2 pb-10 divide-gray-100">
      <BBTable
        ref="tableRef"
        :column-list="tableHeaderList"
        :data-source="state.semanticItemList"
        :show-header="true"
        :custom-header="false"
        :left-bordered="true"
        :right-bordered="true"
        :top-bordered="true"
        :bottom-bordered="true"
        :compact-section="true"
        :row-clickable="false"
      >
        <template
          #body="{ rowData, row }: { rowData: SemanticItem, row: number }"
        >
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
            <NSelect
              :value="rowData.item.fullMaskAlgorithmId"
              :options="algorithmList"
              :consistent-menu-width="false"
              :placeholder="
                $t('custom-approval.security-rule.condition.select-value')
              "
              :disabled="!hasPermission || !hasSensitiveDataFeature"
              size="small"
              style="min-width: 7rem; width: auto; overflow-x: hidden"
              @update:value="(val: string) => onInput(row, (data) => data.item.fullMaskAlgorithmId = val)"
            />
          </BBTableCell>
          <BBTableCell class="bb-grid-cell">
            <NSelect
              :value="rowData.item.partialMaskAlgorithmId"
              :options="algorithmList"
              :consistent-menu-width="false"
              :placeholder="
                $t('custom-approval.security-rule.condition.select-value')
              "
              :disabled="!hasPermission || !hasSensitiveDataFeature"
              size="small"
              style="min-width: 7rem; width: auto; overflow-x: hidden"
              @update:value="(val: string) => onInput(row, (data) => data.item.partialMaskAlgorithmId = val)"
            />
          </BBTableCell>
          <BBTableCell v-if="hasPermission" class="bb-grid-cell w-6">
            <div class="flex justify-end items-center space-x-2">
              <NPopconfirm
                v-if="rowData.mode === 'EDIT'"
                @positive-click="onRemove(row)"
              >
                <template #trigger>
                  <button
                    class="p-1 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                    @click.stop=""
                    :disabled="!hasPermission"
                  >
                    <heroicons-outline:trash class="w-4 h-4" />
                  </button>
                </template>

                <div class="whitespace-nowrap">
                  {{
                    $t("settings.sensitive-data.semantic-types.table.delete")
                  }}
                </div>
              </NPopconfirm>

              <NButton
                v-if="rowData.mode !== 'NORMAL'"
                size="small"
                @click="onCancel(row)"
              >
                {{ $t("common.cancel") }}
              </NButton>
              <NButton
                v-if="rowData.mode !== 'NORMAL'"
                type="primary"
                :disabled="isConfirmDisabled(rowData) || !hasPermission"
                size="small"
                @click.stop="onConfirm(row)"
              >
                {{ $t("common.confirm") }}
              </NButton>

              <button
                v-if="rowData.mode === 'NORMAL'"
                class="p-1 hover:bg-gray-300 rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                @click.stop="rowData.mode = 'EDIT'"
                :disabled="!hasPermission"
              >
                <heroicons-outline:pencil class="w-4 h-4" />
              </button>
            </div>
          </BBTableCell>
        </template>
      </BBTable>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { NPopconfirm, NButton, NSelect, NInput } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn } from "@/bbkit/types";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  useSettingV1Store,
} from "@/store";
import {
  SemanticTypesSetting_SemanticType,
  MaskingAlgorithmSetting_MaskingAlgorithm,
} from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";

type SemanticItemMode = "NORMAL" | "CREATE" | "EDIT";

interface SemanticItem {
  mode: SemanticItemMode;
  dirty: boolean;
  item: SemanticTypesSetting_SemanticType;
}

interface LocalState {
  semanticItemList: SemanticItem[];
  processing: boolean;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  semanticItemList: [],
  processing: false,
});

const settingStore = useSettingV1Store();
const currentUserV1 = useCurrentUserV1();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const algorithmList = computed((): SelectOption[] => {
  return (
    settingStore.getSettingByName("bb.workspace.masking-algorithms")?.value
      ?.maskingAlgorithmSettingValue?.algorithms ?? []
  ).map((algorithm) => ({
    label: algorithm.title,
    value: algorithm.id,
  }));
});

onMounted(async () => {
  const semanticTypeSetting = await settingStore.getOrFetchSettingByName(
    "bb.workspace.semantic-types",
    true
  );
  state.semanticItemList = (
    semanticTypeSetting.value?.semanticTypesSettingValue?.types ?? []
  ).map((semanticType) => {
    return {
      dirty: false,
      item: semanticType,
      mode: "NORMAL",
    };
  });
});

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
  if (hasPermission) {
    // operation.
    list.push({
      title: "",
    });
  }
  return list;
});

const onAdd = () => {
  state.semanticItemList.push({
    mode: "CREATE",
    dirty: false,
    item: SemanticTypesSetting_SemanticType.fromJSON({
      id: uuidv4(),
    }),
  });
};

const onRemove = async (index: number) => {
  const item = state.semanticItemList[index];
  if (!item) {
    return;
  }
  state.semanticItemList.splice(index, 1);
  if (item.mode === "CREATE") {
    return;
  }

  // TODO: call api
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.deleted"),
  });
};

const onConfirm = async (index: number) => {
  const item = state.semanticItemList[index];
  state.semanticItemList[index] = {
    ...item,
    dirty: false,
    mode: "NORMAL",
  };

  // TODO: call api

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(`common.${item.mode === "CREATE" ? "created" : "updated"}`),
  });
};

const onCancel = (index: number) => {
  const item = state.semanticItemList[index];

  if (item.mode === "CREATE") {
    state.semanticItemList.splice(index, 1);
  } else {
    const semanticTypeSetting = settingStore.getSettingByName(
      "bb.workspace.semantic-types"
    );
    const origin = (
      semanticTypeSetting?.value?.semanticTypesSettingValue?.types ?? []
    ).find((s) => s.id === item.item.id);
    if (!origin) {
      return;
    }
    state.semanticItemList[index] = {
      item: origin,
      mode: "NORMAL",
      dirty: false,
    };
  }
};

const onInput = (index: number, callback: (item: SemanticItem) => void) => {
  const item = state.semanticItemList[index];
  if (!item) {
    return;
  }
  callback(item);
  item.dirty = true;
};

const isConfirmDisabled = (data: SemanticItem): boolean => {
  if (
    !data.item.title ||
    !data.item.fullMaskAlgorithmId ||
    !data.item.partialMaskAlgorithmId
  ) {
    return true;
  }
  if (data.mode === "EDIT" && !data.dirty) {
    return true;
  }
  return false;
};
</script>
