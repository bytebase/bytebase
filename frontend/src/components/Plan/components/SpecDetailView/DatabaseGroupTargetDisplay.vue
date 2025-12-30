<template>
  <div class="flex flex-col gap-2 w-full">
    <!-- Database group header -->
    <div class="flex items-center gap-x-2">
      <DatabaseGroupIcon class="w-5 h-5 text-control shrink-0" />
      <NTag size="small" round>
        {{ $t("common.database-group") }}
      </NTag>
      <DatabaseGroupName
        :database-group="dbGroupStore.getDBGroupByName(target)"
        :link="false"
        :plain="true"
      />
      <!-- Only show external link if the database group still exists -->
      <div
        v-if="showExternalLink && isValidDatabaseGroupName(dbGroupStore.getDBGroupByName(target).name)"
        class="flex items-center cursor-pointer opacity-60 hover:opacity-100"
        @click="gotoDatabaseGroupDetailPage"
      >
        <ExternalLinkIcon class="w-4 h-auto" />
      </div>
    </div>
    <!-- Database children - flat layout -->
    <div
      v-if="databases.length > 0"
      class="flex flex-wrap items-center gap-2 pl-7"
    >
      <div
        v-for="db in databases.slice(0, MAX_INLINE_DATABASES)"
        :key="db"
        class="inline-flex items-center gap-x-1 px-2 py-1 border rounded-lg transition-all cursor-default bg-gray-50"
      >
        <DatabaseDisplay :database="db" size="small" show-environment />
      </div>
      <NPopover
        v-if="databases.length > MAX_INLINE_DATABASES"
        placement="bottom"
      >
        <template #trigger>
          <span class="text-xs text-accent cursor-pointer">
            {{ $t("common.n-more", { n: databases.length - MAX_INLINE_DATABASES }) }}
          </span>
        </template>
        <NVirtualList
          class="max-h-64"
          :item-size="24"
          :items="extraDatabaseItems"
          :key-field="'key'"
        >
          <template #default="{ item }">
            <div class="py-1">
              <DatabaseDisplay
                :database="item.value"
                size="small"
                show-environment
              />
            </div>
          </template>
        </NVirtualList>
      </NPopover>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ExternalLinkIcon } from "lucide-vue-next";
import { NPopover, NTag, NVirtualList } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { useRouter } from "vue-router";
import DatabaseGroupIcon from "@/components/DatabaseGroupIcon.vue";
import DatabaseGroupName from "@/components/v2/Model/DatabaseGroupName.vue";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import {
  getProjectNameAndDatabaseGroupName,
  useDatabaseV1Store,
  useDBGroupStore,
} from "@/store";
import { isValidDatabaseGroupName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";

const MAX_INLINE_DATABASES = 5;

const props = withDefaults(
  defineProps<{
    target: string;
    showExternalLink?: boolean;
  }>(),
  {
    showExternalLink: true,
  }
);

const router = useRouter();
const dbGroupStore = useDBGroupStore();
const dbStore = useDatabaseV1Store();

const databases = ref<string[]>([]);

// Fetch database group and populate databases
watchEffect(async () => {
  try {
    const dbGroup = await dbGroupStore.getOrFetchDBGroupByName(props.target, {
      view: DatabaseGroupView.FULL,
      silent: true,
    });
    const matchedDatabases =
      dbGroup.matchedDatabases?.map((db) => db.name) ?? [];
    databases.value = matchedDatabases;

    // Fetch matched databases so DatabaseDisplay can show environment
    if (matchedDatabases.length > 0) {
      await dbStore.batchGetOrFetchDatabases(matchedDatabases);
    }
  } catch {
    databases.value = [];
  }
});

const extraDatabaseItems = computed(() => {
  return databases.value.slice(MAX_INLINE_DATABASES).map((db) => ({
    key: db,
    value: db,
  }));
});

const gotoDatabaseGroupDetailPage = () => {
  const [projectId, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    props.target
  );
  const url = router.resolve({
    name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
    params: {
      projectId,
      databaseGroupName,
    },
  }).fullPath;
  window.open(url, "_blank");
};
</script>
