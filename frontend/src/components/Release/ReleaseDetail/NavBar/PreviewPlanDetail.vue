<template>
  <NAlert v-if="shouldShowAppliedAlert" type="info">{{
    $t(
      "release.messages.bytebase-will-skip-those-files-has-been-applied-on-target-database"
    )
  }}</NAlert>
  <div class="w-full flex flex-row flex-wrap gap-2">
    <p class="w-full">
      {{ $t("release.tasks-to-apply") }}
    </p>
    <div
      v-for="spec in specList"
      :key="spec.id"
      class="max-w-52 flex flex-row items-center flex-wrap border px-2 py-1 rounded-md"
    >
      <DatabaseView
        class="text-sm"
        :database="spec.changeDatabaseConfig!.target"
        :link="false"
        :show-not-found="true"
      />
      <div class="w-full truncate space-x-1">
        <NTag round :size="'small'">
          "TODO(d): version"
        </NTag>
        <span class="text-sm">
          {{
            spec.specReleaseSource?.file
              ? getReleaseFileStatement(
                  getFileByName(spec.specReleaseSource.file)!
                )
              : "-"
          }}
        </span>
      </div>
    </div>
    <p v-if="specList.length === 0" class="text-gray-400 italic">
      {{ $t("release.no-tasks-to-apply.self") }}
    </p>
  </div>
  <div v-if="previewPlanResult.outOfOrderFiles.length > 0 && !allowOutOfOrder">
    <p>
      {{ $t("release.out-of-order-files") }}
    </p>
    <NTable class="mt-2" size="small">
      <thead>
        <tr>
          <th class="w-64">
            {{ $t("common.database") }}
          </th>
          <th>{{ $t("release.files") }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="temp in previewPlanResult.outOfOrderFiles"
          :key="temp.database"
        >
          <td>
            <DatabaseView
              class="text-sm"
              :database="temp.database"
              :link="false"
              :show-not-found="true"
            />
          </td>
          <td>
            <div class="flex flex-row items-center gap-2">
              <NTag
                v-for="file in temp.files"
                :key="file"
                round
                :size="'small'"
              >
                {{ getFileByName(file)?.version }}
              </NTag>
            </div>
          </td>
        </tr>
      </tbody>
    </NTable>
  </div>
  <div v-if="previewPlanResult.appliedButModifiedFiles.length > 0">
    <p>
      {{ $t("release.applied-but-modifed-files") }}
    </p>
    <NTable class="mt-2" size="small">
      <thead>
        <tr>
          <th class="w-64">
            {{ $t("common.database") }}
          </th>
          <th>{{ $t("release.files") }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="temp in previewPlanResult.appliedButModifiedFiles"
          :key="temp.database"
        >
          <td>
            <DatabaseView
              class="text-sm"
              :database="temp.database"
              :link="false"
              :show-not-found="true"
            />
          </td>
          <td>
            <div class="flex flex-row items-center gap-2">
              <NTag
                v-for="file in temp.files"
                :key="file"
                round
                :size="'small'"
              >
                {{ getFileByName(file)?.version }}
              </NTag>
            </div>
          </td>
        </tr>
      </tbody>
    </NTable>
  </div>
</template>

<script lang="ts" setup>
import { NTag, NTable, NAlert } from "naive-ui";
import { computed } from "vue";
import type { DatabaseSelectState } from "@/components/DatabaseAndGroupSelector";
import DatabaseView from "@/components/v2/Model/DatabaseView.vue";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import type { ComposedRelease } from "@/types";
import type {
  Plan_Spec,
  PreviewPlanResponse,
} from "@/types/proto/v1/plan_service";
import { getReleaseFileStatement } from "@/utils";

const props = defineProps<{
  release: ComposedRelease;
  previewPlanResult: PreviewPlanResponse;
  databaseSelectState?: DatabaseSelectState;
  allowOutOfOrder?: boolean;
}>();

const databaseStore = useDatabaseV1Store();

const databaseGroupStore = useDBGroupStore();

const specList = computed((): Plan_Spec[] => {
  return props.previewPlanResult.plan?.specs || [];
});

const targetDatabases = computed(() => {
  if (!props.databaseSelectState) {
    return [];
  }
  if (props.databaseSelectState.changeSource === "DATABASE") {
    return props.databaseSelectState.selectedDatabaseNameList.map((name) =>
      databaseStore.getDatabaseByName(name)
    );
  } else {
    return (
      databaseGroupStore.getDBGroupByName(
        props.databaseSelectState.selectedDatabaseNameList[0]
      )?.matchedDatabases || []
    );
  }
});

// Should show the alert when the number of files to apply is greater than the number of specs in the preview plan.
const shouldShowAppliedAlert = computed(() => {
  return (
    props.release.files.length * targetDatabases.value.length >
    specList.value.length
  );
});

const getFileByName = (name: string) => {
  return props.release.files.find((file) => name.endsWith(`files/${file.id}`));
};
</script>
