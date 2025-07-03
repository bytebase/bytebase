<template>
  <Drawer v-model:show="show">
    <DrawerContent :title="$t('changelist.apply-to-database')">
      <template #default>
        <div
          class="w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
        >
          <div class="flex flex-col">
            <div class="mb-4 flex flex-row justify-start items-center gap-2">
              <span class="textlabel">{{ t("changelist.change-type") }}:</span>
              <NRadioGroup v-model:value="state.changeType">
                <NRadio :value="'DDL'" :label="t('issue.title.edit-schema')" />
                <NRadio :value="'DML'" :label="t('issue.title.change-data')" />
              </NRadioGroup>
            </div>
            <DatabaseAndGroupSelector
              :project="project"
              v-model:value="state.targetSelectState"
            />
          </div>
          <div
            v-if="state.isGenerating"
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
                state.targetSelectState?.changeSource === 'DATABASE' &&
                state.targetSelectState?.selectedDatabaseNameList.length > 0
              "
              class="textinfolabel"
            >
              {{
                $t("database.selected-n-databases", {
                  n: state.targetSelectState?.selectedDatabaseNameList.length,
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
import { NRadioGroup, NRadio } from "naive-ui";
import { zindexable as vZindexable } from "vdirs";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter, type LocationQueryRaw } from "vue-router";
import { BBSpin } from "@/bbkit";
import DatabaseAndGroupSelector, {
  type DatabaseSelectState,
} from "@/components/DatabaseAndGroupSelector/";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";
import { useChangelistDetailContext } from "../context";

type LocalState = {
  changeType: "DDL" | "DML";
  isGenerating: boolean;
  targetSelectState?: DatabaseSelectState;
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
  changeType: "DDL",
  isGenerating: false,
});

const nextButtonErrors = computed(() => {
  const errors: string[] = [];
  if (
    !state.targetSelectState ||
    (state.targetSelectState.changeSource === "DATABASE" &&
      state.targetSelectState.selectedDatabaseNameList.length === 0) ||
    (state.targetSelectState.changeSource === "GROUP" &&
      !state.targetSelectState.selectedDatabaseGroup)
  ) {
    errors.push("Please select at least one database");
  }
  return errors;
});

const handleClickNext = async () => {
  if (!state.targetSelectState) {
    return;
  }

  state.isGenerating = true;
  try {
    const databaseList = state.targetSelectState.selectedDatabaseNameList.map(
      (name) => databaseStore.getDatabaseByName(name)
    );
    const databaseGroup = dbGroupStore.getDBGroupByName(
      state.targetSelectState.selectedDatabaseGroup || ""
    );
    const query: LocationQueryRaw = {
      template:
        state.changeType === "DDL"
          ? "bb.issue.database.schema.update"
          : "bb.issue.database.data.update",
      name: generateIssueTitle(
        "bb.issue.database.schema.update",
        state.targetSelectState.changeSource === "DATABASE"
          ? databaseList.map((db) => db.databaseName)
          : [databaseGroup?.title ?? ""]
      ),
      changelist: changelist.value.name,
      description: `Apply changelist [${changelist.value.description}]`,
    };
    if (state.targetSelectState.changeSource === "DATABASE") {
      query.databaseList = databaseList.map((db) => db.name).join(",");
    } else {
      query.databaseGroupName = state.targetSelectState.selectedDatabaseGroup;
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
</script>
