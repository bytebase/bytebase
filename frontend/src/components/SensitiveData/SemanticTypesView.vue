<template>
  <div class="w-full space-y-4">
    <div class="flex items-center justify-end space-x-2">
      <NButton
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="state.showTemplateDrawer = true"
      >
        {{ $t("settings.sensitive-data.semantic-types.use-predefined-type") }}
      </NButton>
      <NButton
        type="primary"
        :disabled="!hasPermission || !hasSensitiveDataFeature"
        @click="onAdd"
      >
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        {{ $t("common.add") }}
      </NButton>
    </div>
    <div class="textinfolabel">
      {{ $t("settings.sensitive-data.semantic-types.label") }}
    </div>
    <SemanticTypesTable
      :readonly="!hasPermission || !hasSensitiveDataFeature"
      :row-clickable="false"
      :semantic-item-list="state.semanticItemList"
      @cancel="onCancel"
      @remove="onRemove"
      @confirm="onConfirm"
    />
  </div>
  <SemanticTemplateDrawer
    :show="state.showTemplateDrawer"
    @apply="onTemplateApply"
    @dismiss="state.showTemplateDrawer = false"
  />
</template>
<script lang="ts" setup>
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { featureToRef, pushNotification, useSettingV1Store } from "@/store";
import { SemanticTypeSetting_SemanticType } from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import SemanticTemplateDrawer from "./components/SemanticTemplateDrawer.vue";
import type { SemanticItem } from "./components/SemanticTypesTable.vue";
import SemanticTypesTable from "./components/SemanticTypesTable.vue";

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
const hasPermission = computed(() => {
  return hasWorkspacePermissionV2("bb.policies.update");
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
    item: SemanticTypeSetting_SemanticType.fromPartial({
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

  await settingStore.upsertSetting({
    name: "bb.workspace.semantic-types",
    value: {
      semanticTypeSettingValue: {
        types: state.semanticItemList
          .filter((data) => data.mode === "NORMAL")
          .map((data) => data.item),
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
    state.semanticItemList
      .filter((data) => data.mode === "NORMAL")
      .map((data) => data.item),
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
  if (state.semanticItemList.find((item) => item.item.id === template.id)) {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: t(
        "settings.sensitive-data.semantic-types.template.duplicate-warning",
        {
          title: template.title,
        }
      ),
    });
    return;
  }
  const semanticItem: SemanticItem = {
    dirty: false,
    mode: "NORMAL",
    item: SemanticTypeSetting_SemanticType.fromPartial({
      ...template,
    }),
  };
  state.semanticItemList.push(semanticItem);
  await onUpsert(
    [...semanticTypeSettingValue.value, semanticItem.item],
    t("common.created")
  );
};
</script>
