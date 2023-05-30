<template>
  <div
    v-if="targetProject.tenantMode === TenantMode.TENANT_MODE_ENABLED"
    class="space-y-4 flex flex-col justify-center items-center"
  >
    <div v-for="key in PRESET_LABEL_KEYS" :key="key" class="w-64">
      <label class="textlabel capitalize">
        {{ hidePrefix(key) }}
        <span v-if="isRequiredLabel(key)" style="color: red">*</span>
      </label>

      <div class="flex flex-col space-y-1 w-64 mt-1">
        <NInput
          :value="getLabelValue(key)"
          :placeholder="getLabelPlaceholder(key)"
          @input="setLabelValue(key, $event)"
        />
      </div>

      <div v-if="isParsedLabel(key)" class="mt-2 textinfolabel">
        <i18n-t keypath="label.parsed-from-template">
          <template #name>{{ database.databaseName }}</template>
          <template #template>
            <code class="text-xs font-mono bg-control-bg">
              {{ targetProject.dbNameTemplate }}
            </code>
          </template>
        </i18n-t>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
export default {
  inheritAttrs: false,
};
</script>

<script lang="ts" setup>
import { computed, toRef, watch } from "vue";
import { capitalize } from "lodash-es";
import { useI18n } from "vue-i18n";
import { NInput } from "naive-ui";

import type { ComposedDatabase } from "@/types";
import {
  buildDatabaseNameRegExpByTemplate,
  hidePrefix,
  parseLabelListInTemplate,
  PRESET_LABEL_KEYS,
  PRESET_LABEL_KEY_PLACEHOLDERS,
} from "@/utils";
import { useProjectV1ByUID } from "@/store";
import { TenantMode } from "@/types/proto/v1/project_service";

const props = defineProps<{
  database: ComposedDatabase;
  labels: Record<string, string>;
  targetProjectId: string;
}>();
const emit = defineEmits<{
  (event: "update:labels", labels: Record<string, string>): void;
}>();

const { t } = useI18n();

const { project: targetProject } = useProjectV1ByUID(
  toRef(props, "targetProjectId")
);

const requiredLabelList = computed((): string[] => {
  const project = targetProject.value;
  if (!project.dbNameTemplate) return [];

  // for databases with dbNameTemplate, we need to parse required labels from its template
  return parseLabelListInTemplate(project.dbNameTemplate);
});

const dbNameMatchesTemplate = computed((): boolean => {
  const project = targetProject.value;
  if (!project.dbNameTemplate) {
    // no restrictions, because no template
    return true;
  }
  const regex = buildDatabaseNameRegExpByTemplate(project.dbNameTemplate);
  return regex.test(props.database.databaseName);
});

const isRequiredLabel = (key: string): boolean => {
  return requiredLabelList.value.includes(key);
};

const isParsedLabel = (key: string): boolean => {
  const parsed = labelsParsedFromTemplate.value;
  if (!parsed) return false;
  return key in parsed;
};

const labelsParsedFromTemplate = computed(() => {
  if (!dbNameMatchesTemplate.value) return undefined;

  const regex = buildDatabaseNameRegExpByTemplate(
    targetProject.value.dbNameTemplate
  );
  const match = props.database.name.match(regex);
  if (!match) return undefined;

  const parsedLabelList: Record<string, string> = {};
  PRESET_LABEL_KEY_PLACEHOLDERS.forEach(([placeholder, key]) => {
    const value = match.groups?.[placeholder];
    if (value) {
      parsedLabelList[key] = value;
    }
  });

  return parsedLabelList;
});

watch(labelsParsedFromTemplate, (labels) => {
  if (!labels) return;
  for (const key in labels) {
    setLabelValue(key, labels[key]);
  }
});

const getLabelPlaceholder = (key: string): string => {
  // provide "Input Tenant" if Tenant is optional
  // provide "Input {{TENANT}}" if Tenant is required in the template
  key = isRequiredLabel(key)
    ? `{{${hidePrefix(key).toUpperCase()}}}`
    : capitalize(hidePrefix(key));
  return t("create-db.input-label-value", { key });
};

const getLabelValue = (key: string) => {
  return props.labels[key] ?? "";
};

const setLabelValue = (key: string, value: string) => {
  const labels = { ...props.labels };
  if (value) {
    labels[key] = value;
  } else {
    delete labels[key];
  }
  emit("update:labels", labels);
};
</script>
