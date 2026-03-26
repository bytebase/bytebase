<template>
  <div class="w-full flex flex-col gap-y-4">
    <div class="flex items-center justify-between">
      <div class="text-sm text-control-light">
        {{ $t("database.classification.description") }}
        <LearnMoreLink
          url="https://docs.bytebase.com/security/data-masking/data-classification?source=console"
          class="ml-1"
        />
      </div>

      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="['bb.settings.set']"
      >
        <div class="flex items-center justify-end gap-x-2 shrink-0">
          <template v-if="editing">
            <NButton
              :disabled="!editorDirty"
              @click="onRevert"
            >
              {{ $t("common.revert") }}
            </NButton>
            <NButton
              @click="onCancelEdit"
            >
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              type="primary"
              :disabled="!editorDirty || slotProps.disabled || !hasClassificationFeature"
              @click="onSave"
            >
              {{ $t("common.save") }}
            </NButton>
          </template>
          <template v-else>
            <BBButtonConfirm
              v-if="!emptyConfig"
              :tertiary="true"
              :text="false"
              :type="'DELETE'"
              :ok-text="$t('common.delete')"
              :require-confirm="true"
              :hide-icon="true"
              :button-text="$t('common.clear')"
              :disabled="slotProps.disabled || !hasClassificationFeature"
              @confirm="clearSetting"
            />
            <NButton
              type="primary"
              :disabled="slotProps.disabled || !hasClassificationFeature"
              @click="editing = true"
            >
              {{ $t("common.edit") }}
            </NButton>
          </template>
        </div>
      </PermissionGuardWrapper>
    </div>

    <template v-if="editing">
      <div class="textinfolabel space-y-1">
        <p>{{ $t("settings.sensitive-data.classification.guide-intro") }}</p>
        <ul class="list-disc list-inside">
          <li>{{ $t("settings.sensitive-data.classification.guide-levels") }}</li>
          <li>{{ $t("settings.sensitive-data.classification.guide-classification") }}</li>
          <li>{{ $t("settings.sensitive-data.classification.guide-hierarchy") }}</li>
        </ul>
      </div>
      <div class="border rounded-sm overflow-hidden" style="height: 50vh">
        <MonacoEditor
          :content="editorContent"
          :readonly="!allowEdit || !hasClassificationFeature"
          language="json"
          class="w-full h-full"
          @update:content="onEditorChange"
        />
      </div>
    </template>

    <div v-if="!editing && !emptyConfig" class="h-full">
      <ClassificationTree :classification-config="state.classification" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { head, isEmpty } from "lodash-es";
import { NButton } from "naive-ui";
import { v4 as uuidv4 } from "uuid";
import { computed, reactive, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirm } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { featureToRef, pushNotification, useSettingV1Store } from "@/store";
import type {
  DataClassificationSetting_DataClassificationConfig_DataClassification as DataClassification,
  DataClassificationSetting_DataClassificationConfig,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  DataClassificationSetting_DataClassificationConfig_DataClassificationSchema,
  DataClassificationSetting_DataClassificationConfig_LevelSchema,
  DataClassificationSetting_DataClassificationConfigSchema,
  DataClassificationSettingSchema,
  Setting_SettingName,
  SettingValueSchema as SettingSettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import LearnMoreLink from "../LearnMoreLink.vue";
import classificationExample from "./classification-example.json";
import ClassificationTree from "./components/ClassificationTree.vue";

interface ClassificationJSON {
  title: string;
  levels: { title: string; level: number }[];
  classification: {
    [key: string]: { id: string; title: string; level?: number };
  };
}

interface LocalState {
  classification: DataClassificationSetting_DataClassificationConfig;
}

const { t } = useI18n();
const settingStore = useSettingV1Store();

const formerConfig = computed(() => {
  const classification = head(settingStore.classification);
  return create(DataClassificationSetting_DataClassificationConfigSchema, {
    id: uuidv4(),
    title: classification?.title || "",
    levels: classification?.levels || [],
    classification: classification?.classification || {},
  });
});

const state = reactive<LocalState>({
  classification: create(
    DataClassificationSetting_DataClassificationConfigSchema,
    {
      id: uuidv4(),
      title: "",
      levels: [],
      classification: {},
    }
  ),
});

// Initialize state with formerConfig
watchEffect(async () => {
  await settingStore.getOrFetchSettingByName(
    Setting_SettingName.DATA_CLASSIFICATION
  );
  const config = formerConfig.value;
  Object.assign(state.classification, {
    id: config.id,
    title: config.title,
    levels: config.levels,
    classification: config.classification,
  });
});

const emptyConfig = computed(
  () => Object.keys(state.classification.classification).length === 0
);

const configToJSON = (
  config: DataClassificationSetting_DataClassificationConfig
): string => {
  const data: ClassificationJSON = {
    title: config.title,
    levels: config.levels.map((l) => ({
      title: l.title,
      level: l.level,
    })),
    classification: {},
  };
  for (const [key, val] of Object.entries(config.classification)) {
    const entry: ClassificationJSON["classification"][string] = {
      id: val.id,
      title: val.title,
    };
    if (val.level !== undefined && val.level !== 0) {
      entry.level = val.level;
    }
    data.classification[key] = entry;
  }
  return JSON.stringify(data, null, 2);
};

const savedContent = computed(() => {
  if (emptyConfig.value) {
    return JSON.stringify(classificationExample, null, 2);
  }
  return configToJSON(state.classification);
});

const editing = ref(false);
const editorContent = ref("");
const editorDirty = computed(() => {
  return editorContent.value !== savedContent.value;
});

watchEffect(() => {
  if (editing.value) return;
  editorContent.value = savedContent.value;
});

const onEditorChange = (content: string) => {
  editorContent.value = content;
};

const onRevert = () => {
  editorContent.value = savedContent.value;
};

const onCancelEdit = () => {
  editorContent.value = savedContent.value;
  editing.value = false;
};

const onSave = async () => {
  let data: ClassificationJSON;
  try {
    data = JSON.parse(editorContent.value);
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("settings.sensitive-data.classification.invalid-json"),
    });
    return;
  }

  if (isEmpty(data.classification) || Array.isArray(data.classification)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("settings.sensitive-data.classification.missing-classification"),
    });
    return;
  }
  if (Object.keys(data.classification).length === 0) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("settings.sensitive-data.classification.empty-classification"),
    });
    return;
  }
  if (!Array.isArray(data.levels) || data.levels.length === 0) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("settings.sensitive-data.classification.missing-levels"),
    });
    return;
  }

  const config = create(
    DataClassificationSetting_DataClassificationConfigSchema,
    {
      id: state.classification.id || uuidv4(),
      title: data.title || "",
      levels: data.levels.map((level) =>
        create(
          DataClassificationSetting_DataClassificationConfig_LevelSchema,
          level
        )
      ),
      classification: Object.values(data.classification).reduce(
        (map, item) => {
          map[item.id] = create(
            DataClassificationSetting_DataClassificationConfig_DataClassificationSchema,
            item
          );
          return map;
        },
        {} as { [key: string]: DataClassification }
      ),
    }
  );

  await upsertSetting([config]);
  editing.value = false;
};

const clearSetting = async () => {
  await upsertSetting([]);
};

const upsertSetting = async (
  configs: DataClassificationSetting_DataClassificationConfig[]
) => {
  await settingStore.upsertSetting({
    name: Setting_SettingName.DATA_CLASSIFICATION,
    value: create(SettingSettingValueSchema, {
      value: {
        case: "dataClassification",
        value: create(DataClassificationSettingSchema, {
          configs,
        }),
      },
    }),
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const allowEdit = computed(() => {
  return hasWorkspacePermissionV2("bb.settings.set");
});

const hasClassificationFeature = featureToRef(
  PlanFeature.FEATURE_DATA_CLASSIFICATION
);
</script>
