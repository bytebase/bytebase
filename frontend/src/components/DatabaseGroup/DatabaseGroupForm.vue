<template>
  <div class="w-full">
    <FeatureAttentionForInstanceLicense
      v-if="existMatchedUnactivateInstance"
      custom-class="mb-4"
      type="warning"
      feature="bb.feature.database-grouping"
    />
    <div class="w-full grid grid-cols-3 gap-x-6">
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
    </div>
    <NDivider />
    <div class="w-full grid grid-cols-5 gap-x-6">
      <div class="col-span-3">
        <p class="pl-1 text-lg mb-2">
          {{ $t("database-group.condition.self") }}
        </p>
        <ExprEditor
          :expr="state.expr"
          :allow-admin="true"
          :enable-raw-expression="true"
          :factor-list="FactorList"
          :factor-support-dropdown="factorSupportDropdown"
          :factor-options-map="DatabaseGroupFactorOptionsMap(project)"
        />
        <p
          v-if="matchingError"
          class="mt-2 text-sm border border-red-600 px-2 py-1 rounded-lg bg-red-50 text-red-600"
        >
          {{ matchingError }}
        </p>
      </div>
      <div class="col-span-2">
        <MatchedDatabaseView
          :loading="state.isRequesting"
          :matched-database-list="matchedDatabaseList"
          :unmatched-database-list="unmatchedDatabaseList"
        />
      </div>
    </div>
    <NDivider />
    <div class="w-full pl-1">
      <p class="text-lg mb-2">
        {{ $t("common.options") }}
      </p>
      <div>
        <NCheckbox
          v-model:checked="state.multitenancy"
          size="large"
          :label="$t('database-group.multitenancy.self')"
        />
        <p class="text-sm text-gray-400 pl-6 ml-0.5">
          {{ $t("database-group.multitenancy.description") }}
          <LearnMoreLink
            url="https://www.bytebase.com/docs/change-database/batch-change/?source=console#multitenancy"
            class="text-sm"
          />
        </p>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { NCheckbox, NInput, NDivider } from "naive-ui";
import { ClientError, Status } from "nice-grpc-web";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import ExprEditor from "@/components/ExprEditor";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  emptySimpleExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import {
  useDatabaseV1ListByProject,
  useDBGroupStore,
  useSubscriptionV1Store,
} from "@/store";
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
import type { DatabaseGroup } from "@/types/proto/v1/database_group_service";
import { getErrorCode } from "@/utils/grpcweb";
import { FeatureAttentionForInstanceLicense } from "../FeatureGuard";
import { ResourceIdField } from "../v2";
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
  multitenancy: boolean;
};

const { t } = useI18n();
const dbGroupStore = useDBGroupStore();
const subscriptionV1Store = useSubscriptionV1Store();
const state = reactive<LocalState>({
  isRequesting: false,
  resourceId: "",
  placeholder: "",
  expr: wrapAsGroup(emptySimpleExpr()),
  multitenancy: false,
});
const { databaseList } = useDatabaseV1ListByProject(props.project.name);
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
  state.multitenancy = composedDatabaseGroup.multitenancy;
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

const matchingError = ref<string | undefined>(undefined);
const matchedDatabaseList = ref<ComposedDatabase[]>([]);
const unmatchedDatabaseList = ref<ComposedDatabase[]>([]);
const updateDatabaseMatchingState = useDebounceFn(async () => {
  if (!validateSimpleExpr(state.expr)) {
    matchingError.value = undefined;
    matchedDatabaseList.value = [];
    unmatchedDatabaseList.value = databaseList.value;
    return;
  }

  state.isRequesting = true;
  try {
    const result = await dbGroupStore.fetchDatabaseGroupMatchList({
      projectName: props.project.name,
      expr: state.expr,
    });

    matchingError.value = undefined;
    matchedDatabaseList.value = result.matchedDatabaseList;
    unmatchedDatabaseList.value = result.unmatchedDatabaseList;
  } catch (error) {
    matchingError.value = (error as ClientError).details;
    matchedDatabaseList.value = [];
    unmatchedDatabaseList.value = databaseList.value;
  }
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
        database.instanceResource
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
