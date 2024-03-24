<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="space-y-4">
    <div>
      <div v-if="getWebhookLink !== ''" class="mb-2">
        <label class="textlabel mt-2">
          <i18n-t keypath="repository.our-webhook-link">
            <template #webhookLink>
              <a class="normal-link" :href="getWebhookLink" target="_blank">{{
                getWebhookLink
              }}</a>
            </template>
          </i18n-t>
        </label>
      </div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="gitprovider" class="textlabel">
          {{ $t("repository.git-provider") }}
        </label>
        <VCSIcon :type="vcsType" />
      </div>
      <BBTextField
        id="gitprovider"
        name="gitprovider"
        class="mt-2 w-full"
        :disabled="true"
        :value="vcsName"
      />
    </div>
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="repository" class="textlabel">
          {{ $t("common.repository") }}
        </label>
        <div
          v-if="!create && allowEdit"
          class="ml-1 normal-link text-sm"
          @click.prevent="$emit('change-repository')"
        >
          {{ $t("common.change") }}
        </div>
      </div>
      <BBTextField
        id="repository"
        name="repository"
        class="mt-2 w-full"
        :disabled="true"
        :value="repositoryInfo.fullPath"
      />
    </div>
    <div>
      <div class="textlabel">
        {{ $t("common.branch") }} <span class="text-red-600">*</span>
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.branch-observe-file-change") }}
      </div>
      <BBTextField
        id="branch"
        v-model:value="repositoryConfig.branchFilter"
        name="branch"
        class="mt-2 w-full"
        placeholder="e.g. main"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <div class="textlabel">{{ $t("repository.base-directory") }}</div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.base-directory-description") }}
      </div>
      <BBTextField
        id="basedirectory"
        v-model:value="repositoryConfig.baseDirectory"
        name="basedirectory"
        class="mt-2 w-full"
        :disabled="!allowEdit"
      />
    </div>
    <!-- Project schemaChangeType selector -->
    <div>
      <div class="textlabel">
        {{ $t("project.settings.schema-change-type") }}
        <span class="text-red-600">*</span>
        <LearnMoreLink
          url="https://www.bytebase.com/docs/change-database/state-based-migration?source=console"
          class="ml-1"
        />
      </div>
      <NSelect
        :options="schemaChangeTypeOptions"
        :value="schemaChangeType"
        :render-label="renderLabel"
        class="mt-1"
        @update:value="$emit('change-schema-change-type', $event)"
      />
    </div>
    <div>
      <div class="textlabel">
        {{ $t("repository.file-path-template") }}
        <span class="text-red-600">*</span>
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.file-path-template-description") }}
        <LearnMoreLink
          url="https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files?source=console#file-path-template"
          class="ml-1"
        />
      </div>
      <BBTextField
        id="filepathtemplate"
        v-model:value="repositoryConfig.filePathTemplate"
        name="filepathtemplate"
        class="mt-2 w-full"
        :disabled="!allowEdit"
      />
      <div class="mt-2 textinfolabel capitalize">
        <span class="text-red-600">*</span>
        {{ $t("common.required-placeholder") }}:
        {{ FILE_REQUIRED_PLACEHOLDER }};
        <template v-if="fileOptionalPlaceholder.length > 0">
          {{ $t("common.optional-placeholder") }}:
          {{ fileOptionalPlaceholder.join(", ") }};
        </template>
        {{ $t("common.optional-directory-wildcard") }}:
        {{ FILE_OPTIONAL_DIRECTORY_WILDCARD }}
      </div>
      <div v-if="isProjectSchemaChangeTypeDDL" class="mt-2 textinfolabel">
        • {{ $t("repository.file-path-example-schema-migration") }}:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "ddl"
          )
        }}
      </div>
      <div class="mt-2 textinfolabel">
        • {{ $t("repository.file-path-example-data-migration") }}:
        {{
          sampleFilePath(
            repositoryConfig.baseDirectory,
            repositoryConfig.filePathTemplate,
            "dml"
          )
        }}
      </div>
    </div>
    <div v-if="canEnableSQLReview">
      <div class="textlabel flex gap-x-1">
        {{ $t("repository.sql-review-ci") }}
        <FeatureBadge feature="bb.feature.vcs-sql-review" />
      </div>
      <div class="mt-1 textinfolabel">
        {{
          $t("repository.sql-review-ci-description", {
            pr:
              vcsType === VCSProvider_Type.GITLAB
                ? $t("repository.merge-request")
                : $t("repository.pull-request"),
            pathTemplate: $t("repository.file-path-template"),
          })
        }}
      </div>
      <BBAttention
        v-if="
          instanceWithoutLicense.length > 0 &&
          subscriptionStore.currentPlan !== PlanType.FREE &&
          hasFeature('bb.feature.vcs-sql-review')
        "
        class="my-4"
        type="warning"
        :title="$t('subscription.features.bb-feature-vcs-sql-review.title')"
        :description="
          $t('subscription.instance-assignment.missing-license-for-instances', {
            count: instanceWithoutLicense.length,
            name: instanceWithoutLicense.map((ins) => ins.title).join(','),
          })
        "
        :action-text="
          canManageInstanceLicense
            ? $t('subscription.instance-assignment.assign-license')
            : ''
        "
        @click="state.showInstanceAssignmentDrawer = true"
      />
      <div class="flex space-x-4 mt-2">
        <NCheckbox
          :disabled="!allowEdit"
          :label="enableSQLReviewTitle"
          :checked="repositoryConfig.enableSQLReviewCI"
          @update:checked="(on: boolean) => {
            repositoryConfig.enableSQLReviewCI = on;
            onSQLReviewCIToggle(on);
          }"
        />
      </div>
    </div>
    <FeatureModal
      feature="bb.feature.vcs-sql-review"
      :open="state.showFeatureModal"
      @cancel="state.showFeatureModal = false"
    />
  </div>
  <InstanceAssignment
    :show="state.showInstanceAssignmentDrawer"
    @dismiss="state.showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { NCheckbox, NSelect, SelectOption } from "naive-ui";
import { reactive, computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { BBBetaBadge } from "@/bbkit";
import {
  hasFeature,
  useSubscriptionV1Store,
  useDatabaseV1Store,
  useCurrentUserV1,
} from "@/store";
import { ExternalRepositoryInfo, RepositoryConfig } from "@/types";
import {
  Project,
  TenantMode,
  SchemaChange,
  schemaChangeToJSON,
} from "@/types/proto/v1/project_service";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";
import { hasWorkspacePermissionV2, supportSQLReviewCI } from "@/utils";

const FILE_REQUIRED_PLACEHOLDER = "{{DB_NAME}}, {{VERSION}}, {{TYPE}}";
const FILE_OPTIONAL_DIRECTORY_WILDCARD = "*, **";
const SINGLE_ASTERISK_REGEX = /\/\*\//g;
const DOUBLE_ASTERISKS_REGEX = /\/\*\*\//g;

interface LocalState {
  showFeatureModal: boolean;
  showInstanceAssignmentDrawer: boolean;
}

defineEmits<{
  (event: "change-repository"): void;
  (event: "change-schema-change-type", changeType: SchemaChange): void;
}>();

const props = withDefaults(
  defineProps<{
    allowEdit?: boolean;
    create?: boolean;
    vcsType: VCSProvider_Type;
    vcsName: string;
    repositoryInfo: ExternalRepositoryInfo;
    repositoryConfig: RepositoryConfig;
    project: Project;
    schemaChangeType: SchemaChange;
  }>(),
  {
    allowEdit: true,
    create: false,
  }
);

const { t } = useI18n();

const state = reactive<LocalState>({
  showFeatureModal: false,
  showInstanceAssignmentDrawer: false,
});

const subscriptionStore = useSubscriptionV1Store();

const databaseV1List = computed(() => {
  return useDatabaseV1Store().databaseListByProject(props.project.name);
});

const canManageInstanceLicense = computed((): boolean => {
  return hasWorkspacePermissionV2(
    useCurrentUserV1().value,
    "bb.instances.update"
  );
});

const instanceWithoutLicense = computed(() => {
  return databaseV1List.value
    .map((db) => db.instanceEntity)
    .filter((ins) => !ins.activation);
});

const isTenantProject = computed(() => {
  return props.project.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});
const isProjectSchemaChangeTypeDDL = computed(() => {
  return (props.schemaChangeType || SchemaChange.DDL) === SchemaChange.DDL;
});
const canEnableSQLReview = computed(() => {
  return supportSQLReviewCI(props.vcsType);
});
const enableSQLReviewTitle = computed(() => {
  switch (props.vcsType) {
    case VCSProvider_Type.GITLAB:
      return t("repository.sql-review-ci-enable-gitlab");
    case VCSProvider_Type.GITHUB:
      return t("repository.sql-review-ci-enable-github");
    case VCSProvider_Type.AZURE_DEVOPS:
      return t("repository.sql-review-ci-enable-azure");
    default:
      return t("repository.sql-review-ci-enable-title");
  }
});

const sampleFilePath = (
  baseDirectory: string,
  filePathTemplate: string,
  type: string
): string => {
  type Item = {
    placeholder: string;
    sampleText: string;
  };
  const placeholderList: Item[] = [
    {
      placeholder: "{{VERSION}}",
      sampleText: "202101131000",
    },
    {
      placeholder: "{{DB_NAME}}",
      sampleText: "db1",
    },
    {
      placeholder: "{{TYPE}}",
      sampleText: type,
    },
    {
      placeholder: "{{ENV_ID}}",
      sampleText: "env1",
    },
    {
      placeholder: "{{ENV_NAME}}", // for legacy support
      sampleText: "env1",
    },
    {
      placeholder: "{{DESCRIPTION}}",
      sampleText: "create_tablefoo_for_bar",
    },
  ];
  let result = `${baseDirectory}/${filePathTemplate}`;
  // To replace the wildcard.
  result = result.replace(SINGLE_ASTERISK_REGEX, "/foo/");
  result = result.replace(DOUBLE_ASTERISKS_REGEX, "/foo/bar/");
  for (const item of placeholderList) {
    const re = new RegExp(item.placeholder, "g");
    result = result.replace(re, item.sampleText);
  }
  return result;
};

const getWebhookLink = computed(() => {
  if (props.vcsType === VCSProvider_Type.AZURE_DEVOPS) {
    const parts = props.repositoryInfo.externalId.split("/");
    if (parts.length !== 3) {
      return "";
    }
    const [organization, project, _] = parts;
    return `https://dev.azure.com/${organization}/${project}/_settings/serviceHooks`;
  }
  return "";
});

const fileOptionalPlaceholder = computed(() => {
  const tags = [] as string[];
  // Only allows {{ENV_ID}} to be an optional placeholder for non-tenant mode projects
  if (!isTenantProject.value) tags.push("{{ENV_ID}}");
  tags.push("{{DESCRIPTION}}");
  return tags;
});

const onSQLReviewCIToggle = (on: boolean) => {
  if (on && !hasFeature("bb.feature.vcs-sql-review")) {
    state.showFeatureModal = true;
  }
};

const renderLabel = (option: SelectOption) => {
  const value = option.value as SchemaChange;
  const child = [
    h(
      "span",
      {},
      t(
        `project.settings.select-schema-change-type-${schemaChangeToJSON(
          value
        ).toLowerCase()}`
      )
    ),
  ];
  if (value === SchemaChange.SDL) {
    child.push(h(BBBetaBadge, { class: "!leading-3" }));
  }

  return h("div", { class: "flex items-center gap-x-2" }, child);
};

const schemaChangeTypeOptions = computed(() => {
  return [SchemaChange.DDL, SchemaChange.SDL].map((val) => ({
    value: val,
    label: schemaChangeToJSON(val),
  }));
});
</script>
