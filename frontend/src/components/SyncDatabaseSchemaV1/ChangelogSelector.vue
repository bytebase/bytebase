<template>
  <RemoteResourceSelector
    ref="remoteResourceSelectorRef"
    :multiple="false"
    :filterable="false"
    :disabled="disabled"
    :value="value"
    :fallback-option="false"
    :render-label="renderSchemaVersionLabel"
    :consistent-menu-width="false"
    :placeholder="$t('changelog.self')"
    :search="handleSearch"
    @update:value="(val) => $emit('update:value', val as (string | undefined))"
  />
</template>

<script lang="tsx" setup>
import { NTag } from "naive-ui";
import { computed, ref, watch } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import RemoteResourceSelector from "@/components/v2/Select/RemoteResourceSelector/index.vue";
import type { ResourceSelectOption } from "@/components/v2/Select/RemoteResourceSelector/types";
import { useChangelogStore, useDatabaseV1Store } from "@/store";
import { getDateForPbTimestampProtoEs, isValidDatabaseName } from "@/types";
import type { Changelog } from "@/types/proto-es/v1/database_service_pb";
import {
  Changelog_Status,
  Changelog_Type,
} from "@/types/proto-es/v1/database_service_pb";
import {
  isValidChangelogName,
  mockLatestChangelog,
} from "@/utils/v1/changelog";
import HumanizeDate from "../misc/HumanizeDate.vue";

const props = defineProps<{
  database?: string;
  value?: string;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string | undefined): void;
}>();

const ALLOWED_CHANGELOG_TYPES: Changelog_Type[] = [
  Changelog_Type.BASELINE,
  Changelog_Type.MIGRATE,
];

const changelogStore = useChangelogStore();
const databaseStore = useDatabaseV1Store();
const remoteResourceSelectorRef =
  ref<ComponentExposed<typeof RemoteResourceSelector<Changelog>>>();

const disabled = computed(() => !isValidDatabaseName(props.database));

watch(
  () => props.database,
  async (database) => {
    if (!isValidDatabaseName(database)) {
      return;
    }
    await remoteResourceSelectorRef.value?.reset();
    emit("update:value", remoteResourceSelectorRef.value?.options[0]?.value);
  },
  { immediate: true }
);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  if (!isValidDatabaseName(props.database)) {
    return {
      nextPageToken: "",
      options: [],
    };
  }

  const { changelogs, nextPageToken } = await changelogStore.fetchChangelogList(
    {
      parent: props.database,
      pageToken: params.pageToken,
      pageSize: params.pageSize,
      filter: [
        `status == "${Changelog_Status[Changelog_Status.DONE]}"`,
        `type in [${ALLOWED_CHANGELOG_TYPES.map((t) => `"${Changelog_Type[t]}"`).join(", ")}]`,
      ].join(" && "),
    }
  );

  const options = changelogs.map((changelog) => ({
    resource: changelog,
    value: changelog.name,
    label: changelog.name,
  }));

  if (options.length === 0) {
    const db = databaseStore.getDatabaseByName(props.database);
    const changelog = mockLatestChangelog(db);
    options.push({
      resource: changelog,
      value: changelog.name,
      label: changelog.name,
    });
  }

  return {
    nextPageToken,
    options,
  };
};

const renderSchemaVersionLabel = (
  option: ResourceSelectOption<Changelog>,
  _selected: boolean,
  _searchText: string
) => {
  const { resource } = option;
  if (!resource || !isValidChangelogName(resource.name)) {
    return "Latest version";
  }

  return (
    <div class="flex flex-row justify-start items-center truncate gap-1">
      <HumanizeDate
        class="text-control-light"
        date={getDateForPbTimestampProtoEs(resource.createTime)}
      />
      <NTag round size="small">
        {Changelog_Type[resource.type]}
      </NTag>
      {resource.planTitle && (
        <NTag round size="small">
          {resource.planTitle}
        </NTag>
      )}
    </div>
  );
};
</script>