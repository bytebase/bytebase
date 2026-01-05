<template>
  <div class="w-full flex flex-col gap-y-4">
    <div class="flex justify-end">
      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="['bb.policies.update']"
      >
        <div class="flex items-center gap-x-2">
          <NButton
            :disabled="slotProps.disabled || !hasSensitiveDataFeature"
            @click="state.showTemplateDrawer = true"
          >
            {{ $t("settings.sensitive-data.semantic-types.use-predefined-type") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="slotProps.disabled || !hasSensitiveDataFeature"
            @click="onAdd"
          >
            <template #icon>
              <PlusIcon class="h-4 w-4" />
            </template>
            {{ $t("common.add") }}
          </NButton>
        </div>
      </PermissionGuardWrapper>
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
import { create } from "@bufbuild/protobuf";
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { featureToRef, pushNotification, useSettingV1Store } from "@/store";
import type { SemanticTypeSetting_SemanticType } from "@/types/proto-es/v1/setting_service_pb";
import {
  SemanticTypeSetting_SemanticTypeSchema,
  Setting_SettingName,
  SettingValueSchema as SettingSettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
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
const hasSensitiveDataFeature = featureToRef(PlanFeature.FEATURE_DATA_MASKING);

const semanticTypeSettingValue = computed(() => {
  const semanticTypeSetting = settingStore.getSettingByName(
    Setting_SettingName.SEMANTIC_TYPES
  );
  return semanticTypeSetting?.value?.value?.case === "semanticType"
    ? (semanticTypeSetting.value.value.value.types ?? [])
    : [];
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
    item: create(SemanticTypeSetting_SemanticTypeSchema, {
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
    name: Setting_SettingName.SEMANTIC_TYPES,
    value: create(SettingSettingValueSchema, {
      value: {
        case: "semanticType",
        value: {
          types: state.semanticItemList
            .filter((data) => data.mode === "NORMAL")
            .map((data) => data.item),
        },
      },
    }),
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
    name: Setting_SettingName.SEMANTIC_TYPES,
    value: create(SettingSettingValueSchema, {
      value: {
        case: "semanticType",
        value: {
          types: semanticItemList,
        },
      },
    }),
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
      Setting_SettingName.SEMANTIC_TYPES
    );
    const types =
      semanticTypeSetting?.value?.value?.case === "semanticType"
        ? (semanticTypeSetting.value.value.value.types ?? [])
        : [];
    const origin = types.find((s) => s.id === item.item.id);
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
    item: create(SemanticTypeSetting_SemanticTypeSchema, {
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
