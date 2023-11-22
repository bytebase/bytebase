<template>
  <div class="w-full space-y-4">
    <div
      class="flex flex-col lg:flex-row items-start lg:items-center justify-between space-y-2 lg:space-x-2"
    >
      <div class="textinfolabel">
        {{ $t("settings.sensitive-data.semantic-types.label") }}
      </div>
      <div class="flex items-center justify-end space-x-2">
        <NButton
          :disabled="!hasPermission || !hasSensitiveDataFeature"
          @click="state.showTemplateDrawer = true"
        >
          {{ $t("settings.sensitive-data.semantic-types.add-from-template") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!hasPermission || !hasSensitiveDataFeature"
          @click="onAdd"
        >
          {{ $t("settings.sensitive-data.semantic-types.add-type") }}
        </NButton>
      </div>
    </div>
    <div class="space-y-5 divide-y-2 pb-10 divide-gray-100">
      <SemanticTypesTable
        :readonly="!hasPermission || !hasSensitiveDataFeature"
        :row-clickable="false"
        :semantic-item-list="state.semanticItemList"
        @cancel="onCancel"
        @remove="onRemove"
        @confirm="onConfirm"
      />
    </div>
  </div>
  <SemanticTemplateDrawer
    :show="state.showTemplateDrawer"
    @apply="onTemplateApply"
    @dismiss="state.showTemplateDrawer = false"
  />
</template>
<script lang="ts" setup>
import { NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  useSettingV1Store,
} from "@/store";
import { SemanticTypeSetting_SemanticType } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";
import SemanticTemplateDrawer from "./components/SemanticTemplateDrawer.vue";
import SemanticTypesTable, {
  SemanticItem,
} from "./components/SemanticTypesTable.vue";

interface LocalState {
  semanticItemList: SemanticItem[];
  processing: boolean;
  showTemplateDrawer: boolean;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  semanticItemList: [],
  processing: false,
  showTemplateDrawer: false,
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

const semanticTypeSettingValue = computed(() => {
  const semanticTypeSetting = settingStore.getSettingByName(
    "bb.workspace.semantic-types"
  );
  return semanticTypeSetting?.value?.semanticTypeSettingValue?.types ?? [];
});

onMounted(async () => {
  state.semanticItemList = semanticTypeSettingValue.value.map(
    (semanticType) => {
      return {
        dirty: false,
        item: semanticType,
        mode: "NORMAL",
      };
    }
  );
});

const onAdd = () => {
  state.semanticItemList.push({
    mode: "CREATE",
    dirty: false,
    item: SemanticTypeSetting_SemanticType.fromJSON({
      id: uuidv4(),
      fullMaskAlgorithmId: "",
      partialMaskAlgorithmId: "",
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

  await settingStore.upsertSetting({
    name: "bb.workspace.semantic-types",
    value: {
      semanticTypeSettingValue: {
        types: state.semanticItemList.map((data) => data.item),
      },
    },
  });

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

  await onUpsert(
    state.semanticItemList.map((data) => data.item),
    t(`common.${item.mode === "CREATE" ? "created" : "updated"}`)
  );
};

const onUpsert = async (
  semanticItemList: SemanticTypeSetting_SemanticType[],
  notification: string
) => {
  await settingStore.upsertSetting({
    name: "bb.workspace.semantic-types",
    value: {
      semanticTypeSettingValue: {
        types: semanticItemList,
      },
    },
  });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: notification,
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
      semanticTypeSetting?.value?.semanticTypeSettingValue?.types ?? []
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

const onTemplateApply = async (template: SemanticTypeSetting_SemanticType) => {
  const semanticItem: SemanticItem = {
    dirty: false,
    mode: "NORMAL",
    item: SemanticTypeSetting_SemanticType.fromPartial({
      ...template,
      id: uuidv4(),
    }),
  };
  state.semanticItemList.push(semanticItem);
  await onUpsert(
    [...semanticTypeSettingValue.value, semanticItem.item],
    t("common.created")
  );
};
</script>
