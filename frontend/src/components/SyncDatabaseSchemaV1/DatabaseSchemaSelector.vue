<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start gap-y-3 mb-6"
  >
    <div
      class="w-full flex flex-col gap-y-2 lg:flex-row justify-start items-start lg:items-center"
    >
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("common.database") }}
      </span>
      <EnvironmentSelect
        class="mr-3"
        name="environment"
        :value="state.environmentName"
        @update:value="handleEnvironmentSelect($event as (string | undefined))"
      />
      <DatabaseSelect
        :placeholder="$t('database.select')"
        :project-name="project.name"
        :value="state.databaseName"
        :environment-name="state.environmentName"
        :allowed-engine-type-list="ALLOWED_ENGINES"
        @update:value="(val) => handleDatabaseSelect(val as (string | undefined))"
      />
    </div>
    <div
      class="w-full flex flex-col gap-y-2"
    >
      <div class="text-sm">
        {{ $t("database.sync-schema.schema-version.self") }}
        <div class="textinfolabel">
        {{ t("changelog.select") }}
      </div>
      </div>
      <div class="w-full flex flex-row justify-start items-center relative">
        <ChangelogSelector
          v-model:value="state.changelogName"
          :database="state.databaseName"
        />
      </div>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { DatabaseSelect, EnvironmentSelect } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import ChangelogSelector from "./ChangelogSelector.vue";
import { ALLOWED_ENGINES, type ChangelogSourceSchema } from "./types";

const props = defineProps<{
  project: Project;
  sourceSchema?: ChangelogSourceSchema;
}>();

const emit = defineEmits<{
  (event: "update", sourceSchema: ChangelogSourceSchema): void;
}>();

interface LocalState {
  showFeatureModal: boolean;
  environmentName?: string;
  databaseName?: string;
  changelogName?: string;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  showFeatureModal: false,
  environmentName: props.sourceSchema?.environmentName,
  databaseName: props.sourceSchema?.databaseName,
  changelogName: props.sourceSchema?.changelogName,
});
const databaseStore = useDatabaseV1Store();

const handleEnvironmentSelect = async (name: string | undefined) => {
  if (name !== state.environmentName) {
    state.databaseName = "";
  }
  state.environmentName = name;
};

const handleDatabaseSelect = async (name: string | undefined) => {
  if (isValidDatabaseName(name)) {
    const database = databaseStore.getDatabaseByName(name);
    if (!database) {
      return;
    }
    state.environmentName = database.effectiveEnvironment;
    state.databaseName = name;
    state.changelogName = "";
  } else {
    state.databaseName = "";
  }
};

watch(
  [
    () => state.environmentName,
    () => state.databaseName,
    () => state.changelogName,
  ],
  ([environmentName, databaseName, changelogName]) => {
    const params: ChangelogSourceSchema = {
      environmentName,
      databaseName,
      changelogName,
    };
    emit("update", params);
  }
);

watch(
  [
    () => props.sourceSchema?.environmentName,
    () => props.sourceSchema?.databaseName,
    () => props.sourceSchema?.changelogName,
  ],
  ([environmentName, databaseName, changelogName]) => {
    state.environmentName = environmentName;
    state.databaseName = databaseName;
    state.changelogName = changelogName;
  }
);
</script>
