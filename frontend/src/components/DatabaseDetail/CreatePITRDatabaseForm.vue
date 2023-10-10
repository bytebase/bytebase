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
  </div>
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
import { computed, onBeforeMount, PropType, reactive, watch } from "vue";
import { isPITRAvailableOnInstanceV1 } from "@/plugins/pitr";
import { useDBSchemaV1Store, useInstanceV1Store } from "@/store";
import {
  ComposedInstance,
  ComposedDatabase,
  defaultCharsetOfEngineV1,
  defaultCollationOfEngineV1,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
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

const isReservedName = computed(() => {
  return state.context.databaseName.toLowerCase() == "bytebase";
});

const selectedInstance = computed(() => {
  return instanceV1Store.getInstanceByUID(state.context.instanceId);
});

const instanceFilter = (instance: ComposedInstance): boolean => {
  return isPITRAvailableOnInstanceV1(instance);
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
  return (
    !isEmpty(state.context.databaseName) &&
    !isReservedName.value &&
    !!state.context.projectId &&
    !!state.context.environmentId &&
    !!state.context.instanceId
  );
};

defineExpose({ validate });
</script>
