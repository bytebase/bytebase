<template>
  <div class="w-full">
    <FeatureAttentionForInstanceLicense
      v-if="existMatchedUnactivateInstance"
      custom-class="mb-4"
      type="warning"
      feature="bb.feature.database-grouping"
    />
    <div class="w-full grid grid-cols-3 gap-x-6 pb-6 mb-4 border-b">
      <div>
        <p class="text-lg mb-2">{{ $t("common.name") }}</p>
        <NInput v-model:value="state.placeholder" />
        <div class="mt-2">
          <ResourceIdField
            ref="resourceIdField"
            editing-class="mt-4"
            resource-type="database-group"
            :readonly="!isCreating"
            :value="state.resourceId"
            :resource-title="state.placeholder"
            :validate="validateResourceId"
          />
        </div>
      </div>
      <div>
        <p class="text-lg mb-2">{{ $t("common.project") }}</p>
        <ProjectSelect
          :project="project.uid"
          :disabled="true"
          style="width: auto"
        />
      </div>
    </div>
    <div class="w-full grid grid-cols-5 gap-x-6">
      <div class="col-span-3">
        <p class="pl-1 text-lg mb-2">
          {{ $t("database-group.condition.self") }}
        </p>
        <ExprEditor
          :expr="state.expr"
          :allow-admin="true"
          :factor-list="FactorList"
          :factor-support-dropdown="factorSupportDropdown"
          :factor-options-map="DatabaseGroupFactorOptionsMap()"
        />
      </div>
      <div class="col-span-2">
        <MatchedDatabaseView
          :loading="state.isRequesting"
          :matched-database-list="matchedDatabaseList"
          :unmatched-database-list="unmatchedDatabaseList"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { NInput } from "naive-ui";
import { Status } from "nice-grpc-web";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import ExprEditor from "@/components/ExprEditor";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { emptySimpleExpr, wrapAsGroup } from "@/plugins/cel";
import { useDBGroupStore, useSubscriptionV1Store } from "@/store";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
} from "@/store/modules/v1/common";
import type {
  ComposedDatabase,
  ComposedDatabaseGroup,
  ComposedProject,
  ResourceId,
  ValidatedMessage,
} from "@/types";
import type { DatabaseGroup } from "@/types/proto/v1/project_service";
import { getErrorCode } from "@/utils/grpcweb";
import { ProjectSelect, ResourceIdField } from "../v2";
import MatchedDatabaseView from "./MatchedDatabaseView.vue";
import { factorSupportDropdown, DatabaseGroupFactorOptionsMap } from "./utils";
import { FactorList } from "./utils";

const props = defineProps<{
  project: ComposedProject;
  databaseGroup?: DatabaseGroup;
  parentDatabaseGroup?: ComposedDatabaseGroup;
}>();

type LocalState = {
  isRequesting: boolean;
  resourceId: string;
  placeholder: string;
  selectedDatabaseGroupId?: string;
  expr: ConditionGroupExpr;
};

const { t } = useI18n();
const dbGroupStore = useDBGroupStore();
const subscriptionV1Store = useSubscriptionV1Store();
const state = reactive<LocalState>({
  isRequesting: false,
  resourceId: "",
  placeholder: "",
  expr: wrapAsGroup(emptySimpleExpr()),
});
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

const isCreating = computed(() => props.databaseGroup === undefined);

onMounted(async () => {
  const databaseGroup = props.databaseGroup;
  if (!databaseGroup) {
    return;
  }

  const databaseGroupEntity = databaseGroup as DatabaseGroup;
  const [, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    databaseGroup.name
  );
  state.resourceId = databaseGroupName;
  state.placeholder = databaseGroupEntity.databasePlaceholder;
  const composedDatabaseGroup = await dbGroupStore.getOrFetchDBGroupByName(
    databaseGroup.name,
    { silent: true }
  );
  if (composedDatabaseGroup.simpleExpr) {
    state.expr = cloneDeep(composedDatabaseGroup.simpleExpr);
  }
});

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  const request = dbGroupStore.getOrFetchDBGroupByName(
    `${props.project.name}/${databaseGroupNamePrefix}${resourceId}`,
    { silent: true }
  );

  if (!request) {
    return [];
  }

  try {
    const data = await request;
    if (data) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t(`resource.database-group`),
          }),
        },
      ];
    }
  } catch (error) {
    if (getErrorCode(error) !== Status.NOT_FOUND) {
      throw error;
    }
  }

  return [];
};

const matchedDatabaseList = ref<ComposedDatabase[]>([]);
const unmatchedDatabaseList = ref<ComposedDatabase[]>([]);
const updateDatabaseMatchingState = useDebounceFn(async () => {
  state.isRequesting = true;
  const result = await dbGroupStore.fetchDatabaseGroupMatchList({
    projectName: props.project.name,
    expr: state.expr,
  });

  matchedDatabaseList.value = result.matchedDatabaseList;
  unmatchedDatabaseList.value = result.unmatchedDatabaseList;
  state.isRequesting = false;
}, 500);

watch(
  [() => props.project.name, () => state.expr],
  updateDatabaseMatchingState,
  {
    immediate: true,
    deep: true,
  }
);

const existMatchedUnactivateInstance = computed(() => {
  return matchedDatabaseList.value.some(
    (database) =>
      !subscriptionV1Store.hasInstanceFeature(
        "bb.feature.database-grouping",
        database.instanceEntity
      )
  );
});

defineExpose({
  getFormState: () => {
    return {
      ...state,
      resourceId: resourceIdField.value?.resourceId || "",
      existMatchedUnactivateInstance: existMatchedUnactivateInstance.value,
    };
  },
});
</script>
