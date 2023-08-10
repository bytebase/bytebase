<template>
  <component :is="drawer ? DrawerContent : 'div'" v-bind="bindings">
    <div class="space-y-6 divide-y divide-block-border">
      <div class="divide-y divide-block-border w-[850px]">
        <div v-if="isCreating" class="w-full mt-4 mb-6 grid grid-cols-4 gap-2">
          <template v-for="engine in EngineList" :key="engine">
            <div
              class="flex relative justify-start p-2 border rounded cursor-pointer hover:bg-control-bg-hover"
              :class="
                basicInfo.engine === engine && 'font-medium bg-control-bg-hover'
              "
              @click.capture="changeInstanceEngine(engine)"
            >
              <div class="flex flex-row justify-start items-center">
                <input
                  type="radio"
                  class="btn mr-2"
                  :checked="basicInfo.engine === engine"
                />
                <img
                  v-if="EngineIconPath[engine]"
                  class="w-5 h-auto max-h-[20px] object-contain mr-1"
                  :src="EngineIconPath[engine]"
                />
                <p class="text-center text-sm">
                  {{ engineNameV1(engine) }}
                </p>
                <template v-if="isEngineBeta(engine)">
                  <BBBetaBadge
                    class="absolute -top-px -right-px rounded text-xs !bg-gray-500 px-1 !py-0"
                  />
                </template>
              </div>
            </div>
          </template>
        </div>

        <!-- Instance Name -->
        <div class="pt-4 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="name" class="textlabel flex flex-row items-center">
              {{ $t("instance.instance-name") }}
              <span class="text-red-600 mr-2">*</span>
              <template v-if="props.instance">
                <InstanceV1EngineIcon
                  :instance="props.instance"
                  :tooltip="false"
                />
                <span class="ml-1">{{ props.instance.engineVersion }}</span>
              </template>
            </label>
            <input
              id="name"
              v-model="basicInfo.title"
              required
              name="name"
              type="text"
              class="textfield mt-1 w-full"
              :disabled="!allowEdit"
            />
          </div>

          <div
            v-if="subscriptionStore.currentPlan !== PlanType.FREE"
            class="sm:col-span-2 ml-0 sm:ml-3"
          >
            <label for="activation" class="textlabel block">
              {{ $t("subscription.instance-assignment.assign-license") }}
              (<router-link to="/setting/subscription" class="accent-link">
                {{
                  $t("subscription.instance-assignment.n-license-remain", {
                    n: availableLicenseCountText,
                  })
                }}</router-link
              >)
            </label>
            <BBSwitch
              class="mt-2"
              :text="false"
              :value="basicInfo.activation"
              :disabled="
                !allowEdit ||
                (!basicInfo.activation && availableLicenseCount === 0)
              "
              @toggle="changeInstanceActivation"
            />
          </div>

          <div
            :key="basicInfo.environment"
            class="sm:col-span-3 sm:col-start-1 -mt-4"
          >
            <ResourceIdField
              ref="resourceIdField"
              v-model:value="resourceId"
              class="max-w-full flex-nowrap"
              resource-type="instance"
              :readonly="!isCreating"
              :resource-title="basicInfo.title"
              :validate="validateResourceId"
            />
          </div>

          <div class="sm:col-span-2 sm:col-start-1">
            <label for="environment" class="textlabel">
              {{ $t("common.environment") }}
            </label>
            <EnvironmentSelect
              id="environment"
              class="mt-1 w-full"
              name="environment"
              :disabled="!isCreating"
              :selected-id="environment.uid"
              @select-environment-id="handleSelectEnvironmentUID"
            />
          </div>

          <OracleSyncModeInput
            v-if="basicInfo.engine === Engine.ORACLE"
            :schema-tenant-mode="basicInfo.options?.schemaTenantMode ?? false"
            :allow-edit="allowEdit"
            @update:schema-tenant-mode="changeSyncMode"
          />

          <div class="sm:col-span-3 sm:col-start-1">
            <template v-if="basicInfo.engine !== Engine.SPANNER">
              <label for="host" class="textlabel block">
                <template v-if="basicInfo.engine === Engine.SNOWFLAKE">
                  {{ $t("instance.account-name") }}
                  <span class="text-red-600 mr-2">*</span>
                </template>
                <template v-else>
                  {{ $t("instance.host-or-socket") }}
                  <span class="text-red-600 mr-2">*</span>
                </template>
              </label>
              <input
                id="host"
                v-model="adminDataSource.host"
                required
                type="text"
                name="host"
                :placeholder="
                  basicInfo.engine === Engine.SNOWFLAKE
                    ? $t('instance.your-snowflake-account-name')
                    : $t('instance.sentence.host.snowflake')
                "
                class="textfield mt-1 w-full"
                :disabled="!allowEdit"
              />
              <div
                v-if="basicInfo.engine === Engine.SNOWFLAKE"
                class="mt-2 textinfolabel"
              >
                {{ $t("instance.sentence.proxy.snowflake") }}
              </div>
            </template>
            <SpannerHostInput
              v-else
              v-model:host="adminDataSource.host"
              :allow-edit="allowEdit"
            />
          </div>

          <template v-if="basicInfo.engine !== Engine.SPANNER">
            <div class="sm:col-span-1">
              <label for="port" class="textlabel block">
                {{ $t("instance.port") }}
              </label>
              <input
                id="port"
                type="text"
                name="port"
                class="textfield mt-1 w-full"
                :value="adminDataSource.port"
                :placeholder="defaultPort"
                :disabled="!allowEdit || !allowEditPort"
                @wheel="(e: WheelEvent) => (e.target as HTMLInputElement).blur()"
                @input="adminDataSource.port = trimInputValue($event.target)"
              />
            </div>
          </template>

          <div
            v-if="basicInfo.engine === Engine.MONGODB"
            class="sm:col-span-4 sm:col-start-1"
          >
            <label
              for="connectionStringSchema"
              class="textlabel flex flex-row items-center"
            >
              {{ $t("data-source.connection-string-schema") }}
            </label>
            <label
              v-for="type in MongoDBConnectionStringSchemaList"
              :key="type"
              class="radio h-7"
            >
              <input
                type="radio"
                class="btn"
                name="connectionStringSchema"
                :value="type"
                :checked="type === currentMongoDBConnectionSchema"
                @change="handleMongodbConnectionStringSchemaChange"
              />
              <span class="label">
                {{ type }}
              </span>
            </label>
          </div>

          <!--Do not show external link on create to reduce cognitive load-->
          <div v-if="!isCreating" class="sm:col-span-3 sm:col-start-1">
            <label for="external-link" class="textlabel inline-flex">
              <span class>
                {{
                  basicInfo.engine === Engine.SNOWFLAKE
                    ? $t("instance.snowflake-web-console")
                    : $t("instance.external-link")
                }}
              </span>
              <button
                class="ml-1 btn-icon"
                :disabled="instanceLink.trim().length === 0"
                @click.prevent="window.open(urlfy(instanceLink), '_blank')"
              >
                <heroicons-outline:external-link class="w-4 h-4" />
              </button>
            </label>
            <template v-if="basicInfo.engine === Engine.SNOWFLAKE">
              <input
                id="external-link"
                required
                name="external-link"
                type="text"
                class="textfield mt-1 w-full"
                disabled="true"
                :value="instanceLink"
              />
            </template>
            <template v-else>
              <div class="mt-1 textinfolabel">
                {{ $t("instance.sentence.console.snowflake") }}
              </div>
              <input
                id="external-link"
                v-model="basicInfo.externalLink"
                required
                name="external-link"
                type="text"
                :disabled="!allowEdit"
                class="textfield mt-1 w-full"
                :placeholder="SnowflakeExtraLinkPlaceHolder"
              />
            </template>
          </div>
        </div>

        <!-- Connection Info -->
        <p class="mt-6 pt-4 w-full text-lg leading-6 font-medium text-gray-900">
          {{ $t("instance.connection-info") }}
        </p>

        <DataSourceSection />

        <BBAttention
          v-if="outboundIpList && actuatorStore.isSaaSMode"
          class="my-5 border-none"
          :style="'INFO'"
          :title="$t('instance.sentence.outbound-ip-list')"
          :description="outboundIpList"
        />

        <div class="mt-6 pt-0 border-none">
          <div class="flex flex-row space-x-2">
            <button
              type="button"
              class="btn-normal whitespace-nowrap flex items-center gap-x-1"
              :disabled="!allowCreate || state.isRequesting || !allowEdit"
              @click.prevent="testConnection(false /* !silent */)"
            >
              <BBSpin v-if="state.isTestingConnection" />
              <span>{{ $t("instance.test-connection") }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Action Button Group -->
      <div v-if="!drawer" class="pt-4">
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-4">
            <NButton
              v-if="allowEdit"
              :disabled="!allowUpdate || state.isRequesting"
              :loading="state.isRequesting"
              type="primary"
              @click.prevent="doUpdate"
            >
              {{ $t("common.update") }}
            </NButton>
          </div>
        </div>
      </div>

      <InstanceArchiveRestoreButton
        v-if="!isCreating && instance"
        :instance="(instance as ComposedInstance)"
      />
    </div>

    <template v-if="drawer" #footer>
      <div class="w-full flex justify-between items-center">
        <div class="w-full flex justify-end items-center gap-x-3">
          <NButton
            :disabled="state.isRequesting || state.isTestingConnection"
            @click.prevent="cancel"
          >
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            :disabled="
              !allowCreate || state.isRequesting || state.isTestingConnection
            "
            :loading="state.isRequesting"
            type="primary"
            @click.prevent="tryCreate"
          >
            {{ $t("common.create") }}
          </NButton>
        </div>
      </div>
    </template>
  </component>

  <FeatureModal
    feature="bb.feature.read-replica-connection"
    :open="showReadOnlyDataSourceFeatureModal"
    :instance="instance"
    @cancel="showReadOnlyDataSourceFeatureModal = false"
  />

  <BBAlert
    v-if="state.showCreateInstanceWarningModal"
    :style="'WARN'"
    :ok-text="t('instance.ignore-and-create')"
    :title="$t('instance.connection-info-seems-to-be-incorrect')"
    :description="state.createInstanceWarning"
    :progress-text="$t('common.creating')"
    @ok="handleWarningModalOkClick"
    @cancel="state.showCreateInstanceWarningModal = false"
  ></BBAlert>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual, omit } from "lodash-es";
import { NButton } from "naive-ui";
import { Status } from "nice-grpc-common";
import {
  computed,
  reactive,
  PropType,
  ref,
  watch,
  onMounted,
  toRef,
} from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import EnvironmentSelect from "@/components/EnvironmentSelect.vue";
import { InstanceArchiveRestoreButton } from "@/components/Instance";
import { DrawerContent, InstanceV1EngineIcon } from "@/components/v2";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { instanceServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useSettingV1Store,
  useActuatorV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
  useGracefulRequest,
} from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import {
  UNKNOWN_ID,
  ResourceId,
  ValidatedMessage,
  unknownEnvironment,
  ComposedInstance,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  DataSource,
  DataSourceType,
  Instance,
  InstanceOptions,
} from "@/types/proto/v1/instance_service";
import { PlanType } from "@/types/proto/v1/subscription_service";
import {
  isDev,
  isValidSpannerHost,
  extractInstanceResourceName,
  engineNameV1,
  instanceV1Slug,
  calcUpdateMask,
} from "@/utils";
import { extractGrpcErrorMessage, getErrorCode } from "@/utils/grpcweb";
import DataSourceSection from "./DataSourceSection/DataSourceSection.vue";
import OracleSyncModeInput from "./OracleSyncModeInput.vue";
import SpannerHostInput from "./SpannerHostInput.vue";
import { EditDataSource, extractDataSourceEditState } from "./common";
import { extractBasicInfo } from "./common";
import {
  MongoDBConnectionStringSchemaList,
  SnowflakeExtraLinkPlaceHolder,
  defaultPortForEngine,
  EngineIconPath,
  EngineList,
} from "./constants";
import { provideInstanceFormContext } from "./context";
import { useInstanceSpecs } from "./specs";

const props = defineProps({
  instance: {
    type: Object as PropType<Instance>,
    default: undefined,
  },
  drawer: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["dismiss"]);

const bindings = computed(() => {
  if (props.drawer) {
    return {
      title: t("quick-action.add-instance"),
    };
  }
  return {};
});

const cancel = () => {
  emit("dismiss");
};

interface LocalState {
  editingDataSourceId: string | undefined;
  showFeatureModal: boolean;
  isTestingConnection: boolean;
  isRequesting: boolean;
  showCreateInstanceWarningModal: boolean;
  createInstanceWarning: string;
}

const { t } = useI18n();
const router = useRouter();
const instanceV1Store = useInstanceV1Store();
const settingV1Store = useSettingV1Store();
const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();

const state = reactive<LocalState>({
  editingDataSourceId: props.instance?.dataSources.find(
    (ds) => ds.type === DataSourceType.ADMIN
  )?.id,
  showFeatureModal: false,
  isTestingConnection: false,
  isRequesting: false,
  showCreateInstanceWarningModal: false,
  createInstanceWarning: "",
});

const instance = toRef(props, "instance");
const context = provideInstanceFormContext({ instance });
const {
  isCreating,
  allowEdit,
  basicInfo,
  dataSourceEditState,
  adminDataSource,
  editingDataSource,
  readonlyDataSourceList,
  hasReadonlyReplicaFeature,
  showReadOnlyDataSourceFeatureModal,
} = context;
const {
  showDatabase,
  showSSL,
  showSSH,
  isEngineBeta,
  defaultPort,
  instanceLink,
  allowEditPort,
} = useInstanceSpecs(context);

const availableLicenseCount = computed(() => {
  return Math.max(
    0,
    subscriptionStore.instanceLicenseCount -
      instanceV1Store.activateInstanceCount
  );
});

const availableLicenseCountText = computed((): string => {
  if (subscriptionStore.instanceLicenseCount === Number.MAX_VALUE) {
    return t("subscription.unlimited");
  }
  return `${availableLicenseCount.value}`;
});

const environment = computed(() => {
  return (
    useEnvironmentV1Store().getEnvironmentByName(basicInfo.value.environment) ??
    unknownEnvironment()
  );
});

const resourceId = computed({
  get() {
    const id = extractInstanceResourceName(basicInfo.value.name);
    if (id === String(UNKNOWN_ID)) return "";
    return id;
  },
  set(id) {
    basicInfo.value.name = `instances/${id}`;
  },
});
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

onMounted(async () => {
  if (isCreating.value) {
    adminDataSource.value.host = isDev() ? "127.0.0.1" : "host.docker.internal";
    adminDataSource.value.srv = false;
    adminDataSource.value.authenticationDatabase = "";
  }
  await settingV1Store.fetchSettingList();
});

watch(
  () => basicInfo.value.engine,
  () => {
    if (isCreating.value) {
      adminDataSource.value.port = defaultPortForEngine(basicInfo.value.engine);
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => props.instance?.activation,
  (val) => {
    if (val !== undefined) {
      basicInfo.value.activation = val;
    }
  }
);

const outboundIpList = computed(() => {
  if (!settingV1Store.workspaceProfileSetting) {
    return "";
  }
  return settingV1Store.workspaceProfileSetting.outboundIpList.join(",");
});

const currentMongoDBConnectionSchema = computed(() => {
  return adminDataSource.value.srv === false
    ? MongoDBConnectionStringSchemaList[0]
    : MongoDBConnectionStringSchemaList[1];
});
const allowCreate = computed(() => {
  if (basicInfo.value.engine === Engine.SPANNER) {
    return (
      basicInfo.value.title.trim() &&
      isValidSpannerHost(adminDataSource.value.host) &&
      adminDataSource.value.updatedPassword
    );
  }

  return (
    basicInfo.value.title.trim() &&
    resourceIdField.value?.resourceId &&
    resourceIdField.value?.isValidated &&
    adminDataSource.value.host
  );
});

const valueChanged = computed(() => {
  const original = getOriginalEditState();
  const editing = {
    basicInfo: basicInfo.value,
    dataSources: dataSourceEditState.value.dataSources,
  };
  return !isEqual(editing, original);
});

const allowUpdate = computed((): boolean => {
  if (!valueChanged.value) {
    return false;
  }
  if (basicInfo.value.engine === Engine.SPANNER) {
    if (!isValidSpannerHost(adminDataSource.value.host)) {
      return false;
    }
    if (readonlyDataSourceList.value.length > 0) {
      if (
        readonlyDataSourceList.value.some((ds) => !isValidSpannerHost(ds.host))
      ) {
        return false;
      }
    }
  }
  return true;
});

const handleSelectEnvironmentUID = (uid: number | string) => {
  const environment = useEnvironmentV1Store().getEnvironmentByUID(String(uid));
  basicInfo.value.environment = environment.name;
};

// The default host name is 127.0.0.1 or host.docker.internal which is not applicable to Snowflake, so we change
// the host name between 127.0.0.1/host.docker.internal and "" if user hasn't changed default yet.
const changeInstanceEngine = (engine: Engine) => {
  if (engine === Engine.SNOWFLAKE || engine === Engine.SPANNER) {
    if (
      adminDataSource.value.host === "127.0.0.1" ||
      adminDataSource.value.host === "host.docker.internal"
    ) {
      adminDataSource.value.host = "";
    }
  } else {
    if (!adminDataSource.value.host) {
      adminDataSource.value.host = isDev()
        ? "127.0.0.1"
        : "host.docker.internal";
    }
  }
  basicInfo.value.engine = engine;
};

const changeSyncMode = (schemaTenantMode: boolean) => {
  if (!basicInfo.value.options) {
    basicInfo.value.options = InstanceOptions.fromJSON({});
  }
  basicInfo.value.options.schemaTenantMode = schemaTenantMode;
};

const trimInputValue = (target: Event["target"]) => {
  return ((target as HTMLInputElement)?.value ?? "").trim();
};

const handleMongodbConnectionStringSchemaChange = (event: Event) => {
  const ds = editingDataSource.value;
  if (!ds) return;
  switch ((event.target as HTMLInputElement).value) {
    case MongoDBConnectionStringSchemaList[0]:
      ds.srv = false;
      break;
    case MongoDBConnectionStringSchemaList[1]:
      // MongoDB doesn't support specify port if using srv record.
      ds.port = "";
      ds.srv = true;
      break;
    default:
      ds.srv = false;
  }
};

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const instance = await instanceV1Store.getOrFetchInstanceByName(
      instanceNamePrefix + resourceId,
      true /* silent */
    );
    if (instance) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t("resource.instance"),
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

const updateEditState = async (instance: Instance) => {
  basicInfo.value = extractBasicInfo(instance);
  const updatedEditState = extractDataSourceEditState(instance);
  dataSourceEditState.value.dataSources = updatedEditState.dataSources;
  if (
    updatedEditState.dataSources.findIndex(
      (ds) => ds.id === dataSourceEditState.value.editingDataSourceId
    ) < 0
  ) {
    // The original selected data source id is no-longer in the latest data source list
    dataSourceEditState.value.editingDataSourceId =
      updatedEditState.editingDataSourceId;
  }

  // Backend will sync the schema when connection info changed, so we need to fetch the synced schema here.
  instanceV1Store.fetchInstanceRoleListByName(instance.name);
};

const handleWarningModalOkClick = async () => {
  state.showCreateInstanceWarningModal = false;
  doCreate();
};

const tryCreate = async () => {
  const testResult = await testConnection(true /* silent */);
  if (testResult.success) {
    doCreate();
  } else {
    state.createInstanceWarning = t("instance.unable-to-connect", [
      testResult.message,
    ]);
    state.showCreateInstanceWarningModal = true;
  }
};

// We will also create the database * denoting all databases
// and its RW data source. The username, password is actually
// stored in that data source object instead of in the instance self.
// Conceptually, data source is the proper place to store connection info (thinking of DSN)
const doCreate = async () => {
  if (!isCreating.value) {
    return;
  }
  const instanceCreate: Instance = {
    ...basicInfo.value,
    engineVersion: "",
    dataSources: [],
  };
  const adminDataSourceCreate = extractDataSourceFromEdit(
    instanceCreate,
    adminDataSource.value
  );
  instanceCreate.dataSources = [adminDataSourceCreate];

  state.isRequesting = true;
  try {
    await useGracefulRequest(async () => {
      const createdInstance = await instanceV1Store.createInstance(
        instanceCreate
      );
      router.push(`/instance/${instanceV1Slug(createdInstance)}`);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t(
          "instance.successfully-created-instance-createdinstance-name",
          [createdInstance.title]
        ),
      });
    });
  } finally {
    state.isRequesting = false;
    emit("dismiss");
  }
};

const doUpdate = async () => {
  const { instance } = props;
  if (!instance) {
    return;
  }
  if (!checkRODataSourceFeature(instance)) {
    showReadOnlyDataSourceFeatureModal.value = true;
    return;
  }
  // When clicking **Update** we may have more than one thing to do (if needed)
  // 1. Patch the instance itself.
  // 2. Update the admin datasource.
  // 3. Create OR update read-only data source(s).

  const maybeUpdateInstanceBasicInfo = async () => {
    const instancePatch = {
      ...instance,
      ...basicInfo.value,
    };
    const updateMask: string[] = [];
    if (instancePatch.title !== instance.title) {
      updateMask.push("title");
    }
    if (instancePatch.externalLink !== instance.externalLink) {
      updateMask.push("external_link");
    }
    if (instancePatch.activation !== instance.activation) {
      updateMask.push("activation");
    }
    if (
      instancePatch.options?.schemaTenantMode !==
      instance.options?.schemaTenantMode
    ) {
      updateMask.push("options.schema_tenant_mode");
    }
    return await instanceV1Store.updateInstance(instancePatch, updateMask);
  };
  const updateDataSource = async (
    editing: DataSource,
    original: DataSource | undefined,
    editState: EditDataSource
  ) => {
    if (!original) return;
    const updateMask = calcDataSourceUpdateMask(editing, original, editState);
    if (updateMask.length === 0) {
      return;
    }
    return await instanceV1Store.updateDataSource(
      instance,
      editing,
      updateMask
    );
  };
  const maybeUpdateAdminDataSource = async () => {
    const original = instance.dataSources.find(
      (ds) => ds.type === DataSourceType.ADMIN
    );
    const editing = extractDataSourceFromEdit(instance, adminDataSource.value);
    return await updateDataSource(editing, original, adminDataSource.value);
  };
  const maybeUpsertReadonlyDataSources = async () => {
    if (readonlyDataSourceList.value.length === 0) {
      // Nothing to do
      return;
    }
    // Upsert readonly data sources one by one
    for (let i = 0; i < readonlyDataSourceList.value.length; i++) {
      const editing = readonlyDataSourceList.value[i];
      const patch = extractDataSourceFromEdit(instance, editing);
      if (editing.pendingCreate) {
        await instanceV1Store.createDataSource(instance, patch);
      } else {
        const original = instance.dataSources.find(
          (ds) => ds.id === editing.id
        );
        await updateDataSource(patch, original, editing);
      }
    }
  };
  state.isRequesting = true;
  try {
    await useGracefulRequest(async () => {
      await maybeUpdateInstanceBasicInfo();
      await maybeUpdateAdminDataSource();
      await maybeUpsertReadonlyDataSources();
      const updatedInstance = instanceV1Store.getInstanceByName(instance.name);
      await updateEditState(updatedInstance);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("instance.successfully-updated-instance-instance-name", [
          updatedInstance.title,
        ]),
      });
    });
  } finally {
    state.isRequesting = false;
  }
};

const testConnection = async (
  silent = false
): Promise<{ success: boolean; message: string }> => {
  if (!editingDataSource.value) {
    throw new Error("should never reach this line");
  }

  // In different scenes, we use different methods to test connection.
  const ok = () => {
    if (!silent) {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("instance.successfully-connected-instance"),
      });
    }
    state.isTestingConnection = false;
    return { success: true, message: "" };
  };
  const fail = (host: string, err: unknown) => {
    const error = extractGrpcErrorMessage(err);
    if (!silent) {
      let title = t("instance.failed-to-connect-instance");
      if (host === "localhost" || host === "127.0.0.1") {
        title = t("instance.failed-to-connect-instance-localhost");
      }
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title,
        description: error,
        // Manual hide, because user may need time to inspect the error
        manualHide: true,
      });
    }
    state.isTestingConnection = false;
    return { success: false, message: error };
  };
  state.isTestingConnection = true;
  if (isCreating.value) {
    // When creating new instance, use
    // adminDataSource + CreateInstanceRequest.validateOnly = true
    const instance: Instance = {
      ...basicInfo.value,
      engineVersion: "",
      dataSources: [],
    };
    const adminDataSourceCreate = extractDataSourceFromEdit(
      instance,
      adminDataSource.value
    );
    instance.dataSources = [adminDataSourceCreate];
    try {
      await instanceServiceClient.createInstance(
        {
          instance,
          instanceId: extractInstanceResourceName(instance.name),
          validateOnly: true,
        },
        {
          silent: true,
        }
      );
      return ok();
    } catch (err) {
      return fail(adminDataSourceCreate.host, err);
    }
  } else {
    // Editing existed instance.
    const instance = props.instance!;
    const editingDS = editingDataSource.value;
    const ds = extractDataSourceFromEdit(instance, editingDS);
    if (editingDS.pendingCreate) {
      // When read-only data source is about to be created, use
      // editingDataSource + AddDataSourceRequest.validateOnly = true
      try {
        await instanceServiceClient.addDataSource(
          {
            instance: instance.name,
            dataSource: ds,
            validateOnly: true,
          },
          {
            silent: true,
          }
        );
        return ok();
      } catch (err) {
        return fail(ds.host, err);
      }
    } else {
      // When a data source (admin or read-only) has been edited, use
      // editingDataSource + UpdateDataSourceRequest.validateOnly = true
      try {
        const original = instance.dataSources.find(
          (ds) => ds.id === editingDS.id
        );
        if (!original) {
          throw new Error("should never reach this line");
        }
        const updateMask = calcDataSourceUpdateMask(ds, original, editingDS);
        await instanceServiceClient.updateDataSource(
          {
            instance: instance.name,
            dataSource: ds,
            updateMask,
            validateOnly: true,
          },
          {
            silent: true,
          }
        );
        return ok();
      } catch (err) {
        return fail(ds.host, err);
      }
    }
  }
};

// getOriginalEditState returns the origin instance data including
// basic information, admin data source and read-only data source.
const getOriginalEditState = () => {
  return {
    basicInfo: extractBasicInfo(props.instance),
    dataSources: extractDataSourceEditState(props.instance).dataSources,
  };
};

const calcDataSourceUpdateMask = (
  editing: DataSource,
  original: DataSource,
  editState: EditDataSource
) => {
  const updateMask = new Set(
    calcUpdateMask(editing, original, true /* toSnakeCase */)
  );
  const { useEmptyPassword, updateSsh, updateSsl } = editState;
  if (useEmptyPassword) {
    // We need to implicitly set "password" need to be updated
    // if the "use empty password" option if checked
    editing.password = "";
    updateMask.add("password");
  }
  if (updateSsl) {
    updateMask.add("ssl_ca");
    updateMask.add("ssl_key");
    updateMask.add("ssl_cert");
  }
  if (updateSsh) {
    updateMask.add("ssh_host");
    updateMask.add("ssh_port");
    updateMask.add("ssh_user");
    updateMask.add("ssh_password");
    updateMask.add("ssh_private_key");
  }

  return Array.from(updateMask);
};

const extractDataSourceFromEdit = (
  instance: Instance,
  edit: EditDataSource
): DataSource => {
  const ds = cloneDeep(
    omit(
      edit,
      "pendingCreate",
      "updatedPassword",
      "useEmptyPassword",
      "updateSsl",
      "updateSsh"
    )
  );
  if (edit.updatedPassword) {
    ds.password = edit.updatedPassword;
  }
  if (edit.useEmptyPassword) {
    ds.password = "";
  }

  // Clean up unused fields for certain engine types.
  if (!showDatabase.value) {
    ds.database = "";
  }
  if (instance.engine !== Engine.ORACLE) {
    ds.sid = "";
    ds.serviceName = "";
  }
  if (instance.engine !== Engine.MONGODB) {
    ds.srv = false;
    ds.authenticationDatabase = "";
  }
  if (!showSSH.value) {
    ds.sshHost = "";
    ds.sshPort = "";
    ds.sshUser = "";
    ds.sshPassword = "";
    ds.sshPrivateKey = "";
  }
  if (!showSSL.value) {
    ds.sslCa = "";
    ds.sslCert = "";
    ds.sslKey = "";
  }

  return ds;
};

const checkRODataSourceFeature = (instance: Instance) => {
  // This is to
  // - Disallow creating any new RO data sources
  // - Disallow updating any existed RO data sources
  // if feature is not available.

  // Early pass if the feature is available.
  if (hasReadonlyReplicaFeature.value) {
    return true;
  }

  if (readonlyDataSourceList.value.length === 0) {
    // Not creating or editing any RO data source
    return true;
  }

  const checkOne = (ds: EditDataSource) => {
    if (ds.pendingCreate) {
      // Disallowed to create any new RO data sources
      return false;
    } else {
      const editing = extractDataSourceFromEdit(instance, ds);
      const original = instance.dataSources.find((d) => d.id === ds.id);
      if (original) {
        const updateMask = calcDataSourceUpdateMask(editing, original, ds);
        // Disallowed to update any existed RO data source
        if (updateMask.length > 0) {
          return false;
        }
      }
    }
    return true;
  };
  // Need to check all RO data sources
  return readonlyDataSourceList.value.every(checkOne);
};

const changeInstanceActivation = async (on: boolean) => {
  basicInfo.value.activation = on;
  if (props.instance) {
    const instancePatch = {
      ...props.instance,
      activation: on,
    };
    await instanceV1Store.updateInstance(instancePatch, ["activation"]);
  }
};
</script>
