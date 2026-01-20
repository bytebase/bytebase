<template>
  <div
    class="w-full flex flex-row justify-start items-center flex-wrap gap-2 text-sm"
  >
    <span>
      {{ t("database.sync-schema.source-schema") }}
    </span>
    <template v-if="changelogSourceSchema">
      <NTag round @click="gotoDatabase">
        <span class="opacity-60 mr-1">{{ t("common.database") }}</span>
        <EngineIcon custom-class="inline-flex w-4 h-auto" :engine="engine" />
        <span>{{ extractDatabaseResourceName(databaseFromChangelog.name).databaseName }}</span>
      </NTag>
      <NTag round @click="gotoChangelog">
        <span class="opacity-60 mr-1">{{ t("common.changelog") }}</span>
        <span>{{ changelogUID ? `#${changelogUID}` : "Latest" }}</span>
      </NTag>
    </template>
    <template v-else>
      <NTag round>
        {{ t("schema-editor.raw-sql") }}
      </NTag>
      <NTag round>
        <span class="opacity-60 mr-1">{{ t("database.engine") }}</span>
        <EngineIcon custom-class="inline-flex w-4 h-auto" :engine="engine" />
        <span>{{ engineNameV1(engine) }}</span>
      </NTag>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  databaseV1Url,
  engineNameV1,
  extractChangelogUID,
  extractDatabaseResourceName,
  isValidChangelogName,
} from "@/utils";
import { EngineIcon } from "../Icon";
import { type ChangelogSourceSchema } from "./types";

const props = defineProps<{
  project: Project;
  schemaString: string;
  engine: Engine;
  changelogSourceSchema?: ChangelogSourceSchema;
}>();

const { t } = useI18n();
const databaseStore = useDatabaseV1Store();

const databaseFromChangelog = computed(() => {
  return databaseStore.getDatabaseByName(
    props.changelogSourceSchema?.databaseName || "" // Or unknown database.
  );
});

const changelogUID = computed(() => {
  const maybeChangelogName = props.changelogSourceSchema?.changelogName || "";
  if (!isValidChangelogName(maybeChangelogName)) {
    return undefined;
  }
  return extractChangelogUID(maybeChangelogName);
});

const gotoDatabase = () => {
  if (isValidDatabaseName(databaseFromChangelog.value.name)) {
    window.open(databaseV1Url(databaseFromChangelog.value));
  }
};

const gotoChangelog = () => {
  if (isValidChangelogName(props.changelogSourceSchema?.changelogName || "")) {
    window.open(
      `${databaseV1Url(databaseFromChangelog.value)}/changelogs/${changelogUID.value}`
    );
  }
};
</script>
