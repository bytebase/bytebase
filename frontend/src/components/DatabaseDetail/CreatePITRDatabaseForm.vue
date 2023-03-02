<template>
  <div class="space-y-2">
    <label class="textlabel">
      {{ $t("common.project") }}
      <span class="text-red-600">*</span>
    </label>
    <ProjectSelect
      class="mt-1"
      :disabled="true"
      :include-default-project="true"
      :selected-id="state.context.projectId"
      @select-project-id="(id: number) => (state.context.projectId = id)"
    />
  </div>

  <div class="col-span-2 col-start-2 w-64">
    <label for="name" class="textlabel">
      {{ $t("create-db.new-database-name") }}
      <span class="text-red-600">*</span>
    </label>
    <input
      id="name"
      v-model="state.context.databaseName"
      required
      name="name"
      type="text"
      class="textfield mt-1 w-full"
    />
    <span v-if="isReservedName" class="text-red-600">
      <i18n-t keypath="create-db.reserved-db-error">
        <template #databaseName>
          {{ state.context.databaseName }}
        </template>
      </i18n-t>
    </span>
    <DatabaseNameTemplateTips
      v-if="isDbNameTemplateMode"
      :project="project"
      :name="state.context.databaseName"
      :label-list="state.context.labelList"
    />
  </div>

  <!-- Providing more dropdowns for required labels as if they are normal required props of DB -->
  <DatabaseLabelForm
    v-if="isTenantProject"
    ref="labelForm"
    :project="project"
    :label-list="state.context.labelList"
    filter="required"
  />

  <div class="space-y-2">
    <label class="textlabel">
      {{ $t("common.environment") }}
      <span class="text-red-600">*</span>
    </label>
    <EnvironmentSelect
      class="mt-1"
      :selected-id="state.context.environmentId"
      :disabled="true"
      @select-environment-id="(id: number) => (state.context.environmentId = id)"
    />
  </div>

  <div class="space-y-2">
    <label class="textlabel w-full flex items-center gap-1">
      <InstanceEngineIcon
        v-if="state.context.instanceId"
        :instance="selectedInstance"
      />
      <label for="instance" class="textlabel">
        {{ $t("common.instance") }} <span class="text-red-600">*</span>
      </label>
    </label>
    <InstanceSelect
      class="mt-1"
      :selected-id="state.context.instanceId"
      :environment-id="state.context.environmentId"
      :filter="instanceFilter"
      @select-instance-id="(id: number) => (state.context.instanceId = id)"
    />
  </div>

  <!-- Providing other dropdowns for optional labels as if they are normal optional props of DB -->
  <DatabaseLabelForm
    v-if="isTenantProject"
    :project="project"
    :label-list="state.context.labelList"
    filter="optional"
  />

  <div class="space-y-2">
    <label class="textlabel w-full flex gap-1">
      {{
        selectedInstance.engine == "POSTGRES"
          ? $t("db.encoding")
          : $t("db.character-set")
      }}
    </label>
    <input
      id="charset"
      v-model="state.context.characterSet"
      name="charset"
      type="text"
      class="textfield mt-1 w-full"
      :placeholder="defaultCharset(selectedInstance.engine)"
    />
  </div>

  <div class="col-span-2 col-start-2 w-64">
    <label for="collation" class="textlabel">
      {{ $t("db.collation") }}
    </label>
    <input
      id="collation"
      v-model="state.context.collation"
      name="collation"
      type="text"
      class="textfield mt-1 w-full"
      :placeholder="defaultCollation(selectedInstance.engine) || 'default'"
    />
  </div>

  <!-- Assignee is not required. Since we are definitely DBA or Owner to see this form -->
</template>

<script lang="ts" setup>
import { computed, onBeforeMount, PropType, reactive, ref, watch } from "vue";
import { cloneDeep, isEmpty } from "lodash-es";
import {
  Database,
  Instance,
  Project,
  defaultCharset,
  defaultCollation,
} from "@/types";
import { CreatePITRDatabaseContext } from "./utils";
import {
  DatabaseLabelForm,
  DatabaseNameTemplateTips,
} from "@/components/CreateDatabasePrepForm";
import { useInstanceStore, useProjectStore, useDBSchemaStore } from "@/store";
import { isPITRAvailableOnInstance } from "@/plugins/pitr";

interface LocalState {
  context: CreatePITRDatabaseContext;
}

const props = defineProps({
  database: {
    type: Object as PropType<Database>,
    required: true,
  },
  context: {
    type: Object as PropType<CreatePITRDatabaseContext>,
    default: undefined,
  },
});

const emit = defineEmits<{
  (e: "update", context: CreatePITRDatabaseContext): void;
}>();

const extractLocalContextFromProps = (): CreatePITRDatabaseContext => {
  const { database, context } = props;
  if (context) {
    return context;
  } else {
    const dbSchemaMetadata = dbSchemaStore.getDatabaseMetadataByDatabaseId(
      props.database.id
    );

    return {
      projectId: database.project.id,
      instanceId: database.instance.id,
      environmentId: database.instance.environment.id,
      databaseName: `${database.name}_recovery`, // looks like "my_db_recovery"
      characterSet: dbSchemaMetadata.characterSet,
      collation: dbSchemaMetadata.collation,
      labelList: cloneDeep(database.labels),
    };
  }
};

const instanceStore = useInstanceStore();
const projectStore = useProjectStore();
const dbSchemaStore = useDBSchemaStore();

// Refresh the instance list
const prepareInstanceList = () => {
  instanceStore.fetchInstanceList();
};

onBeforeMount(prepareInstanceList);

const state = reactive<LocalState>({
  context: extractLocalContextFromProps(),
});

const project = computed((): Project => {
  return projectStore.getProjectById(state.context.projectId);
});

const isReservedName = computed(() => {
  return state.context.databaseName.toLowerCase() == "bytebase";
});

const isTenantProject = computed((): boolean => {
  return project.value.tenantMode === "TENANT";
});

// reference to <DatabaseLabelForm /> to call validate()
const labelForm = ref<InstanceType<typeof DatabaseLabelForm> | null>(null);

const isDbNameTemplateMode = computed((): boolean => {
  if (project.value.tenantMode !== "TENANT") return false;
  // true if dbNameTemplate is not empty
  return !!project.value.dbNameTemplate;
});

const selectedInstance = computed((): Instance => {
  return instanceStore.getInstanceById(state.context.instanceId);
});

const instanceFilter = (instance: Instance): boolean => {
  return isPITRAvailableOnInstance(instance);
};

// Sync values from props when changes.
watch([() => props.database, () => props.context], () => {
  state.context = extractLocalContextFromProps();
});

// Emit 'update' event when local value changes.
watch(
  () => state.context,
  (context) => {
    emit("update", context);
  },
  {
    deep: true,
    immediate: true,
  }
);

const validate = (): boolean => {
  // If we are not in template mode, none of labels are required
  // So we just treat this case as 'yes, valid'
  const isLabelValid = isDbNameTemplateMode.value
    ? labelForm.value?.validate()
    : true;
  return (
    !isEmpty(state.context.databaseName) &&
    !isReservedName.value &&
    !!isLabelValid &&
    !!state.context.projectId &&
    !!state.context.environmentId &&
    !!state.context.instanceId
  );
};

defineExpose({ validate });
</script>
