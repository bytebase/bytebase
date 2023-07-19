<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('database.sync-schema.target-databases')"
      :closable="true"
      class="w-[30rem] max-w-[100vw] relative"
    >
      <div class="flex items-center justify-end mx-2 mb-2">
        <BBTableSearch
          class="m-px"
          :placeholder="$t('database.search-database')"
          @change-text="(text: string) => (state.searchText = text)"
        />
      </div>
      <NCollapse
        class="overflow-y-auto"
        arrow-placement="left"
        :default-expanded-names="
          databaseListGroupByEnvironment.map((group) => group.environment.uid)
        "
      >
        <NCollapseItem
          v-for="{
            environment,
            databaseList: databaseListInEnvironment,
          } in databaseListGroupByEnvironment"
          :key="environment.uid"
          :name="environment.uid"
        >
          <template #header>
            <label class="flex items-center gap-x-2" @click.stop="">
              <EnvironmentV1Name :environment="environment" :link="false" />
            </label>
          </template>

          <template #header-extra>
            <div class="flex items-center text-xs text-gray-500 mr-2">
              {{
                $t(
                  "database.n-selected-m-in-total",
                  getSelectionStateSummaryForEnvironment(
                    environment,
                    databaseListInEnvironment
                  )
                )
              }}
            </div>
          </template>

          <div class="relative bg-white rounded-md -space-y-px px-2">
            <template
              v-for="database in databaseListInEnvironment"
              :key="database.uid"
            >
              <label
                class="border-control-border relative border p-3 flex flex-col gap-y-2 md:flex-row md:pl-4 md:pr-6"
                :class="
                  database.syncState === State.ACTIVE
                    ? 'cursor-pointer'
                    : 'cursor-not-allowed'
                "
              >
                <div class="radio text-sm flex justify-start md:flex-1">
                  <input
                    type="checkbox"
                    class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
                    :checked="isDatabaseSelected(database.uid)"
                    @input="(e: any) => toggleDatabaseSelected(database.uid, e.target.checked)"
                  />
                  <span
                    class="font-medium ml-2 text-main"
                    :class="database.syncState !== State.ACTIVE && 'opacity-40'"
                    >{{ database.databaseName }}</span
                  >
                </div>
                <div
                  class="flex items-center gap-x-1 textinfolabel ml-6 pl-0 md:ml-0 md:pl-0 md:justify-end"
                >
                  <InstanceV1Name
                    :instance="database.instanceEntity"
                    :link="false"
                  />
                </div>
              </label>
            </template>
          </div>
        </NCollapseItem>
      </NCollapse>

      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton
            type="primary"
            :loading="state.isLoading"
            @click="handleConfirm"
          >
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script setup lang="ts">
import { head } from "lodash-es";
import { computed, reactive } from "vue";
import {
  NCollapse,
  NCollapseItem,
  NButton,
  NDrawer,
  NDrawerContent,
} from "naive-ui";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import { ComposedDatabase } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import { Engine, State, engineToJSON } from "@/types/proto/v1/common";
import axios from "axios";
import dayjs from "dayjs";
import { useRouter } from "vue-router";

type LocalState = {
  searchText: string;
  isLoading: boolean;
  selectedDatabaseList: ComposedDatabase[];
};

const props = defineProps<{
  projectId: string;
  engine: Engine;
  schema: string;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const router = useRouter();
const environmentV1Store = useEnvironmentV1Store();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  searchText: "",
  isLoading: false,
  selectedDatabaseList: [],
});

const databaseListGroupByEnvironment = computed(() => {
  const project = useProjectV1Store().getProjectByUID(props.projectId);
  const databaseList =
    databaseStore
      .databaseListByProject(project.name)
      .filter((db) => db.databaseName.includes(state.searchText))
      .filter((db) => db.instanceEntity.engine === props.engine) || [];
  const listByEnv = environmentV1Store.environmentList.map((environment) => {
    const list = databaseList.filter(
      (db) => db.instanceEntity.environment === environment.name
    );
    return {
      environment,
      databaseList: list,
    };
  });

  return listByEnv.filter((group) => group.databaseList.length > 0);
});

const isDatabaseSelected = (databaseId: string) => {
  const idList = state.selectedDatabaseList.map((db) => db.uid);
  return idList.includes(databaseId);
};

const toggleDatabaseSelected = (databaseId: string, selected: boolean) => {
  const index = state.selectedDatabaseList.findIndex(
    (db) => db.uid === databaseId
  );
  if (selected) {
    if (index < 0) {
      // Now we only allow select one database.
      state.selectedDatabaseList = [databaseStore.getDatabaseByUID(databaseId)];
    }
  } else {
    if (index >= 0) {
      state.selectedDatabaseList.splice(index, 1);
    }
  }
};

const getSelectionStateSummaryForEnvironment = (
  environment: Environment,
  databaseList: ComposedDatabase[]
) => {
  const set = new Set(
    state.selectedDatabaseList
      .filter((db) => db.instanceEntity.environment === environment.name)
      .map((db) => db.uid)
  );
  const selected = databaseList.filter((db) => set.has(db.uid)).length;
  const total = databaseList.length;

  return { selected, total };
};

const handleConfirm = async () => {
  if (state.selectedDatabaseList.length === 0) {
    return;
  }
  if (state.isLoading) {
    return false;
  }
  state.isLoading = true;
  const database = head(state.selectedDatabaseList) as ComposedDatabase;
  const targetDatabaseSchema = await databaseStore.fetchDatabaseSchema(
    `${database.name}/schema`,
    true
  );

  const diff = await getSchemaDiff(
    props.engine,
    targetDatabaseSchema.schema,
    props.schema
  );

  const query: Record<string, any> = {
    template: "bb.issue.database.schema.update",
    project: props.projectId,
    mode: "normal",
    ghost: undefined,
  };
  query.databaseList = `${database.uid}`;
  query.sqlList = JSON.stringify([diff]);
  query.name = generateIssueName([database.databaseName]);

  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query,
  };
  router.push(routeInfo);
};

const generateIssueName = (databaseNameList: string[]) => {
  const issueNameParts: string[] = [];
  if (databaseNameList.length === 1) {
    issueNameParts.push(`[${databaseNameList[0]}]`);
  } else {
    issueNameParts.push(`[${databaseNameList.length} databases]`);
  }
  issueNameParts.push(`Alter schema`);
  const datetime = dayjs().format("@MM-DD HH:mm");
  const tz = "UTC" + dayjs().format("ZZ");
  issueNameParts.push(`${datetime} ${tz}`);
  return issueNameParts.join(" ");
};

const getSchemaDiff = async (
  engine: Engine,
  sourceSchema: string,
  targetSchema: string
) => {
  const { data } = await axios.post("/v1/sql/schema/diff", {
    engineType: engineToJSON(engine),
    sourceSchema,
    targetSchema,
  });
  return data as string;
};
</script>
