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
      @select-project-id="(id: string) => (state.context.projectId = id)"
    />
  </div>

  <div class="col-span-2 col-start-2">
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
      :labels="state.context.labels"
    />
  </div>

  <!-- Providing more dropdowns for required labels as if they are normal required props of DB -->
  <DatabaseLabelForm
    v-if="isTenantProject"
    ref="labelForm"
    :project="project"
    :labels="state.context.labels"
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
      @select-environment-id="(id: string) => (state.context.environmentId = id)"
    />
  </div>

  <div class="space-y-2">
    <label class="textlabel w-full flex items-center gap-1">
      <label for="instance" class="textlabel">
        {{ $t("common.instance") }} <span class="text-red-600">*</span>
      </label>
    </label>
    <InstanceSelect
      class="mt-1"
      :selected-id="String(state.context.instanceId)"
      :environment-id="state.context.environmentId"
      :filter="instanceFilter"
      @select-instance-id="(id: number) => (state.context.instanceId = String(id))"
    />
  </div>

  <!-- Providing other dropdowns for optional labels as if they are normal optional props of DB -->
  <DatabaseLabelForm
    v-if="isTenantProject"
    :project="project"
    :labels="state.context.labels"
    filter="optional"
  />

  <div class="space-y-2">
    <label class="textlabel w-full flex gap-1">
      {{
        selectedInstance.engine === Engine.POSTGRES
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
      :placeholder="defaultCharsetOfEngineV1(selectedInstance.engine)"
    />
  </div>

  <div class="col-span-2 col-start-2">
    <label for="collation" class="textlabel">
      {{ $t("db.collation") }}
    </label>
    <input
      id="collation"
      v-model="state.context.collation"
      name="collation"
      type="text"
      class="textfield mt-1 w-full"
      :placeholder="
        defaultCollationOfEngineV1(selectedInstance.engine) || 'default'
      "
    />
  </div>

  <!-- Assignee is not required. Since we are definitely DBA or Owner to see this form -->
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty } from "lodash-es";
import {
  computed,
  onBeforeMount,
  PropType,
  reactive,
  ref,
  toRef,
  watch,
} from "vue";
import {
  DatabaseLabelForm,
  DatabaseNameTemplateTips,
  useDBNameTemplateInputState,
} from "@/components/CreateDatabasePrepForm";
import { isPITRAvailableOnInstanceV1 } from "@/plugins/pitr";
import {
  useDBSchemaV1Store,
  useProjectV1ByUID,
  useInstanceV1Store,
} from "@/store";
import {
  ComposedInstance,
  ComposedDatabase,
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { TenantMode } from "@/types/proto/v1/project_service";
import { CreatePITRDatabaseContext } from "./utils";

interface LocalState {
  context: CreatePITRDatabaseContext;
}

const props = defineProps({
  database: {
    type: Object as PropType<ComposedDatabase>,
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
    const dbSchemaMetadata = dbSchemaStore.getDatabaseMetadata(
      props.database.name
    );

    return {
      projectId: database.projectEntity.uid,
      instanceId: database.instanceEntity.uid,
      environmentId: database.instanceEntity.environmentEntity.uid,
      databaseName: `${database.databaseName}_recovery`, // looks like "my_db_recovery"
      characterSet: dbSchemaMetadata.characterSet,
      collation: dbSchemaMetadata.collation,
      labels: cloneDeep(database.labels),
    };
  }
};

const instanceV1Store = useInstanceV1Store();
const dbSchemaStore = useDBSchemaV1Store();

// Refresh the instance list
const prepareInstanceList = () => {
  instanceV1Store.fetchInstanceList();
};

onBeforeMount(prepareInstanceList);

const state = reactive<LocalState>({
  context: extractLocalContextFromProps(),
});

const { project } = useProjectV1ByUID(computed(() => state.context.projectId));

const isReservedName = computed(() => {
  return state.context.databaseName.toLowerCase() == "bytebase";
});

const isTenantProject = computed((): boolean => {
  return project.value.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

// reference to <DatabaseLabelForm /> to call validate()
const labelForm = ref<InstanceType<typeof DatabaseLabelForm> | null>(null);

const isDbNameTemplateMode = computed((): boolean => {
  if (project.value.tenantMode !== TenantMode.TENANT_MODE_ENABLED) return false;
  // true if dbNameTemplate is not empty
  return !!project.value.dbNameTemplate;
});

const selectedInstance = computed(() => {
  return instanceV1Store.getInstanceByUID(state.context.instanceId);
});

const instanceFilter = (instance: ComposedInstance): boolean => {
  return isPITRAvailableOnInstanceV1(instance);
};

useDBNameTemplateInputState(project, {
  databaseName: toRef(state.context, "databaseName"),
  labels: toRef(state.context, "labels"),
});

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
