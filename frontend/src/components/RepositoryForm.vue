<!-- eslint-disable vue/no-mutating-props -->
<template>
  <div class="space-y-4">
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="gitprovider" class="textlabel">
          {{ $t("repository.git-provider") }}
        </label>
        <template v-if="vcsType.startsWith('GITLAB')">
          <img class="h-4 w-auto" src="../assets/gitlab-logo.svg" />
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
      </div>
      <BBSelect
        id="schemamigrationtype"
        :disabled="!allowEdit"
        :selected-item="schemaChangeType"
        :item-list="['DDL', 'SDL']"
        class="mt-1"
        @select-item="
          (type: string) => {
            $emit('change-schema-change-type', type);
          }
        "
      >
        <template #menuItem="{ item }">
          {{
            $t(
              `project.settings.select-schema-change-type-${item.toLowerCase()}`
            )
          }}
          <BBBetaBadge v-if="item === 'SDL'" />
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
        <a
          href="https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files#file-path-template?source=console"
          target="_BLANK"
          class="font-normal normal-link ml-1"
        >
          {{ $t("common.learn-more") }}</a
        >
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
        <FeatureBadge
          feature="bb.feature.vcs-schema-write-back"
          class="text-accent"
        />
      </div>
      <div class="mt-1 textinfolabel">
        <template v-if="isProjectSchemaChangeTypeSDL">
          {{ $t("project.settings.schema-path-template-sdl-description") }}
        </template>
        <template v-else>
          {{ $t("repository.schema-writeback-description") }}
          <span class="font-medium text-main">{{
            $t("repository.schema-writeback-protected-branch")
          }}</span>
        </template>
        <a
          href="https://www.bytebase.com/docs/vcs-integration/name-and-organize-schema-files#schema-path-template?source=console"
          target="_BLANK"
          class="font-normal normal-link ml-1"
        >
          {{ $t("common.learn-more") }}</a
        >
      </div>
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
        v-else
        type="text"
        class="textfield mt-2 w-full"
        :value="getRquiredPlanString('bb.feature.vcs-schema-write-back')"
        :disabled="true"
      />
      <div v-if="schemaTagPlaceholder" class="mt-2 textinfolabel">
        <span class="text-red-600">*</span>
        <span class="ml-1">{{ $t("repository.if-specified") }},</span>
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
        {{ $t("repository.sheet-path-template")
        }}<FeatureBadge
          feature="bb.feature.vcs-sheet-sync"
          class="text-accent"
        />
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
        :value="getRquiredPlanString('bb.feature.vcs-sheet-sync')"
        :disabled="true"
      />
      <div class="mt-2 textinfolabel capitalize">
        <span class="text-red-600">*</span>
        {{ $t("common.required-placeholder") }}: {{ "\{\{NAME\}\}" }};
        <template v-if="schemaOptionalTagPlaceholder.length > 0">
          {{ $t("common.optional-placeholder") }}: {{ "\{\{ENV_NAME\}\}" }},
          {{ "\{\{DB_NAME\}\}" }}
        </template>
      </div>
    </div>
    <div>
      <div class="textlabel flex gap-x-1">
        {{ $t("repository.sql-review-ci") }}
        <FeatureBadge feature="bb.feature.vcs-sql-review" class="text-accent" />
      </div>
      <div class="mt-1 textinfolabel">
        {{
          $t("repository.sql-review-ci-description", {
            pr: vcsType.startsWith("GITLAB")
              ? $t("repository.merge-request")
              : $t("repository.pull-request"),
            pathTemplate:
              schemaChangeType == "DDL"
                ? $t("repository.file-path-template")
                : $t("repository.schema-path-template"),
          })
        }}
      </div>
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
      v-if="state.showFeatureModal"
      feature="bb.feature.vcs-sql-review"
      @cancel="state.showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts">
import { reactive, PropType, defineComponent, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  ExternalRepositoryInfo,
  Project,
  RepositoryConfig,
  SchemaChangeType,
  VCSType,
} from "@/types";
import BBBetaBadge from "@/bbkit/BBBetaBadge.vue";
import { hasFeature, useSubscriptionStore } from "@/store";

const FILE_REQUIRED_PLACEHOLDER = "{{DB_NAME}}, {{VERSION}}, {{TYPE}}";
const SCHEMA_REQUIRED_PLACEHOLDER = "{{DB_NAME}}";
const FILE_OPTIONAL_DIRECTORY_WILDCARD = "*, **";
const SINGLE_ASTERISK_REGEX = /\/\*\//g;
const DOUBLE_ASTERISKS_REGEX = /\/\*\*\//g;

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {
  showFeatureModal: boolean;
}

export default defineComponent({
  name: "RepositoryForm",
  components: { BBBetaBadge },
  props: {
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
      type: String as PropType<VCSType>,
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
      type: String as PropType<SchemaChangeType>,
    },
  },
  emits: ["change-repository", "change-schema-change-type"],
  setup(props) {
    const { t } = useI18n();

    const state = reactive<LocalState>({
      showFeatureModal: false,
    });

    const subscriptionStore = useSubscriptionStore();

    const isTenantProject = computed(() => {
      return props.project.tenantMode === "TENANT";
    });
    const isProjectSchemaChangeTypeDDL = computed(() => {
      return (props.schemaChangeType || "DDL") === "DDL";
    });
    const isProjectSchemaChangeTypeSDL = computed(() => {
      return (props.schemaChangeType || "DDL") === "SDL";
    });
    const enableSQLReviewTitle = computed(() => {
      return props.vcsType.startsWith("GITLAB")
        ? t("repository.sql-review-ci-enable-gitlab")
        : t("repository.sql-review-ci-enable-github");
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
          placeholder: "{{ENV_NAME}}",
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
      // Only allows {{ENV_NAME}} to be an optional placeholder for non-tenant mode projects
      if (!isTenantProject.value) tags.push("{{ENV_NAME}}");
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
      // Only allows {{ENV_NAME}} to be an optional placeholder for non-tenant mode projects
      if (!isTenantProject.value) tags.push("{{ENV_NAME}}");
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

    return {
      FILE_REQUIRED_PLACEHOLDER,
      SCHEMA_REQUIRED_PLACEHOLDER,
      FILE_OPTIONAL_DIRECTORY_WILDCARD,
      fileOptionalPlaceholder,
      schemaOptionalTagPlaceholder,
      schemaTagPlaceholder,
      state,
      hasFeature,
      getRquiredPlanString: subscriptionStore.getRquiredPlanString,
      isProjectSchemaChangeTypeDDL,
      isProjectSchemaChangeTypeSDL,
      enableSQLReviewTitle,
      sampleFilePath,
      sampleSchemaPath,
      onSQLReviewCIToggle,
    };
  },
});
</script>
