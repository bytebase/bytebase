<template>
  <template v-if="!dbNameMatchesTemplate">
    <div class="textinfolabel" v-bind="$attrs">
      <i18n-t keypath="label.cannot-transfer-template-not-match">
        <template #name>{{ database.name }}</template>
        <template #template>
          <code class="text-sm font-mono bg-control-bg">
            {{ targetProject.dbNameTemplate }}
          </code>
        </template>
      </i18n-t>
    </div>
  </template>
  <template v-else>
    <div
      class="space-y-4 flex flex-col justify-center items-center"
      v-bind="$attrs"
    >
      <div v-for="label in availableLabelList" :key="label.id" class="w-64">
        <label class="textlabel capitalize">
          {{ hidePrefix(label.key) }}
          <span v-if="isRequiredLabel(label.key)" style="color: red">*</span>
        </label>

        <div class="flex flex-col space-y-1 w-64 mt-1">
          <select
            class="btn-select w-full disabled:cursor-not-allowed"
            :value="getLabelValue(label.key)"
            :disabled="isParsedLabel(label.key)"
            @change="(e: any) => setLabelValue(label.key, e.target.value)"
          >
            <option disabled>{{ getLabelPlaceholder(label.key) }}</option>
            <option
              v-for="value in getLabelValueList(label)"
              :key="value"
              :value="value"
            >
              {{ value === "" ? $t("label.empty-label-value") : value }}
            </option>
          </select>
        </div>

        <div v-if="isParsedLabel(label.key)" class="mt-2 textinfolabel">
          <i18n-t keypath="label.parsed-from-template">
            <template #name>{{ database.name }}</template>
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

  <slot name="buttons" :next="next" :valid="allowNext"></slot>
</template>

<script lang="ts">
export default {
  inheritAttrs: false,
};
</script>

<script lang="ts" setup>
import { computed, reactive, watchEffect, watch } from "vue";
import { capitalize, cloneDeep } from "lodash-es";
import {
  Database,
  DatabaseLabel,
  Label,
  LabelKeyType,
  LabelValueType,
  Project,
  ProjectId,
} from "../../types";
import {
  buildDatabaseNameRegExpByTemplate,
  isReservedLabel,
  parseLabelListInTemplate,
  hidePrefix,
} from "../../utils";
import { useI18n } from "vue-i18n";
import { useLabelStore, useProjectStore } from "@/store";
import { storeToRefs } from "pinia";

const props = defineProps<{
  database: Database;
  targetProjectId: ProjectId;
}>();

const emit = defineEmits<{
  (event: "next", labelList: DatabaseLabel[]): void;
}>();

const state = reactive({
  databaseLabelList: cloneDeep(props.database.labels),
});

const labelStore = useLabelStore();
const projectStore = useProjectStore();

const { t } = useI18n();

const targetProject = computed(() => {
  return projectStore.getProjectById(props.targetProjectId) as Project;
});

const { labelList } = storeToRefs(labelStore);

const availableLabelList = computed(() => {
  return labelList.value.filter((label) => !isReservedLabel(label));
});

const prepare = () => {
  projectStore.fetchProjectById(props.targetProjectId);
  labelStore.fetchLabelList();
};

watchEffect(prepare);

const requiredLabelList = computed((): Label[] => {
  const project = targetProject.value;
  if (!project.dbNameTemplate) return [];

  // for databases with dbNameTemplate, we need to parse required labels from its template
  return parseLabelListInTemplate(
    project.dbNameTemplate,
    availableLabelList.value
  );
});

const dbNameMatchesTemplate = computed((): boolean => {
  const project = targetProject.value;
  if (!project.dbNameTemplate) {
    // no restrictions, because no template
    return true;
  }
  const regex = buildDatabaseNameRegExpByTemplate(
    project.dbNameTemplate,
    availableLabelList.value
  );
  return regex.test(props.database.name);
});

const isRequiredLabel = (key: LabelKeyType): boolean => {
  return requiredLabelList.value.some((label) => label.key === key);
};

const isParsedLabel = (key: LabelKeyType): boolean => {
  return labelListParsedFromTemplate.value.some((label) => label.key === key);
};

const labelListParsedFromTemplate = computed((): DatabaseLabel[] => {
  if (!dbNameMatchesTemplate.value) return [];

  const regex = buildDatabaseNameRegExpByTemplate(
    targetProject.value.dbNameTemplate,
    availableLabelList.value
  );
  const match = props.database.name.match(regex);
  if (!match) return [];

  const parsedLabelList: DatabaseLabel[] = [];
  availableLabelList.value.forEach((label) => {
    const group = `label_${label.id}`;
    if (match.groups?.[group]) {
      const value = match.groups[group];
      parsedLabelList.push({
        key: label.key,
        value,
      });
    }
  });

  return parsedLabelList;
});

watch(labelListParsedFromTemplate, (list) => {
  list.forEach((label) => {
    setLabelValue(label.key, label.value);
  });
});

const allowNext = computed(() => {
  if (!dbNameMatchesTemplate.value) return false;

  // every required label must be filled
  return requiredLabelList.value.every((label) => {
    return !!getLabelValue(label.key);
  });
});

const getLabelPlaceholder = (key: LabelKeyType): string => {
  // provide "Select Tenant" if Tenant is optional
  // provide "Select {{TENANT}}" if Tenant is required in the template
  key = isRequiredLabel(key)
    ? `{{${hidePrefix(key).toUpperCase()}}}`
    : capitalize(hidePrefix(key));
  return t("create-db.select-label-value", { key });
};

const getLabelValue = (key: LabelKeyType): LabelValueType | undefined => {
  return (
    state.databaseLabelList.find((label) => label.key === key)?.value || ""
  );
};

const getLabelValueList = (label: Label): LabelValueType[] => {
  const valueList = [...label.valueList];
  if (!isRequiredLabel(label.key)) {
    // for optional labels
    // provide a "<empty value>" option ahead of other values
    valueList.unshift("");
  }
  return valueList;
};

const setLabelValue = (key: LabelKeyType, value: LabelValueType) => {
  const label = state.databaseLabelList.find((label) => label.key === key);
  if (label) {
    label.value = value;
  } else {
    state.databaseLabelList.push({ key, value });
  }
  state.databaseLabelList = state.databaseLabelList.filter(
    (label) => !!label.value
  );
};

watch(
  () => props.database,
  (db) => {
    state.databaseLabelList = cloneDeep(db.labels);
  }
);

const next = () => {
  emit("next", cloneDeep(state.databaseLabelList));
};
</script>
