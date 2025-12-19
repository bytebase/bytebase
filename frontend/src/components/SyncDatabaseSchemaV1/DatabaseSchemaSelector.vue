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
        :environment-name="state.environmentName"
        @update:environment-name="handleEnvironmentSelect"
      />
      <DatabaseSelect
        :placeholder="$t('database.select')"
        :project-name="project.name"
        :database-name="state.databaseName"
        :environment-name="state.environmentName"
        :allowed-engine-type-list="ALLOWED_ENGINES"
        @update:database-name="handleDatabaseSelect"
      />
    </div>
    <div
      class="w-full flex flex-col gap-y-2 lg:flex-row justify-start items-start lg:items-center"
    >
      <span class="flex w-40 items-center shrink-0 text-sm">
        {{ $t("database.sync-schema.schema-version.self") }}
      </span>
      <div class="w-full flex flex-row justify-start items-center relative">
        <NSelect
          :loading="isPreparingSchemaVersionOptions"
          :value="state.changelogName"
          :options="schemaVersionOptions"
          :placeholder="$t('changelog.select')"
          :disabled="schemaVersionOptions.length === 0"
          :render-label="renderSchemaVersionLabel"
          :fallback-option="
            isMockLatestSchemaChangelogSelected
              ? fallbackSchemaVersionOption
              : false
          "
          @update:value="handleSchemaVersionSelect"
        />
      </div>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { head } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { NSelect, NTag } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { DatabaseSelect, EnvironmentSelect } from "@/components/v2";
import {
  useChangelogStore,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import {
  getDateForPbTimestampProtoEs,
  isValidDatabaseName,
  UNKNOWN_ID,
} from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Status,
  Changelog_Type,
} from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  extractChangelogUID,
  isValidChangelogName,
  mockLatestChangelog,
} from "@/utils/v1/changelog";
import HumanizeDate from "../misc/HumanizeDate.vue";
import { ALLOWED_ENGINES, type ChangelogSourceSchema } from "./types";

const ALLOWED_CHANGELOG_TYPES: Changelog_Type[] = [
  Changelog_Type.BASELINE,
  Changelog_Type.MIGRATE,
];

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
const dbSchemaStore = useDBSchemaV1Store();
const changelogStore = useChangelogStore();

const isPreparingSchemaVersionOptions = ref(false);

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
    dbSchemaStore.getOrFetchDatabaseMetadata({
      database: database.name,
    });
  } else {
    state.databaseName = undefined;
  }
};

const databaseChangelogList = (databaseName: string) => {
  return changelogStore
    .changelogListByDatabase(databaseName)
    .filter(
      (changelog) =>
        ALLOWED_CHANGELOG_TYPES.includes(changelog.type) &&
        changelog.status === Changelog_Status.DONE
    );
};

const schemaVersionOptions = computed(() => {
  const { databaseName } = state;
  if (!isValidDatabaseName(databaseName)) {
    return [];
  }
  const changelogs = databaseChangelogList(databaseName);
  if (changelogs.length === 0) return [];
  const options: SelectOption[] = [
    {
      value: "PLACEHOLDER",
      label: t("changelog.select"),
      disabled: true,
      style: "cursor: default",
    },
  ];
  options.push(
    ...changelogs.map<SelectOption>((changelog, index) => {
      return {
        changelog,
        index,
        value: changelog.name,
        label: changelog.name,
      };
    })
  );
  return options;
});

const renderSchemaVersionLabel = (option: SelectOption) => {
  if (option.disabled || !option.changelog) {
    return option.label as string;
  }
  const changelog = option.changelog as Changelog;
  if (!isValidChangelogName(changelog.name)) {
    return "Latest version";
  }

  return (
    <div class="flex flex-row justify-start items-center truncate gap-1">
      <HumanizeDate
        class="text-control-light"
        date={getDateForPbTimestampProtoEs(changelog.createTime)}
      />
      <NTag round size="small">
        {Changelog_Type[changelog.type]}
      </NTag>
      {changelog.version && (
        <NTag round size="small">
          {changelog.version}
        </NTag>
      )}
      {changelog.statement ? (
        <span class="truncate">{changelog.statement}</span>
      ) : (
        <span class="text-gray-400">{t("common.empty")}</span>
      )}
    </div>
  );
};

const isMockLatestSchemaChangelogSelected = computed(() => {
  if (!state.changelogName) return false;
  return extractChangelogUID(state.changelogName) === String(UNKNOWN_ID);
});

const fallbackSchemaVersionOption = (value: string): SelectOption => {
  if (extractChangelogUID(value) === String(UNKNOWN_ID)) {
    const { databaseName } = state;
    if (isValidDatabaseName(databaseName)) {
      const db = databaseStore.getDatabaseByName(databaseName);
      const changelog = mockLatestChangelog(db);
      return {
        changelog,
        index: 0,
        value: changelog.name,
        label: changelog.name,
      };
    }
  }
  return {
    value: "PLACEHOLDER",
    disabled: true,
    label: t("changelog.select"),
    style: "cursor: default",
  };
};

const handleSchemaVersionSelect = async (_: string, option: SelectOption) => {
  const changelog = option.changelog as Changelog;
  state.changelogName = changelog.name;
};

watch(
  () => state.databaseName,
  async (databaseName) => {
    if (!isValidDatabaseName(databaseName)) {
      state.changelogName = undefined;
      return;
    }

    const database = databaseStore.getDatabaseByName(databaseName);
    if (database) {
      try {
        isPreparingSchemaVersionOptions.value = true;
        const changelogList =
          await changelogStore.getOrFetchChangelogListOfDatabase(database.name);
        const filteredChangelogList = changelogList.filter((changelog) =>
          ALLOWED_CHANGELOG_TYPES.includes(changelog.type)
        );

        if (filteredChangelogList.length > 0) {
          // Default select the first changelog.
          state.changelogName = head(filteredChangelogList)?.name;
        } else {
          // If database has no changelog, we will use its latest schema.
          state.changelogName = mockLatestChangelog(database).name;
        }
      } finally {
        isPreparingSchemaVersionOptions.value = false;
      }
    } else {
      state.changelogName = undefined;
    }
  },
  { immediate: true }
);

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
