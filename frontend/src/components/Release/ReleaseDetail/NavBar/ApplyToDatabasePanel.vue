<template>
  <Drawer v-bind="$attrs" @close="emit('close')">
    <DrawerContent :title="'Apply to databases'">
      <template #default>
        <div
          class="w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
        >
          <DatabaseAndGroupSelector
            :project="project"
            @update="handleTargetChange"
          />
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
            <NButton @click.prevent="emit('close')">
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
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import DatabaseAndGroupSelector, {
  type DatabaseSelectState,
} from "@/components/DatabaseAndGroupSelector/";
import { Drawer, DrawerContent, ErrorTipsButton } from "@/components/v2";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { DatabaseGroup } from "@/types/proto/v1/database_group_service";
import { extractProjectResourceName, generateIssueTitle } from "@/utils";
import { useReleaseDetailContext } from "../context";

const emit = defineEmits<{
  (event: "close"): void;
}>();

type LocalState = {
  isGenerating: boolean;
  targetSelectState?: DatabaseSelectState;
};

const router = useRouter();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const { release, project } = useReleaseDetailContext();

const state = reactive<LocalState>({
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

const handleTargetChange = (databaseSelectState: DatabaseSelectState) => {
  state.targetSelectState = databaseSelectState;
};

const handleClickNext = async () => {
  if (!state.targetSelectState) {
    return;
  }

  state.isGenerating = true;
  try {
    const databaseList = state.targetSelectState.selectedDatabaseNameList.map(
      (name) => databaseStore.getDatabaseByName(name)
    );
    const databaseGroup = DatabaseGroup.fromPartial({
      ...dbGroupStore.getDBGroupByName(
        state.targetSelectState.selectedDatabaseGroup || ""
      ),
    });
    const changeType = "bb.issue.database.schema.update";
    const query: Record<string, any> = {
      template: changeType,
      name: generateIssueTitle(
        changeType,
        state.targetSelectState.changeSource === "DATABASE"
          ? databaseList.map((db) => db.databaseName)
          : [databaseGroup?.databasePlaceholder]
      ),
      release: release.value.name,
      description: `Apply release "${release.value.title}"`,
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
