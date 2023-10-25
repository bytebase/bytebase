<template>
  <Drawer :show="show" :close-on-esc="true" @close="show = false">
    <DrawerContent :title="$t('changelist.apply-to-database')">
      <template #default>
        <div
          class="w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
        >
          <div v-if="ready" class="flex flex-col gap-y-4">
            <ProjectStandardView
              :state="state"
              :project="project"
              :database-list="schemaDatabaseList"
              :environment-list="environmentList"
              @select-database="handleSelectDatabase"
            >
              <template #header>
                <div class="flex items-center justify-end mx-2">
                  <SearchBox
                    v-model:value="state.keyword"
                    class="m-px"
                    :placeholder="$t('database.filter-database')"
                  />
                </div>
              </template>
            </ProjectStandardView>
            <SchemalessDatabaseTable
              v-if="guessedDatabaseChangeType === 'DDL'"
              mode="PROJECT"
              class="px-2"
              :database-list="schemalessDatabaseList"
            />
          </div>
          <div
            v-if="!ready || state.isGenerating"
            v-zindexable="{ enabled: true }"
            class="absolute inset-0 flex flex-col items-center justify-center bg-white/50"
          >
            <BBSpin />
          </div>
        </div>
      </template>

      <template #footer>
        <div class="flex-1 flex items-center justify-between">
          <div>
            <div
              v-if="flattenSelectedDatabaseUidList.length > 0"
              class="textinfolabel"
            >
              {{
                $t("database.selected-n-databases", {
                  n: flattenSelectedDatabaseUidList.length,
                })
              }}
            </div>
          </div>

          <div class="flex items-center justify-end gap-x-3">
            <NButton @click.prevent="show = false">
              {{ $t("common.cancel") }}
            </NButton>

            <ErrorTipsButton
              style="--n-padding: 0 10px"
              :errors="nextButtonErrors"
              :button-props="{
                type: 'primary',
              }"
              @click="handleClickNext"
            >
              {{ $t("common.next") }}
            </ErrorTipsButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ProjectStandardView, {
  ProjectStandardViewState,
} from "@/components/AlterSchemaPrepForm/ProjectStandardView.vue";
import { ErrorTipsButton, SearchBox } from "@/components/v2";
import {
  useDatabaseV1Store,
  useEnvironmentV1List,
  useSearchDatabaseV1List,
} from "@/store";
import { ComposedDatabase, DEFAULT_PROJECT_V1_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  filterDatabaseV1ByKeyword,
  guessChangelistChangeType,
  instanceV1HasAlterSchema,
  sortDatabaseV1List,
} from "@/utils";
import { generateIssueName } from "../common";
import { useChangelistDetailContext } from "../context";

type LocalState = ProjectStandardViewState & {
  keyword: string;
  isGenerating: boolean;
};

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const {
  changelist,
  project,
  showApplyToDatabasePanel: show,
} = useChangelistDetailContext();

const state = reactive<LocalState>({
  selectedDatabaseUidListForEnvironment: new Map(),
  alterType: "MULTI_DB",
  keyword: "",
  isGenerating: false,
});

const { ready } = useSearchDatabaseV1List(
  computed(() => ({
    parent: "instances/-",
    filter: `project == "${project.value.name}"`,
  }))
);

const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const guessedDatabaseChangeType = computed(() => {
  if (
    changelist.value.changes.some(
      (change) => guessChangelistChangeType(change) === "DDL"
    )
  ) {
    return "DDL";
  }
  return "DML";
});

const databaseList = computed(() => {
  let list: ComposedDatabase[] = [];
  list = databaseStore.databaseListByProject(project.value.name);

  list = list.filter(
    (db) =>
      db.syncState == State.ACTIVE && db.project !== DEFAULT_PROJECT_V1_NAME
  );

  list = list.filter((db) => {
    return filterDatabaseV1ByKeyword(db, state.keyword.trim(), [
      "name",
      "environment",
      "instance",
    ]);
  });

  return sortDatabaseV1List(list);
});

const schemaDatabaseList = computed(() => {
  if (guessedDatabaseChangeType.value === "DDL") {
    return databaseList.value.filter((db) =>
      instanceV1HasAlterSchema(db.instanceEntity)
    );
  }

  return databaseList.value;
});

const schemalessDatabaseList = computed(() => {
  return databaseList.value.filter(
    (db) => !instanceV1HasAlterSchema(db.instanceEntity)
  );
});

const flattenSelectedDatabaseUidList = computed(() => {
  const flattenDatabaseIdList: string[] = [];
  for (const databaseIdList of state.selectedDatabaseUidListForEnvironment.values()) {
    flattenDatabaseIdList.push(...databaseIdList);
  }
  return flattenDatabaseIdList;
});

const nextButtonErrors = computed(() => {
  const errors: string[] = [];

  if (flattenSelectedDatabaseUidList.value.length === 0) {
    errors.push(t("changelist.error.select-at-least-one-database"));
  }
  return errors;
});

const handleSelectDatabase = (db: ComposedDatabase) => {
  console.log("handleSelectDatabase", db);
};

const handleClickNext = async () => {
  state.isGenerating = true;
  try {
    const databaseUIDList = [...flattenSelectedDatabaseUidList.value];
    const databaseList = databaseUIDList.map((uid) =>
      databaseStore.getDatabaseByUID(uid)
    );

    const query: Record<string, any> = {
      template:
        guessedDatabaseChangeType.value === "DDL"
          ? "bb.issue.database.schema.update"
          : "bb.issue.database.data.update",
      name: generateIssueName(
        databaseList.map((db) => db.databaseName),
        changelist.value
      ),
      project: project.value.uid,
      changelist: changelist.value.name,
      databaseList: databaseUIDList.join(","),
    };

    router.push({
      name: "workspace.issue.detail",
      params: {
        issueSlug: "new",
      },
      query,
    });
  } catch {
    state.isGenerating = false;
  }
};

const reset = () => {
  state.selectedDatabaseUidListForEnvironment = new Map();
  state.keyword = "";
};

watch(
  show,
  (show) => {
    if (show) reset();
  },
  { immediate: true }
);
</script>
