<template>
  <div class="w-full flex flex-row flex-wrap gap-2">
    <p class="w-full">
      {{ $t("release.tasks-to-apply") }}
    </p>
    <div
      v-for="spec in flattenSpecList"
      :key="spec.id"
      class="max-w-52 flex flex-row items-center flex-wrap gap-1 border px-2 py-1 rounded-md"
    >
      <div class="w-full truncate space-x-1">
        <NTag round :size="'small'">
          {{ specReleaseVersion(spec) }}
        </NTag>
        <span class="text-sm">
          {{ extractFilename(spec.specReleaseSource?.file || "") }}
        </span>
      </div>
      <DatabaseView
        class="text-sm"
        :database="databaseForPlanSpec(spec)"
        :link="false"
        :show-not-found="true"
      />
    </div>
  </div>
  <div v-if="previewPlanResult.outOfOrderFiles.length > 0">
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
              :database="databaseStore.getDatabaseByName(temp.database)"
              :link="false"
              :show-not-found="true"
            />
          </td>
          <td>
            <div class="flex flex-row items-center gap-2">
              <span v-for="file in temp.files" :key="file">
                {{ extractFilename(file) }}
              </span>
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
          <th>Files</th>
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
              :database="databaseStore.getDatabaseByName(temp.database)"
              :link="false"
              :show-not-found="true"
            />
          </td>
          <td>
            <div class="flex flex-row items-center gap-2">
              <span v-for="file in temp.files" :key="file">
                {{ extractFilename(file) }}
              </span>
            </div>
          </td>
        </tr>
      </tbody>
    </NTable>
  </div>
</template>

<script lang="ts" setup>
import { unescape } from "lodash-es";
import { NTag, NTable } from "naive-ui";
import { computed } from "vue";
import DatabaseView from "@/components/v2/Model/DatabaseView.vue";
import { useDatabaseV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  Plan_Spec,
  PreviewPlanResponse,
} from "@/types/proto/v1/plan_service";

const props = defineProps<{
  previewPlanResult: PreviewPlanResponse;
}>();

const databaseStore = useDatabaseV1Store();

const flattenSpecList = computed((): Plan_Spec[] => {
  return (
    props.previewPlanResult.plan?.steps.flatMap((step) => {
      return step.specs;
    }) || []
  );
});

const databaseForPlanSpec = (spec: Plan_Spec): ComposedDatabase => {
  return databaseStore.getDatabaseByName(spec.changeDatabaseConfig!.target);
};

const specReleaseVersion = (spec: Plan_Spec): string => {
  return spec.changeDatabaseConfig!.schemaVersion;
};

const extractFilename = (file: string): string => {
  const pattern = /(?:^|\/)files\/(.+)(?:$|\/)/;
  const matches = file.match(pattern);
  if (!matches) {
    return "";
  }
  return unescape(matches[1]);
};
</script>
