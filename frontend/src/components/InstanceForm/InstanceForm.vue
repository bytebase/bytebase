<template>
  <component :is="drawer ? DrawerContent : 'div'" v-bind="bindings">
    <div class="space-y-6 divide-y divide-block-border">
      <div class="divide-y divide-block-border w-[850px]">
        <div v-if="isCreating" class="w-full mt-4 mb-6 grid grid-cols-4 gap-2">
          <template v-for="engine in engineList" :key="engine">
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
              v-for="type in mongodbConnectionStringSchemaList"
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
                :placeholder="snowflakeExtraLinkPlaceHolder"
              />
            </template>
          </div>
        </div>

        <!-- Connection Info -->
        <p class="mt-6 pt-4 w-full text-lg leading-6 font-medium text-gray-900">
          {{ $t("instance.connection-info") }}
        </p>

        <div
          v-if="!isCreating && !hasReadOnlyDataSource && allowEdit"
          class="mt-2 flex flex-row justify-start items-center bg-yellow-50 border-none rounded-lg p-2 px-3"
        >
          <heroicons-outline:exclamation
            class="h-6 w-6 text-yellow-400 flex-shrink-0 mr-1"
          />
          <span class="text-yellow-800 text-sm">
            {{ $t("instance.no-read-only-data-source-warn") }}
          </span>
          <button
            type="button"
            class="btn-normal ml-4 text-sm"
            @click.prevent="handleCreateRODataSource"
          >
            {{ $t("common.create") }}
          </button>
        </div>

        <div
          class="mt-2 grid grid-cols-1 gap-y-2 gap-x-4 border-none sm:grid-cols-3"
        >
          <NTabs
            v-if="!isCreating"
            v-model:value="state.currentDataSourceType"
            class="sm:col-span-3"
            type="line"
          >
            <NTab :name="DataSourceType.ADMIN">
              {{ $t("common.admin") }}
            </NTab>
            <NTab
              :name="DataSourceType.READ_ONLY"
              class="relative"
              :disabled="!hasReadOnlyDataSource"
            >
              <span>{{ $t("common.read-only") }}</span>
              <BBButtonConfirm
                v-if="hasReadOnlyDataSource"
                :style="'DELETE'"
                class="absolute left-full ml-1"
                :require-confirm="!readonlyDataSource?.pendingCreate"
                :ok-text="$t('common.delete')"
                :confirm-title="
                  $t('data-source.delete-read-only-data-source') + '?'
                "
                @confirm="handleDeleteRODataSource"
              />
            </NTab>
          </NTabs>

          <template v-if="basicInfo.engine !== Engine.SPANNER">
            <CreateDataSourceExample
              class-name="sm:col-span-3 border-none mt-2"
              :create-instance-flag="isCreating"
              :engine="basicInfo.engine"
              :data-source-type="state.currentDataSourceType"
            />
            <div class="mt-2 sm:col-span-1 sm:col-start-1">
              <label for="username" class="textlabel block">
                {{ $t("common.username") }}
              </label>
              <!-- For mysql, username can be empty indicating anonymous user.
              But it's a very bad practice to use anonymous user for admin operation,
              thus we make it REQUIRED here.-->
              <input
                id="username"
                v-model="currentDataSource.username"
                name="username"
                type="text"
                class="textfield mt-1 w-full"
                :disabled="!allowEdit"
                :placeholder="
                  basicInfo.engine === Engine.CLICKHOUSE
                    ? $t('common.default')
                    : ''
                "
              />
            </div>
            <div class="mt-2 sm:col-span-1 sm:col-start-1">
              <div class="flex flex-row items-center space-x-2">
                <label for="password" class="textlabel block">
                  {{ $t("common.password") }}
                </label>
                <BBCheckbox
                  v-if="!isCreating && allowUsingEmptyPassword"
                  :title="$t('common.empty')"
                  :value="currentDataSource.useEmptyPassword"
                  :disabled="!allowEdit"
                  @toggle="toggleUseEmptyPassword"
                />
              </div>
              <input
                id="password"
                name="password"
                type="text"
                class="textfield mt-1 w-full"
                autocomplete="off"
                :placeholder="
                  currentDataSource.useEmptyPassword
                    ? $t('instance.no-password')
                    : $t('instance.password-write-only')
                "
                :disabled="!allowEdit || currentDataSource.useEmptyPassword"
                :value="
                  currentDataSource.useEmptyPassword
                    ? ''
                    : currentDataSource.updatedPassword
                "
                @input="
                  currentDataSource.updatedPassword = trimInputValue(
                    $event.target
                  )
                "
              />
            </div>
          </template>
          <SpannerCredentialInput
            v-else
            v-model:value="currentDataSource.updatedPassword"
            :write-only="!isCreating"
            class="mt-2 sm:col-span-3 sm:col-start-1"
          />

          <template v-if="basicInfo.engine === Engine.ORACLE">
            <OracleSIDAndServiceNameInput
              v-model:sid="currentDataSource.sid"
              v-model:service-name="currentDataSource.serviceName"
              :allow-edit="allowEdit"
            />
          </template>

          <template v-if="showAuthenticationDatabase">
            <div class="sm:col-span-1 sm:col-start-1">
              <div class="flex flex-row items-center space-x-2">
                <label for="authenticationDatabase" class="textlabel block">
                  {{ $t("instance.authentication-database") }}
                </label>
              </div>
              <input
                id="authenticationDatabase"
                name="authenticationDatabase"
                type="text"
                class="textfield mt-1 w-full"
                autocomplete="off"
                placeholder="admin"
                :value="currentDataSource.authenticationDatabase"
                @input="
                  currentDataSource.authenticationDatabase = trimInputValue(
                    $event.target
                  )
                "
              />
            </div>
          </template>

          <template
            v-if="
              state.currentDataSourceType === DataSourceType.READ_ONLY &&
              (hasReadonlyReplicaHost || hasReadonlyReplicaPort)
            "
          >
            <div
              v-if="hasReadonlyReplicaHost"
              class="mt-2 sm:col-span-1 sm:col-start-1"
            >
              <div class="flex flex-row items-center space-x-2">
                <label for="host" class="textlabel block">
                  {{ $t("data-source.read-replica-host") }}
                </label>
              </div>
              <input
                id="host"
                name="host"
                type="text"
                class="textfield mt-1 w-full"
                autocomplete="off"
                :value="currentDataSource.host"
                @input="handleCurrentDataSourceHostInput"
              />
            </div>

            <div
              v-if="hasReadonlyReplicaPort"
              class="mt-2 sm:col-span-1 sm:col-start-1"
            >
              <div class="flex flex-row items-center space-x-2">
                <label for="port" class="textlabel block">
                  {{ $t("data-source.read-replica-port") }}
                </label>
              </div>
              <input
                id="port"
                name="port"
                type="text"
                class="textfield mt-1 w-full"
                autocomplete="off"
                :value="currentDataSource.port"
                @input="handleCurrentDataSourcePortInput"
              />
            </div>
          </template>

          <div v-if="showDatabase" class="mt-2 sm:col-span-1 sm:col-start-1">
            <label for="database" class="textlabel block">
              {{ $t("common.database") }}
            </label>
            <input
              id="database"
              v-model="currentDataSource.database"
              name="database"
              type="text"
              class="textfield mt-1 w-full"
              :disabled="!allowEdit"
              :placeholder="$t('common.database')"
            />
          </div>

          <div v-if="showSSL" class="mt-2 sm:col-span-3 sm:col-start-1">
            <div class="flex flex-row items-center">
              <label for="ssl" class="textlabel block">
                {{ $t("data-source.ssl-connection") }}
              </label>
            </div>
            <template v-if="currentDataSource.pendingCreate">
              <SslCertificateForm
                :value="currentDataSource"
                @change="handleCurrentDataSourceSslChange"
              />
            </template>
            <template v-else>
              <template v-if="currentDataSource.updateSsl">
                <SslCertificateForm
                  :value="currentDataSource"
                  @change="handleCurrentDataSourceSslChange"
                />
              </template>
              <template v-else>
                <button
                  class="btn-normal mt-2"
                  :disabled="!allowEdit"
                  @click.prevent="handleEditSsl"
                >
                  {{ $t("common.edit") }} - {{ $t("common.write-only") }}
                </button>
              </template>
            </template>
          </div>

          <div v-if="showSSH" class="mt-2 sm:col-span-3 sm:col-start-1">
            <div class="flex flex-row items-center gap-x-1">
              <label for="ssh" class="textlabel block">
                {{ $t("data-source.ssh-connection") }}
              </label>
              <FeatureBadge
                feature="bb.feature.instance-ssh-connection"
                :instance="instance"
              />
            </div>
            <template v-if="currentDataSource.pendingCreate">
              <SshConnectionForm
                :value="currentDataSource"
                :instance="instance"
                @change="handleCurrentDataSourceSshChange"
              />
            </template>
            <template v-else>
              <template v-if="currentDataSource.updateSsh">
                <SshConnectionForm
                  :value="currentDataSource"
                  :instance="instance"
                  @change="handleCurrentDataSourceSshChange"
                />
              </template>
              <template v-else>
                <button
                  class="btn-normal mt-2"
                  :disabled="!allowEdit"
                  @click.prevent="handleEditSsh"
                >
                  {{ $t("common.edit") }} - {{ $t("common.write-only") }}
                </button>
              </template>
            </template>
          </div>
        </div>

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
    :open="state.showFeatureModal"
    :instance="instance"
    @cancel="state.showFeatureModal = false"
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
import { computed, reactive, PropType, ref, watch, onMounted } from "vue";
import { cloneDeep, isEqual, omit } from "lodash-es";
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";
import { Status } from "nice-grpc-common";
import { useRouter } from "vue-router";

import {
  hasWorkspacePermissionV1,
  isDev,
  isValidSpannerHost,
  extractInstanceResourceName,
  engineNameV1,
  instanceV1HasSSL,
  instanceV1HasSSH,
  supportedEngineV1List,
  instanceV1Slug,
  calcUpdateMask,
} from "@/utils";
import {
  UNKNOWN_ID,
  ResourceId,
  ValidatedMessage,
  DataSourceOptions,
  emptyDataSource,
  UNKNOWN_INSTANCE_NAME,
  UNKNOWN_ENVIRONMENT_NAME,
  unknownEnvironment,
} from "@/types";
import {
  pushNotification,
  useCurrentUserV1,
  useSettingV1Store,
  useActuatorV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
  useGracefulRequest,
} from "@/store";
import { getErrorCode, extractGrpcErrorMessage } from "@/utils/grpcweb";
import EnvironmentSelect from "@/components/EnvironmentSelect.vue";
import OracleSyncModeInput from "./OracleSyncModeInput.vue";
import SslCertificateForm from "./SslCertificateForm.vue";
import SshConnectionForm from "./SshConnectionForm.vue";
import SpannerHostInput from "./SpannerHostInput.vue";
import SpannerCredentialInput from "./SpannerCredentialInput.vue";
import OracleSIDAndServiceNameInput from "./OracleSIDAndServiceNameInput.vue";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { DrawerContent, InstanceV1EngineIcon } from "@/components/v2";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import {
  DataSource,
  DataSourceType,
  Instance,
  InstanceOptions,
} from "@/types/proto/v1/instance_service";
import { Engine, State } from "@/types/proto/v1/common";
import { instanceServiceClient } from "@/grpcweb";
import { PlanType } from "@/types/proto/v1/subscription_service";

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

interface EditDataSource extends DataSource {
  pendingCreate: boolean;
  updatedPassword: string;
  useEmptyPassword?: boolean;
  updateSsl?: boolean;
  updateSsh?: boolean;
}

type BasicInfo = Omit<Instance, "dataSources" | "engineVersion">;

interface LocalState {
  currentDataSourceType: DataSourceType;
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
const currentUserV1 = useCurrentUserV1();
const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();

const state = reactive<LocalState>({
  currentDataSourceType: DataSourceType.ADMIN,
  showFeatureModal: false,
  isTestingConnection: false,
  isRequesting: false,
  showCreateInstanceWarningModal: false,
  createInstanceWarning: "",
});

const hasReadonlyReplicaFeature = computed(() => {
  return subscriptionStore.hasInstanceFeature(
    "bb.feature.read-replica-connection",
    props.instance
  );
});

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

const extractBasicInfo = (instance: Instance | undefined): BasicInfo => {
  return {
    uid: instance?.uid ?? String(UNKNOWN_ID),
    name: instance?.name ?? UNKNOWN_INSTANCE_NAME,
    state: instance?.state ?? State.ACTIVE,
    title: instance?.title ?? t("instance.new-instance"),
    engine: instance?.engine ?? Engine.MYSQL,
    externalLink: instance?.externalLink ?? "",
    environment: instance?.environment ?? UNKNOWN_ENVIRONMENT_NAME,
    activation: instance
      ? instance.activation
      : subscriptionStore.currentPlan !== PlanType.FREE &&
        availableLicenseCount.value > 0,
    options: instance?.options
      ? cloneDeep(instance.options)
      : {
          // default to false (Manage based on database, aka CDB + non-CDB)
          schemaTenantMode: false,
        },
  };
};

const basicInfo = ref<BasicInfo>(extractBasicInfo(props.instance));

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

const getDataSourceByType = (
  instance: Instance | undefined,
  type: DataSourceType
) => {
  return instance?.dataSources.find((ds) => ds.type === type);
};

const extractAdminDataSource = (
  instance: Instance | undefined
): EditDataSource => {
  const ds = getDataSourceByType(instance, DataSourceType.ADMIN);
  return {
    ...cloneDeep(ds ?? emptyDataSource()),
    pendingCreate: ds === undefined,
    updatedPassword: "",
    useEmptyPassword: false,
  };
};
// We only support one admin data source and one read-only data source.
const adminDataSource = ref<EditDataSource>(
  extractAdminDataSource(props.instance)
);

const extractReadOnlyDataSource = (
  instance: Instance | undefined
): EditDataSource | undefined => {
  const ds = getDataSourceByType(instance, DataSourceType.READ_ONLY);
  if (ds) {
    return {
      ...cloneDeep(ds),
      pendingCreate: ds === undefined,
      updatedPassword: "",
      useEmptyPassword: false,
    };
  }
  return undefined;
};
const readonlyDataSource = ref<EditDataSource | undefined>(
  extractReadOnlyDataSource(props.instance)
);

const getDefaultPort = (engine: Engine) => {
  if (engine === Engine.CLICKHOUSE) {
    return "9000";
  } else if (engine === Engine.POSTGRES) {
    return "5432";
  } else if (engine === Engine.SNOWFLAKE) {
    return "443";
  } else if (engine === Engine.TIDB) {
    return "4000";
  } else if (engine === Engine.MONGODB) {
    return "27017";
  } else if (engine === Engine.REDIS) {
    return "6379";
  } else if (engine === Engine.ORACLE) {
    return "1521";
  } else if (engine === Engine.MSSQL) {
    return "1433";
  } else if (engine === Engine.REDSHIFT) {
    return "5439";
  } else if (engine === Engine.OCEANBASE) {
    return "2883";
  }
  return "3306";
};

const isCreating = computed(() => props.instance === undefined);

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
      adminDataSource.value.port = getDefaultPort(basicInfo.value.engine);
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

const engineList = computed(() => {
  return supportedEngineV1List();
});

const outboundIpList = computed(() => {
  if (!settingV1Store.workspaceProfileSetting) {
    return "";
  }
  return settingV1Store.workspaceProfileSetting.outboundIpList.join(",");
});

const EngineIconPath: Record<number, string> = {
  [Engine.MYSQL]: new URL("@/assets/db-mysql.png", import.meta.url).href,
  [Engine.POSTGRES]: new URL("@/assets/db-postgres.png", import.meta.url).href,
  [Engine.TIDB]: new URL("@/assets/db-tidb.png", import.meta.url).href,
  [Engine.SNOWFLAKE]: new URL("@/assets/db-snowflake.png", import.meta.url)
    .href,
  [Engine.CLICKHOUSE]: new URL("@/assets/db-clickhouse.png", import.meta.url)
    .href,
  [Engine.MONGODB]: new URL("@/assets/db-mongodb.png", import.meta.url).href,
  [Engine.SPANNER]: new URL("@/assets/db-spanner.png", import.meta.url).href,
  [Engine.REDIS]: new URL("@/assets/db-redis.png", import.meta.url).href,
  [Engine.ORACLE]: new URL("@/assets/db-oracle.svg", import.meta.url).href,
  [Engine.MSSQL]: new URL("@/assets/db-mssql.svg", import.meta.url).href,
  [Engine.REDSHIFT]: new URL("@/assets/db-redshift.svg", import.meta.url).href,
  [Engine.MARIADB]: new URL("@/assets/db-mariadb.png", import.meta.url).href,
  [Engine.OCEANBASE]: new URL("@/assets/db-oceanbase.png", import.meta.url)
    .href,
};

const mongodbConnectionStringSchemaList = ["mongodb://", "mongodb+srv://"];

const currentMongoDBConnectionSchema = computed(() => {
  return adminDataSource.value.srv === false
    ? mongodbConnectionStringSchemaList[0]
    : mongodbConnectionStringSchemaList[1];
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

const allowEdit = computed(() => {
  if (isCreating.value) return true;

  return (
    props.instance?.state === State.ACTIVE &&
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  );
});

const allowEditPort = computed(() => {
  // MongoDB doesn't support specify port if using srv record.
  return !(
    basicInfo.value.engine === Engine.MONGODB && currentDataSource.value.srv
  );
});

const allowUsingEmptyPassword = computed(() => {
  return basicInfo.value.engine !== Engine.SPANNER;
});

const valueChanged = computed(() => {
  const original = getOriginalEditState();
  const editing = {
    basicInfo: basicInfo.value,
    adminDataSource: adminDataSource.value,
    readonlyDataSource: readonlyDataSource.value,
  };
  return !isEqual(editing, original);
});

const defaultPort = computed(() => {
  return getDefaultPort(basicInfo.value.engine);
});

const currentDataSource = computed((): EditDataSource => {
  if (state.currentDataSourceType === DataSourceType.ADMIN) {
    return adminDataSource.value;
  } else if (state.currentDataSourceType === DataSourceType.READ_ONLY) {
    return readonlyDataSource.value!;
  } else {
    throw new Error("Unknown data source type");
  }
});

const hasReadOnlyDataSource = computed(() => {
  return readonlyDataSource.value !== undefined;
});

const snowflakeExtraLinkPlaceHolder =
  "https://us-west-1.console.aws.amazon.com/rds/home?region=us-west-1#database:id=mysql-instance-foo;is-cluster=false";

const instanceLink = computed(() => {
  if (basicInfo.value.engine === Engine.SNOWFLAKE) {
    if (adminDataSource.value.host) {
      return `https://${
        adminDataSource.value.host.split("@")[0]
      }.snowflakecomputing.com/console`;
    }
  }
  return basicInfo.value.externalLink ?? "";
});

const hasReadonlyReplicaHost = computed((): boolean => {
  return basicInfo.value.engine !== Engine.SPANNER;
});

const hasReadonlyReplicaPort = computed((): boolean => {
  return basicInfo.value.engine !== Engine.SPANNER;
});

const showDatabase = computed((): boolean => {
  return (
    (basicInfo.value.engine === Engine.POSTGRES ||
      basicInfo.value.engine === Engine.REDSHIFT) &&
    state.currentDataSourceType === DataSourceType.ADMIN
  );
});
const showAuthenticationDatabase = computed((): boolean => {
  return basicInfo.value.engine === Engine.MONGODB;
});
const showSSL = computed((): boolean => {
  return instanceV1HasSSL(basicInfo.value.engine);
});
const showSSH = computed((): boolean => {
  return instanceV1HasSSH(basicInfo.value.engine);
});

const allowUpdate = computed((): boolean => {
  if (!valueChanged.value) {
    return false;
  }
  if (basicInfo.value.engine === Engine.SPANNER) {
    if (!isValidSpannerHost(adminDataSource.value.host)) {
      return false;
    }
    if (
      readonlyDataSource.value &&
      !isValidSpannerHost(readonlyDataSource.value.host)
    ) {
      return false;
    }
  }
  return true;
});

const isEngineBeta = (engine: Engine): boolean => {
  return [
    Engine.ORACLE,
    Engine.MSSQL,
    Engine.REDSHIFT,
    Engine.MARIADB,
    Engine.OCEANBASE,
  ].includes(engine);
};

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
  switch ((event.target as HTMLInputElement).value) {
    case mongodbConnectionStringSchemaList[0]:
      currentDataSource.value.srv = false;
      break;
    case mongodbConnectionStringSchemaList[1]:
      // MongoDB doesn't support specify port if using srv record.
      currentDataSource.value.port = "";
      currentDataSource.value.srv = true;
      break;
    default:
      currentDataSource.value.srv = false;
  }
};

const handleCurrentDataSourceHostInput = (event: Event) => {
  if (currentDataSource.value.type === DataSourceType.READ_ONLY) {
    if (!hasReadonlyReplicaFeature.value) {
      if (currentDataSource.value.host || currentDataSource.value.port) {
        currentDataSource.value.host = adminDataSource.value.host;
        currentDataSource.value.port = adminDataSource.value.port;
        state.showFeatureModal = true;
        return;
      }
    }
  }

  currentDataSource.value.host = trimInputValue(event.target);
};

const handleCurrentDataSourcePortInput = (event: Event) => {
  if (currentDataSource.value.type === DataSourceType.READ_ONLY) {
    if (!hasReadonlyReplicaFeature.value) {
      if (currentDataSource.value.host || currentDataSource.value.port) {
        currentDataSource.value.host = adminDataSource.value.host;
        currentDataSource.value.port = adminDataSource.value.port;
        state.showFeatureModal = true;
        return;
      }
    }
  }

  currentDataSource.value.port = trimInputValue(event.target);
};

const toggleUseEmptyPassword = (on: boolean) => {
  currentDataSource.value.useEmptyPassword = on;
  if (on) {
    currentDataSource.value.updatedPassword = "";
  }
};

const handleEditSsl = () => {
  const curr = currentDataSource.value;
  curr.sslCa = "";
  curr.sslCert = "";
  curr.sslKey = "";
  curr.updateSsl = true;
};

const handleEditSsh = () => {
  const curr = currentDataSource.value;
  curr.sshHost = "";
  curr.sshPort = "";
  curr.sshUser = "";
  curr.sshPassword = "";
  curr.sshPrivateKey = "";
  curr.updateSsh = true;
};

const handleCurrentDataSourceSslChange = (
  value: Partial<Pick<DataSource, "sslCa" | "sslCert" | "sslKey">>
) => {
  Object.assign(currentDataSource.value, value);
  currentDataSource.value.updateSsl = true;
};

const handleCurrentDataSourceSshChange = (
  value: Partial<
    Pick<
      DataSourceOptions,
      "sshHost" | "sshPort" | "sshUser" | "sshPassword" | "sshPrivateKey"
    >
  >
) => {
  Object.assign(currentDataSource.value, value);
  currentDataSource.value.updateSsh = true;
};

const handleCreateRODataSource = () => {
  if (isCreating.value) {
    return;
  }

  const tempDataSource: DataSource = {
    ...emptyDataSource(),
    title: "Read-only data source",
    type: DataSourceType.READ_ONLY,
    host: adminDataSource.value.host,
    port: adminDataSource.value.port,
    database: adminDataSource.value.database,
  };
  if (basicInfo.value.engine === Engine.SPANNER) {
    tempDataSource.host = adminDataSource.value.host;
  }
  readonlyDataSource.value = {
    ...tempDataSource,
    pendingCreate: true,
    updatedPassword: "",
    useEmptyPassword: false,
  };
  state.currentDataSourceType = DataSourceType.READ_ONLY;
};

const handleDeleteRODataSource = async () => {
  if (!readonlyDataSource.value) {
    return;
  }

  if (readonlyDataSource.value.pendingCreate) {
    state.currentDataSourceType = DataSourceType.ADMIN;
    readonlyDataSource.value = undefined;
  } else {
    const { instance } = props;
    if (!instance) return;
    const ds = getDataSourceByType(instance, DataSourceType.READ_ONLY);
    if (!ds) return;

    const updated = await instanceV1Store.deleteDataSource(instance, ds);
    state.currentDataSourceType = DataSourceType.ADMIN;
    await updateEditState(updated);
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
  adminDataSource.value = extractAdminDataSource(instance);
  readonlyDataSource.value = extractReadOnlyDataSource(instance);

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
    state.showFeatureModal = true;
    return;
  }

  // When clicking **Update** we may have more than one thing to do (if needed)
  // 1. Patch the instance itself.
  // 2. Update the admin datasource.
  // 3. Create OR update a read-only data source.
  const maybeUpdateInstance = async () => {
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
  const maybeUpsertReadonlyDataSource = async () => {
    if (!readonlyDataSource.value) return;
    const editing = extractDataSourceFromEdit(
      instance,
      readonlyDataSource.value
    );
    if (readonlyDataSource.value.pendingCreate) {
      return await instanceV1Store.createDataSource(instance, editing);
    } else {
      const original = instance.dataSources.find(
        (ds) => ds.type === DataSourceType.READ_ONLY
      );
      return await updateDataSource(
        editing,
        original,
        readonlyDataSource.value
      );
    }
  };

  state.isRequesting = true;
  try {
    await useGracefulRequest(async () => {
      await maybeUpdateInstance();
      await maybeUpdateAdminDataSource();
      await maybeUpsertReadonlyDataSource();

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
    // CreateInstanceRequest.validateOnly = true
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
    const ds = extractDataSourceFromEdit(instance, currentDataSource.value);
    if (currentDataSource.value.pendingCreate) {
      // When read-only data source is about to be created, use
      // AddDataSourceRequest.validateOnly = true
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
      // UpdateDataSourceRequest.validateOnly = true
      try {
        const original = instance.dataSources.find(
          (ds) => ds.type === currentDataSource.value.type
        )!;
        const updateMask = calcDataSourceUpdateMask(
          ds,
          original,
          currentDataSource.value
        );
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
    adminDataSource: extractAdminDataSource(props.instance),
    readonlyDataSource: extractReadOnlyDataSource(props.instance),
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
  // Early pass if the feature is available.
  if (hasReadonlyReplicaFeature.value) {
    return true;
  }

  // Not editing/creating
  if (!readonlyDataSource.value) {
    return true;
  }

  if (readonlyDataSource.value.pendingCreate) {
    // pre-flight feature guard check for creating RO datasource
    return false;
  } else {
    // pre-flight feature guard check for updating RO datasource
    const editing = extractDataSourceFromEdit(
      instance,
      readonlyDataSource.value
    );
    const original = instance.dataSources.find(
      (ds) => ds.type === DataSourceType.READ_ONLY
    );
    if (original) {
      const updateMask = calcDataSourceUpdateMask(
        editing,
        original,
        readonlyDataSource.value
      );
      if (updateMask.length > 0) {
        return false;
      }
    }
  }
  return true;
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
