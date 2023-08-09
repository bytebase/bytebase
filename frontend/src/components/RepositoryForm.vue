<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="space-y-4">
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="gitprovider" class="textlabel">
          {{ $t("repository.git-provider") }}
        </label>
        <template v-if="vcsType === ExternalVersionControl_Type.GITLAB">
          <img class="h-4 w-auto" src="../assets/gitlab-logo.svg" />
        </template>
        <template v-if="vcsType === ExternalVersionControl_Type.GITHUB">
          <img class="h-4 w-auto" src="../assets/github-logo.svg" />
        </template>
        <template v-if="vcsType === ExternalVersionControl_Type.BITBUCKET">
          <img class="h-4 w-auto" src="../assets/bitbucket-logo.svg" />
        </template>
      </div>
      <input
        id="gitprovider"
        name="gitprovider"
        type="text"
        class="textfield mt-1 w-full"
        disabled="true"
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
      <input
        id="repository"
        name="repository"
        type="text"
        class="textfield mt-1 w-full"
        disabled="true"
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
      <input
        id="branch"
        v-model="repositoryConfig.branchFilter"
        name="branch"
        type="text"
        class="textfield mt-2 w-full"
        placeholder="e.g. main"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <div class="textlabel">{{ $t("repository.base-directory") }}</div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.base-directory-description") }}
      </div>
      <input
        id="basedirectory"
        v-model="repositoryConfig.baseDirectory"
        name="basedirectory"
        type="text"
        class="textfield mt-2 w-full"
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
      <BBSelect
        id="schemamigrationtype"
        :disabled="!allowEdit"
        :selected-item="schemaChangeType"
        :item-list="[SchemaChange.DDL, SchemaChange.SDL]"
        class="mt-1"
        @select-item="
          (type: SchemaChange) => {
            $emit('change-schema-change-type', type);
          }
        "
      >
        <template #menuItem="{ item }">
          <div class="flex items-center gap-x-2">
            {{
              $t(
                `project.settings.select-schema-change-type-${schemaChangeToJSON(
                  item
                ).toLowerCase()}`
              )
            }}
            <BBBetaBadge v-if="item === SchemaChange.SDL" class="!leading-3" />
          </div>
        </template>
      </BBSelect>
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
      <input
        id="filepathtemplate"
        v-model="repositoryConfig.filePathTemplate"
        name="filepathtemplate"
        type="text"
        class="textfield mt-2 w-full"
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
    <div>
      <div class="textlabel flex gap-x-1">
        {{ $t("repository.schema-path-template") }}
        <span v-if="isProjectSchemaChangeTypeSDL" class="text-red-600">*</span>
        <FeatureBadge feature="bb.feature.vcs-schema-write-back" />
      </div>
      <div class="mt-1 textinfolabel space-x-1">
        <span v-if="isProjectSchemaChangeTypeSDL">
          {{ $t("project.settings.schema-path-template-sdl-description") }}
        </span>
        <template v-else>
          <span>{{ $t("repository.schema-writeback-description") }}</span>
          <span class="font-medium text-main">
            {{ $t("repository.schema-writeback-protected-branch") }}
          </span>
        </template>
        <span v-if="!hasFeature('bb.feature.vcs-schema-write-back')">
          {{
            $t(
              subscriptionStore.getFeatureRequiredPlanString(
                "bb.feature.vcs-schema-write-back"
              )
            )
          }}
        </span>
        <LearnMoreLink
          url="https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files?source=console#schema-path-template"
          class="ml-1"
        />
      </div>
      <BBAttention
        v-if="
          instanceWithoutLicense.length > 0 &&
          hasFeature('bb.feature.vcs-schema-write-back')
        "
        class="my-4"
        :style="`WARN`"
        :title="
          $t('subscription.features.bb-feature-vcs-schema-write-back.title')
        "
        :description="
          $t('subscription.instance-assignment.missing-license-for-instances', {
            count: instanceWithoutLicense.length,
            name: instanceWithoutLicense.map((ins) => ins.title).join(','),
          })
        "
        :action-text="$t('subscription.instance-assignment.assign-license')"
        @click-action="state.showInstanceAssignmentDrawer = true"
      />
      <input
        v-if="hasFeature('bb.feature.vcs-schema-write-back')"
        id="schemapathtemplate"
        v-model="repositoryConfig.schemaPathTemplate"
        name="schemapathtemplate"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
      />
      <input
        v-else-if="isProjectSchemaChangeTypeSDL"
        id="schemapathtemplate"
        v-model="repositoryConfig.schemaPathTemplate"
        name="schemapathtemplate"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
      />
      <input
        v-else-if="isProjectSchemaChangeTypeDDL"
        type="text"
        class="textfield mt-2 w-full"
        :value="
          subscriptionStore.getRquiredPlanString(
            'bb.feature.vcs-schema-write-back'
          )
        "
        :disabled="true"
      />
      <div v-if="schemaTagPlaceholder" class="mt-2 textinfolabel">
        <span class="text-red-600">*</span>
        <span v-if="isProjectSchemaChangeTypeDDL" class="ml-1">
          {{ $t("repository.if-specified") }},
        </span>
        <span class="ml-1">{{ schemaTagPlaceholder }}</span>
      </div>
      <div
        v-if="repositoryConfig.schemaPathTemplate"
        class="mt-2 textinfolabel"
      >
        • {{ $t("repository.schema-path-example") }}:
        {{
          sampleSchemaPath(
            repositoryConfig.baseDirectory,
            repositoryConfig.schemaPathTemplate
          )
        }}
      </div>
    </div>
    <div>
      <div class="textlabel flex gap-x-1">
        {{ $t("repository.sheet-path-template") }}
        <FeatureBadge feature="bb.feature.vcs-sheet-sync" />
      </div>
      <div class="mt-1 textinfolabel">
        {{ $t("repository.sheet-path-template-description") }}
      </div>
      <input
        v-if="hasFeature('bb.feature.vcs-sheet-sync')"
        id="sheetpathtemplate"
        v-model="repositoryConfig.sheetPathTemplate"
        name="sheetpathtemplate"
        type="text"
        class="textfield mt-2 w-full"
        :disabled="!allowEdit"
      />
      <input
        v-else
        type="text"
        class="textfield mt-2 w-full"
        :value="
          subscriptionStore.getRquiredPlanString('bb.feature.vcs-sheet-sync')
        "
        :disabled="true"
      />
      <div class="mt-2 textinfolabel capitalize">
        <span class="text-red-600">*</span>
        {{ $t("common.required-placeholder") }}: {{ "\{\{NAME\}\}" }};
        <template v-if="schemaOptionalTagPlaceholder.length > 0">
          {{ $t("common.optional-placeholder") }}: {{ "\{\{ENV_ID\}\}" }},
          {{ "\{\{DB_NAME\}\}" }}
        </template>
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
              vcsType === ExternalVersionControl_Type.GITLAB
                ? $t("repository.merge-request")
                : $t("repository.pull-request"),
            pathTemplate:
              schemaChangeType == SchemaChange.DDL
                ? $t("repository.file-path-template")
                : $t("repository.schema-path-template"),
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
        :style="`WARN`"
        :title="$t('subscription.features.bb-feature-sql-review.title')"
        :description="
          $t('subscription.instance-assignment.missing-license-for-instances', {
            count: instanceWithoutLicense.length,
            name: instanceWithoutLicense.map((ins) => ins.title).join(','),
          })
        "
        :action-text="$t('subscription.instance-assignment.assign-license')"
        @click-action="state.showInstanceAssignmentDrawer = true"
      />
      <div class="flex space-x-4 mt-2">
        <BBCheckbox
          :disabled="!allowEdit"
          :title="enableSQLReviewTitle"
          :value="repositoryConfig.enableSQLReviewCI"
          @toggle="(on: boolean) => {
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
import { reactive, PropType, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  hasFeature,
  useSubscriptionV1Store,
  useDatabaseV1Store,
} from "@/store";
import { ExternalRepositoryInfo, RepositoryConfig } from "@/types";
import { ExternalVersionControl_Type } from "@/types/proto/v1/externalvs_service";
import {
  Project,
  TenantMode,
  SchemaChange,
  schemaChangeToJSON,
} from "@/types/proto/v1/project_service";
import { PlanType } from "@/types/proto/v1/subscription_service";

const FILE_REQUIRED_PLACEHOLDER = "{{DB_NAME}}, {{VERSION}}, {{TYPE}}";
const SCHEMA_REQUIRED_PLACEHOLDER = "{{DB_NAME}}";
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

const props = defineProps({
  allowEdit: {
    default: true,
    type: Boolean,
  },
  create: {
    type: Boolean,
    default: false,
  },
  vcsType: {
    required: true,
    type: Object as PropType<ExternalVersionControl_Type>,
  },
  vcsName: {
    required: true,
    type: String,
  },
  repositoryInfo: {
    required: true,
    type: Object as PropType<ExternalRepositoryInfo>,
  },
  repositoryConfig: {
    required: true,
    type: Object as PropType<RepositoryConfig>,
  },
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  schemaChangeType: {
    required: true,
    type: Object as PropType<SchemaChange>,
  },
});

const { t } = useI18n();

const state = reactive<LocalState>({
  showFeatureModal: false,
  showInstanceAssignmentDrawer: false,
});

const subscriptionStore = useSubscriptionV1Store();

const databaseV1List = computed(() => {
  return useDatabaseV1Store().databaseListByProject(props.project.name);
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
const isProjectSchemaChangeTypeSDL = computed(() => {
  return (props.schemaChangeType || SchemaChange.DDL) === SchemaChange.SDL;
});
const canEnableSQLReview = computed(() => {
  return (
    props.vcsType == ExternalVersionControl_Type.GITHUB ||
    props.vcsType === ExternalVersionControl_Type.GITLAB ||
    props.vcsType === ExternalVersionControl_Type.AZURE_DEVOPS
  );
});
const enableSQLReviewTitle = computed(() => {
  switch (props.vcsType) {
    case ExternalVersionControl_Type.GITLAB:
      return t("repository.sql-review-ci-enable-gitlab");
    case ExternalVersionControl_Type.GITHUB:
      return t("repository.sql-review-ci-enable-github");
    case ExternalVersionControl_Type.AZURE_DEVOPS:
      return t("repository.sql-review-ci-enable-azure");
    default:
      return "";
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

const sampleSchemaPath = (
  baseDirectory: string,
  schemaPathTemplate: string
): string => {
  type Item = {
    placeholder: string;
    sampleText: string;
  };
  const placeholderList: Item[] = [
    {
      placeholder: "{{DB_NAME}}",
      sampleText: "db1",
    },
    {
      placeholder: "{{ENV_ID}}",
      sampleText: "env1",
    },
    {
      placeholder: "{{ENV_NAME}}", // for legacy support
      sampleText: "env1",
    },
  ];
  let result = `${baseDirectory}/${schemaPathTemplate}`;
  for (const item of placeholderList) {
    const re = new RegExp(item.placeholder, "g");
    result = result.replace(re, item.sampleText);
  }
  return result;
};

const fileOptionalPlaceholder = computed(() => {
  const tags = [] as string[];
  // Only allows {{ENV_ID}} to be an optional placeholder for non-tenant mode projects
  if (!isTenantProject.value) tags.push("{{ENV_ID}}");
  tags.push("{{DESCRIPTION}}");
  return tags;
});

const schemaRequiredTagPlaceholder = computed(() => {
  const tags = [] as string[];
  // Only allows {{DB_NAME}} to be an optional placeholder for non-tenant mode projects
  if (!isTenantProject.value) tags.push(SCHEMA_REQUIRED_PLACEHOLDER);
  return tags;
});

const schemaOptionalTagPlaceholder = computed(() => {
  const tags = [] as string[];
  // Only allows {{ENV_ID}} to be an optional placeholder for non-tenant mode projects
  if (!isTenantProject.value) tags.push("{{ENV_ID}}");
  return tags;
});

const schemaTagPlaceholder = computed(() => {
  const placeholders: string[] = [];
  const required = schemaRequiredTagPlaceholder.value;
  const optional = schemaOptionalTagPlaceholder.value;
  if (required.length > 0) {
    placeholders.push(
      `${t("common.required-placeholder")}: ${required.join(", ")}`
    );
  }
  if (optional.length > 0) {
    placeholders.push(
      `${t("common.optional-placeholder")}: ${optional.join(", ")}`
    );
  }
  return placeholders.join("; ");
});

const onSQLReviewCIToggle = (on: boolean) => {
  if (on && !hasFeature("bb.feature.vcs-sql-review")) {
    state.showFeatureModal = true;
  }
};
</script>
