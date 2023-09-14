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
              class=""
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
              class=""
              type="text"
              :placeholder="
                $t('settings.sensitive-data.semantic-types.table.description')
              "
              @input="(val: string) => onInput(row, (data) => data.item.description = val)"
            />
          </BBTableCell>
          <BBTableCell class="bb-grid-cell"> FULL MASKING </BBTableCell>
          <BBTableCell class="bb-grid-cell"> HALF MASKING </BBTableCell>
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
import { NPopconfirm, NButton, NInput } from "naive-ui";
import type { SelectOption } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, nextTick, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import type { BBTableColumn } from "@/bbkit/types";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
  MaskingRulePolicy_MaskingRule,
} from "@/types/proto/v1/org_policy_service";
import { SemanticCategorySetting_SemanticCategory } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";

type SemanticItemMode = "NORMAL" | "CREATE" | "EDIT";

interface SemanticItem {
  mode: SemanticItemMode;
  dirty: boolean;
  item: SemanticCategorySetting_SemanticCategory;
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
const policyStore = usePolicyV1Store();
const currentUserV1 = useCurrentUserV1();
const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sensitive-data",
    currentUserV1.value.userRole
  );
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

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
    item: SemanticCategorySetting_SemanticCategory.fromJSON({
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
    title: t("common.created"),
  });
};

const onCancel = (index: number) => {
  const item = state.semanticItemList[index];

  if (item.mode === "CREATE") {
    state.semanticItemList.splice(index, 1);
  } else {
    // TODO: reset row
    state.semanticItemList[index].mode = "NORMAL";
  }
};

const onInput = (index: number, callback: (item: SemanticItem) => void) => {
  const item = state.semanticItemList[index];
  if (!item) {
    return;
  }
  callback(item);
};

const onDropdownChange = async (
  index: number,
  callback: (item: SemanticItem) => void
) => {
  const item = state.semanticItemList[index];
  if (!item) {
    return;
  }
  callback(item);
  if (item.mode === "CREATE") {
    return;
  }

  // TODO: call api

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const isConfirmDisabled = (data: SemanticItem): boolean => {
  if (!data.item.title) {
    return true;
  }
  return false;
};
</script>
