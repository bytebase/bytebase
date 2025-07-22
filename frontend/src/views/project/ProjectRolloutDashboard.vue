<template>
  <div class="w-full">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="w-full flex flex-1 items-center justify-between gap-x-2">
        <AdvancedSearch
          v-model:params="state.params"
          class="flex-1"
          :scope-options="scopeOptions"
        />
        <UpdatedTimeRange
          :params="state.params"
          @update:params="state.params = $event"
        />
      </div>
    </div>

    <div class="relative w-full mt-4 min-h-[20rem]">
      <PagedTable
        ref="rolloutPagedTable"
        :key="project.name"
        :session-key="`project-${project.name}-rollouts`"
        :fetch-list="fetchRolloutList"
      >
        <template #table="{ list, loading }">
          <RolloutDataTable
            :bordered="true"
            :loading="loading"
            :rollout-list="list"
          />
        </template>
      </PagedTable>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, ref, watch, h } from "vue";
import type { ComponentExposed } from "vue-component-type-helpers";
import { useI18n } from "vue-i18n";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import UpdatedTimeRange from "@/components/AdvancedSearch/UpdatedTimeRange.vue";
import type {
  ScopeOption,
  ValueOption,
} from "@/components/AdvancedSearch/types";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import RolloutDataTable from "@/components/Rollout/RolloutDataTable.vue";
import SystemBotTag from "@/components/misc/SystemBotTag.vue";
import YouTag from "@/components/misc/YouTag.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import {
  useCurrentUserV1,
  useProjectByName,
  useRolloutStore,
  useUserStore,
} from "@/store";
import {
  buildRolloutFindBySearchParams,
  type RolloutFind,
} from "@/store/modules/rollout";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { SYSTEM_BOT_USER_NAME } from "@/types";
import {
  Task_Type,
  type Rollout,
} from "@/types/proto-es/v1/rollout_service_pb";
import { UserType } from "@/types/proto-es/v1/user_service_pb";
import {
  extractProjectResourceName,
  getDefaultPagination,
  type SearchParams,
  type SearchScope,
  type SearchScopeId,
} from "@/utils";

interface LocalState {
  params: SearchParams;
}

const props = defineProps<{
  projectId: string;
}>();

const { t } = useI18n();
const me = useCurrentUserV1();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const rolloutStore = useRolloutStore();
const userStore = useUserStore();
const rolloutPagedTable = ref<ComponentExposed<typeof PagedTable<Rollout>>>();

const readonlyScopes = computed((): SearchScope[] => {
  return [];
});

const defaultSearchParams = () => {
  const params: SearchParams = {
    query: "",
    scopes: [...readonlyScopes.value],
  };
  return params;
};

const state = reactive<LocalState>({
  params: defaultSearchParams(),
});

watch(
  () => project.value.name,
  () => (state.params = defaultSearchParams())
);

const supportedScopes = computed(() => {
  const supportedScopes: SearchScopeId[] = ["creator", "updated"];
  return supportedScopes;
});

// Custom scope options for rollouts that includes creator functionality
const scopeOptions = computed((): ScopeOption[] => {
  const commonOptions = useCommonSearchScopeOptions(
    supportedScopes.value.filter((id) => id !== "creator")
  );

  const renderSpan = (text: string) => h("span", text);

  const searchPrincipalSearchValueOptions = (userTypes: UserType[]) => {
    return ({
      keyword,
      nextPageToken,
    }: {
      keyword: string;
      nextPageToken?: string;
    }) =>
      userStore
        .fetchUserList({
          pageToken: nextPageToken,
          pageSize: getDefaultPagination(),
          filter: {
            types: userTypes,
            query: keyword,
          },
        })
        .then((resp) => ({
          nextPageToken: resp.nextPageToken,
          options: resp.users.map<ValueOption>((user) => {
            return {
              value: user.email,
              keywords: [user.email, user.title],
              bot: user.name === SYSTEM_BOT_USER_NAME,
              render: () => {
                const children = [
                  h(BBAvatar, { size: "TINY", username: user.title }),
                  renderSpan(user.title),
                ];
                if (user.name === me.value.name) {
                  children.push(h(YouTag));
                }
                if (user.name === SYSTEM_BOT_USER_NAME) {
                  children.push(h(SystemBotTag));
                }
                return h(
                  "div",
                  { class: "flex items-center gap-x-1" },
                  children
                );
              },
            };
          }),
        }));
  };

  const creatorOption: ScopeOption = {
    id: "creator",
    title: t("issue.advanced-search.scope.creator.title"),
    description: t("issue.advanced-search.scope.creator.description"),
    search: searchPrincipalSearchValueOptions([
      UserType.USER,
      UserType.SERVICE_ACCOUNT,
      UserType.SYSTEM_BOT,
    ]),
  };

  return [...commonOptions.value, creatorOption];
});

const rolloutSearchParams = computed(() => {
  const defaultScopes = [
    {
      id: "project",
      value: extractProjectResourceName(project.value.name),
    },
  ];
  return {
    query: state.params.query.trim().toLowerCase(),
    scopes: [...state.params.scopes, ...defaultScopes],
  } as SearchParams;
});

const mergedRolloutFind = computed((): RolloutFind => {
  return buildRolloutFindBySearchParams(rolloutSearchParams.value, {
    taskType: [
      Task_Type.DATABASE_DATA_UPDATE,
      Task_Type.DATABASE_SCHEMA_UPDATE,
      Task_Type.DATABASE_SCHEMA_UPDATE_GHOST,
      Task_Type.DATABASE_SCHEMA_UPDATE_SDL,
    ],
  });
});

const fetchRolloutList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const { nextPageToken, rollouts } = await rolloutStore.listRollouts({
    find: mergedRolloutFind.value,
    pageSize,
    pageToken,
  });
  return {
    nextPageToken,
    list: rollouts,
  };
};

watch(
  () => JSON.stringify(mergedRolloutFind.value),
  () => rolloutPagedTable.value?.refresh()
);
</script>
