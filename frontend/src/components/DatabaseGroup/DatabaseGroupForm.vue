<template>
  <FormLayout :title="title">
    <template #body>
      <div>
        <FeatureAttentionForInstanceLicense
          v-if="existMatchedUnactivateInstance"
          custom-class="mb-4"
          type="warning"
          feature="bb.feature.database-grouping"
        />
        <div class="w-full grid grid-cols-3 gap-x-6">
          <div>
            <p class="font-medium text-main mb-2">{{ $t("common.name") }}</p>
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
            <p class="pl-1 font-medium text-main mb-2">
              {{ $t("database-group.condition.self") }}
            </p>
            <ExprEditor
              :expr="state.expr"
              :allow-admin="true"
              :enable-raw-expression="true"
              :factor-list="FactorList"
              :factor-support-dropdown="factorSupportDropdown"
              :option-config-map="getDatabaseGroupOptionConfigMap(project)"
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
          <p class="font-medium text-main mb-2">
            {{ $t("common.options") }}
          </p>
          <div>
            <NCheckbox
              v-model:checked="state.multitenancy"
              size="medium"
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
    <template #footer>
      <div class="w-full flex justify-between items-center">
        <div>
          <NButton v-if="!isCreating" text @click="doDelete">
            <template #icon>
              <Trash2Icon class="w-4 h-auto" />
            </template>
            {{ $t("common.delete") }}
          </NButton>
        </div>
        <div class="flex flex-row justify-end items-center gap-x-2">
          <NButton @click="$emit('dismiss')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" :disabled="!allowConfirm" @click="doConfirm">
            {{ isCreating ? $t("common.save") : $t("common.confirm") }}
          </NButton>
        </div>
      </div>
    </template>
  </FormLayout>
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { cloneDeep, head, isEqual } from "lodash-es";
import { Trash2Icon } from "lucide-vue-next";
import { NCheckbox, NButton, NInput, NDivider, useDialog } from "naive-ui";
import { ClientError, Status } from "nice-grpc-web";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ExprEditor from "@/components/ExprEditor";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import FormLayout from "@/components/v2/Form/FormLayout.vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  emptySimpleExpr,
  validateSimpleExpr,
  buildCELExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import {
  PROJECT_V1_ROUTE_DATABASE_GROUPS,
  PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  useDBGroupStore,
  useSubscriptionV1Store,
  pushNotification,
} from "@/store";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
} from "@/store/modules/v1/common";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import type {
  ComposedDatabase,
  ComposedProject,
  ResourceId,
  ValidatedMessage,
} from "@/types";
import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import type { DatabaseGroup } from "@/types/proto/v1/database_group_service";
import { batchConvertParsedExprToCELString } from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import { FeatureAttentionForInstanceLicense } from "../FeatureGuard";
import { ResourceIdField } from "../v2";
import MatchedDatabaseView from "./MatchedDatabaseView.vue";
import {
  factorSupportDropdown,
  getDatabaseGroupOptionConfigMap,
} from "./utils";
import { FactorList } from "./utils";

const props = defineProps<{
  project: ComposedProject;
  databaseGroup?: DatabaseGroup;
  title?: string;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "created", databaseGroupName: string): void;
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
const { databaseList } = useDatabaseV1List(props.project.name);
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();
const router = useRouter();
const dialog = useDialog();

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

const doDelete = () => {
  dialog.error({
    title: "Confirm to delete",
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      const databaseGroup = props.databaseGroup as DatabaseGroup;
      await dbGroupStore.deleteDatabaseGroup(databaseGroup.name);
      if (
        router.currentRoute.value.name ===
        PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL
      ) {
        router.replace({
          name: PROJECT_V1_ROUTE_DATABASE_GROUPS,
        });
      }
      emit("dismiss");
    },
  });
};

const allowConfirm = computed(() => {
  if (existMatchedUnactivateInstance.value) {
    return false;
  }
  return (
    resourceIdField.value?.resourceId &&
    state.placeholder &&
    validateSimpleExpr(state.expr)
  );
});

const doConfirm = async () => {
  const formState = {
    ...state,
    resourceId: resourceIdField.value?.resourceId || "",
    existMatchedUnactivateInstance: existMatchedUnactivateInstance.value,
  };
  if (!formState || !allowConfirm.value) {
    return;
  }

  let celExpr: CELExpr | undefined = undefined;
  try {
    celExpr = await buildCELExpr(formState.expr);
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `CEL expression error occurred`,
      description: (error as Error).message,
    });
    return;
  }

  const celStrings = await batchConvertParsedExprToCELString([celExpr!]);
  const celString = head(celStrings);
  if (!celString) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `CEL expression error occurred`,
      description: "CEL expression is empty",
    });
    return;
  }

  if (isCreating.value) {
    const resourceId = formState.resourceId;
    await dbGroupStore.createDatabaseGroup({
      projectName: props.project.name,
      databaseGroup: {
        name: `${props.project.name}/databaseGroups/${resourceId}`,
        databasePlaceholder: formState.placeholder,
        databaseExpr: Expr.fromPartial({
          expression: celString,
        }),
        multitenancy: formState.multitenancy,
      },
      databaseGroupId: resourceId,
    });
    emit("created", resourceId);
  } else {
    if (!props.databaseGroup) {
      return;
    }

    const updateMask: string[] = [];
    if (
      !isEqual(props.databaseGroup.databasePlaceholder, formState.placeholder)
    ) {
      updateMask.push("database_placeholder");
    }
    if (
      !isEqual(
        props.databaseGroup.databaseExpr,
        Expr.fromPartial({
          expression: celString,
        })
      )
    ) {
      updateMask.push("database_expr");
    }
    if (!isEqual(props.databaseGroup.multitenancy, formState.multitenancy)) {
      updateMask.push("multitenancy");
    }
    await dbGroupStore.updateDatabaseGroup(
      {
        ...props.databaseGroup!,
        databasePlaceholder: formState.placeholder,
        databaseExpr: Expr.fromPartial({
          expression: celString,
        }),
        multitenancy: formState.multitenancy,
      },
      updateMask
    );
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: isCreating.value ? t("common.created") : t("common.updated"),
  });
  emit("dismiss");
};

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
