<template>
  <Drawer v-model:show="show">
    <DrawerContent :title="$t('changelist.apply-to-database')">
      <template #default>
        <div
          class="w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
        >
          <div v-if="ready" class="flex flex-col">
            <div class="mb-4 flex flex-row justify-start items-center gap-2">
              <span class="textlabel">{{ t("changelist.change-type") }}:</span>
              <NRadioGroup v-model:value="state.changeType">
                <NRadio :value="'DDL'" :label="t('issue.title.edit-schema')" />
                <NRadio :value="'DML'" :label="t('issue.title.change-data')" />
              </NRadioGroup>
            </div>

            <NTabs v-model:value="state.changeSource" type="card">
              <NTabPane name="DATABASE" :tab="$t('common.databases')">
                <DatabaseV1Table
                  mode="PROJECT"
                  :database-list="schemaDatabaseList"
                  :show-selection="true"
                  :selected-database-names="state.selectedDatabaseNameList"
                  @update:selected-databases="
                    state.selectedDatabaseNameList = Array.from($event)
                  "
                />
                <SchemalessDatabaseTable
                  v-if="
                    state.changeType === 'DDL' &&
                    schemalessDatabaseList.length > 0
                  "
                  mode="PROJECT"
                  :database-list="schemalessDatabaseList"
                />
              </NTabPane>
              <NTabPane name="GROUP" :tab="$t('common.database-groups')">
                <DatabaseGroupDataTable
                  :database-group-list="dbGroupList"
                  :show-selection="true"
                  :single-selection="true"
                  :selected-database-group-names="
                    state.selectedDatabaseGroup
                      ? [state.selectedDatabaseGroup]
                      : []
                  "
                  @update:selected-database-groups="
                    state.selectedDatabaseGroup = head(Array.from($event))
                  "
                />
              </NTabPane>
            </NTabs>
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
              v-if="
                state.changeSource === 'DATABASE' &&
                flattenSelectedDatabaseNameList.length > 0
              "
              class="textinfolabel"
            >
              {{
                $t("database.selected-n-databases", {
                  n: flattenSelectedDatabaseNameList.length,
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
import { head } from "lodash-es";
import { NButton } from "naive-ui";
import { NRadioGroup, NRadio, NTabs, NTabPane } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import SchemalessDatabaseTable from "@/components/AlterSchemaPrepForm/SchemalessDatabaseTable.vue";
import DatabaseGroupDataTable from "@/components/DatabaseGroup/DatabaseGroupDataTable.vue";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table/DatabaseV1Table.vue";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useDatabaseV1Store,
  useDBGroupListByProject,
  useDBGroupStore,
} from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import { DatabaseGroup } from "@/types/proto/v1/database_group_service";
import {
  extractProjectResourceName,
  generateIssueTitle,
  instanceV1HasAlterSchema,
  sortDatabaseV1List,
} from "@/utils";
import { useChangelistDetailContext } from "../context";

type LocalState = {
  selectedDatabaseNameList: string[];
  changeType: "DDL" | "DML";
  changeSource: "DATABASE" | "GROUP";
  isGenerating: boolean;
  selectedDatabaseGroup?: string;
};

const { t } = useI18n();
const router = useRouter();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const {
  changelist,
  project,
  showApplyToDatabasePanel: show,
} = useChangelistDetailContext();

const state = reactive<LocalState>({
  selectedDatabaseNameList: [],
  changeType: "DDL",
  changeSource: "DATABASE",
  isGenerating: false,
});

const { ready } = useDatabaseV1List(project.value.name);
const { dbGroupList } = useDBGroupListByProject(project.value.name);

const databaseList = computed(() => {
  let list: ComposedDatabase[] = [];
  list = databaseStore.databaseListByProject(project.value.name);

  list = list.filter(
    (db) => db.syncState == State.ACTIVE && db.project !== DEFAULT_PROJECT_NAME
  );

  return sortDatabaseV1List(list);
});

const schemaDatabaseList = computed(() => {
  if (state.changeType === "DDL") {
    return databaseList.value.filter((db) =>
      instanceV1HasAlterSchema(db.instanceResource)
    );
  }
  return databaseList.value;
});

const schemalessDatabaseList = computed(() => {
  return databaseList.value.filter(
    (db) => !instanceV1HasAlterSchema(db.instanceResource)
  );
});

const flattenSelectedDatabaseNameList = computed(() => {
  return [...state.selectedDatabaseNameList];
});

const nextButtonErrors = computed(() => {
  const errors: string[] = [];
  if (
    (state.changeSource === "DATABASE" &&
      flattenSelectedDatabaseNameList.value.length === 0) ||
    (state.changeSource === "GROUP" && !state.selectedDatabaseGroup)
  ) {
    errors.push(t("changelist.error.select-at-least-one-database"));
  }
  return errors;
});

const handleClickNext = async () => {
  state.isGenerating = true;
  try {
    const databaseList = flattenSelectedDatabaseNameList.value.map((name) =>
      databaseStore.getDatabaseByName(name)
    );
    const databaseGroup = DatabaseGroup.fromPartial({
      ...dbGroupStore.getDBGroupByName(state.selectedDatabaseGroup || ""),
    });
    const query: Record<string, any> = {
      template:
        state.changeType === "DDL"
          ? "bb.issue.database.schema.update"
          : "bb.issue.database.data.update",
      name: generateIssueTitle(
        "bb.issue.database.schema.update",
        state.changeSource === "DATABASE"
          ? databaseList.map((db) => db.databaseName)
          : [databaseGroup?.databasePlaceholder]
      ),
      changelist: changelist.value.name,
      description: `Apply changelist [${changelist.value.description}]`,
    };
    if (state.changeSource === "DATABASE") {
      query.databaseList = databaseList.map((db) => db.name).join(",");
    } else {
      query.databaseGroupName = state.selectedDatabaseGroup;
    }

    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(project.value.name),
        issueSlug: "create",
      },
      query,
    });
  } catch {
    state.isGenerating = false;
  }
};

const reset = () => {
  state.selectedDatabaseNameList = [];
};

watch(
  show,
  (show) => {
    if (show) reset();
  },
  { immediate: true }
);
</script>
