<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="">
      <!-- Instance Name -->
      <div class="grid gap-y-6 gap-x-4 grid-cols-4">
        <div class="col-span-2 col-start-2 w-64">
          <label for="environment" class="textlabel">
            {{ $t("common.environment") }} <span style="color: red">*</span>
          </label>
          <!-- eslint-disable vue/attribute-hyphenation -->
          <EnvironmentSelect
            id="environment"
            class="mt-1 w-full"
            name="environment"
            :disabled="!allowConfigure"
            :selectedId="state.environmentId"
            @select-environment-id="
              (environmentId) => {
                updateState('environmentId', environmentId);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
          <label for="instance" class="textlabel">
            {{ $t("common.instance") }} <span class="text-red-600">*</span>
          </label>
          <!-- eslint-disable vue/attribute-hyphenation -->
          <InstanceSelect
            id="instance"
            class="mt-1 w-full"
            name="instance"
            :disabled="!allowConfigure"
            :selectedId="state.instanceId"
            @select-instance-id="
              (instanceId) => {
                updateState('instanceId', instanceId);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
          <label for="database" class="textlabel">
            {{ $t("common.database") }} <span class="text-red-600">*</span>
          </label>
          <!-- eslint-disable vue/attribute-hyphenation -->
          <DatabaseSelect
            id="database"
            class="mt-1 w-full"
            name="database"
            :disabled="!allowConfigure"
            :mode="'INSTANCE'"
            :instanceId="state.instanceId"
            :selectedId="state.databaseId"
            @select-database-id="
              (databaseId) => {
                updateState('databaseId', databaseId);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
          <label for="datasource" class="textlabel">
            {{ $t("common.data-source") }} <span class="text-red-600">*</span>
          </label>
          <!-- eslint-disable vue/attribute-hyphenation -->
          <DataSourceSelect
            id="datasource"
            class="mt-1 w-full"
            name="datasource"
            :disabled="!allowConfigure"
            :database="database"
            :selectedId="state.dataSourceId"
            @select-data-source-id="
              (dataSourceId) => {
                updateState('dataSourceId', dataSourceId);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
          <label for="user" class="textlabel">
            {{ $t("common.user") }} <span class="text-red-600">*</span>
            <span class="ml-2 text-error text-xs">
              {{ state.granteeError }}
            </span>
          </label>
          <!-- DBA and Owner always have all access, so we only need to grant to developer -->
          <!-- eslint-disable vue/attribute-hyphenation -->
          <MemberSelect
            id="user"
            class="mt-1 w-full"
            name="user"
            :allowed-role-list="['DEVELOPER']"
            :disabled="!allowUpdateDataSourceMember"
            :selectedId="state.granteeId"
            @select-principal-id="
              (principalId) => {
                updateState('granteeId', principalId);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
          <label for="issue" class="textlabel">
            {{ $t("common.issue") }}
          </label>
          <div class="mt-1 relative rounded-md shadow-sm">
            <div
              class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"
            >
              <span class="text-accent font-semibold sm:text-sm"
                >{{ $t("intro.issue") }}/</span
              >
            </div>
            <div class="flex flex-row space-x-2 items-center">
              <input
                id="issue"
                class="textfield w-full pl-12"
                name="issue"
                type="number"
                placeholder="$t('datasource.your-issue-id-e-g-1234')"
                :disabled="!allowUpdateIssueId"
                :value="state.issueId"
                @input="updateState('issueId', $event.target.value)"
              />
              <template v-if="issueLink">
                <router-link
                  :to="issueLink"
                  target="_blank"
                  class="ml-2 normal-link text-sm"
                >
                  {{ $t("common.view") }}
                </router-link>
              </template>
            </div>
          </div>
        </div>
      </div>
    </div>
    <!-- Action Button Group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="doGrant"
      >
        {{ $t("common.grant") }}
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType, defineComponent } from "vue";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import DataSourceSelect from "../components/DataSourceSelect.vue";
import InstanceSelect from "../components/InstanceSelect.vue";
import MemberSelect from "../components/MemberSelect.vue";
import {
  DatabaseId,
  DataSource,
  DataSourceId,
  DataSourceMember,
  DataSourceMemberCreate,
  EnvironmentId,
  InstanceId,
  PrincipalId,
  Issue,
  IssueId,
} from "../types";
import { issueSlug } from "../utils";
import { useI18n } from "vue-i18n";
import { pushNotification, useDatabaseStore, useIssueStore } from "@/store";

interface LocalState {
  environmentId?: EnvironmentId;
  instanceId?: InstanceId;
  databaseId?: DatabaseId;
  dataSourceId?: DataSourceId;
  granteeId?: PrincipalId;
  granteeError: string;
  issueId?: IssueId;
}

export default defineComponent({
  name: "DataSourceMemberCreateForm",
  components: {
    EnvironmentSelect,
    DatabaseSelect,
    DataSourceSelect,
    MemberSelect,
    InstanceSelect,
  },
  props: {
    dataSource: {
      type: Object as PropType<DataSource>,
    },
    principalId: {
      type: Number,
    },
    issueId: {
      type: Number,
    },
  },
  emits: ["submit", "cancel"],
  setup(props, { emit }) {
    const databaseStore = useDatabaseStore();

    const state = reactive<LocalState>({
      environmentId: props.dataSource
        ? props.dataSource.instance.environment.id
        : undefined,
      instanceId: props.dataSource ? props.dataSource.instance.id : undefined,
      databaseId: props.dataSource ? props.dataSource.database.id : undefined,
      dataSourceId: props.dataSource ? props.dataSource.id : undefined,
      granteeId: props.principalId,
      granteeError: "",
      issueId: props.issueId,
    });

    const { t } = useI18n();

    const database = computed(() => {
      return state.databaseId
        ? databaseStore.getDatabaseById(state.databaseId)
        : undefined;
    });

    const dataSource = computed(() => {
      if (props.dataSource) {
        return props.dataSource;
      }

      if (state.dataSourceId) {
        if (state.databaseId) {
          const database = databaseStore.getDatabaseById(state.databaseId);
          if (database) {
            return database.dataSourceList.find(
              (item: DataSource) => item.id == state.dataSourceId
            );
          }
        }
      }

      return undefined;
    });

    // We only configure if data source is not specified.
    const allowConfigure = computed(() => {
      return !props.dataSource;
    });

    const allowUpdateDataSourceMember = computed(() => {
      return !props.principalId && state.dataSourceId;
    });

    const allowUpdateIssueId = computed(() => {
      return !props.issueId;
    });

    const allowCreate = computed(() => {
      return state.dataSourceId && state.granteeId && !state.granteeError;
    });

    const issueLink = computed((): string => {
      if (state.issueId) {
        // We intentionally not to validate whether the issueId is legit, we will do the validation
        // when actually trying to create the database.
        return `/issue/${state.issueId}`;
      }
      return "";
    });

    const validateGrantee = () => {
      if (state.granteeId) {
        const member = dataSource.value.memberList.find(
          (item: DataSourceMember) => item.principal.id == state.granteeId
        );
        if (member) {
          state.granteeError = t(
            "datasource.member-principal-name-already-exists",
            [member.principal.name]
          );
        } else {
          state.granteeError = "";
        }
      }
    };

    const updateState = (field: string, value: number) => {
      if (field == "environmentId") {
        state.environmentId = value;
      } else if (field == "instanceId") {
        state.instanceId = value;
      } else if (field == "databaseId") {
        state.databaseId = value;
      } else if (field == "dataSourceId") {
        state.dataSourceId = value;
      } else if (field == "granteeId") {
        state.granteeId = value;
        validateGrantee();
      } else if (field == "issueId") {
        state.issueId = value;
      }
    };

    const cancel = () => {
      emit("cancel");
    };

    const doGrant = async () => {
      // If issueId id provided, we check its existence first.
      // We only set the issueId if it's valid.
      let linkedIssue: Issue | undefined = undefined;
      if (state.issueId) {
        try {
          linkedIssue = await useIssueStore().fetchIssueById(state.issueId);
        } catch (err) {
          console.warn(`Unable to fetch linked issue id ${state.issueId}`, err);
        }
      }

      const newDataSourceMember: DataSourceMemberCreate = {
        principalId: state.granteeId!,
        issueId: linkedIssue?.id,
      };
      // TODO (yw): there is no action named createDataSourceMember
      // TODO (jim): DataSourceMemberTable, DataSourceMemberForm, and DataSourceDetail now have no
      //   any entrance. These files should probably be removed in the future.
      // store
      //   .dispatch("dataSource/createDataSourceMember", {
      //     dataSourceId: state.dataSourceId,
      //     databaseId: state.databaseId,
      //     newDataSourceMember,
      //   })
      //   .then((dataSource: DataSource) => {
      //     emit("submit", dataSource);

      //     const addedMember = dataSource.memberList.find(
      //       (item: DataSourceMember) => {
      //         return item.principal.id == state.granteeId;
      //       }
      //     );
      //     pushNotification({
      //       module: "bytebase",
      //       style: "SUCCESS",
      //       title: t(
      //         "datasource.successfully-granted-datasource-name-to-addedmember-principal-name",
      //         [dataSource.name, addedMember!.principal.name]
      //       ),
      //       description: linkedIssue
      //         ? t(
      //             "datasource.we-also-linked-the-granted-database-to-the-requested-issue-linkedissue-name",
      //             [linkedIssue.name]
      //           )
      //         : "",
      //       link: linkedIssue
      //         ? `/issue/${issueSlug(linkedIssue.name, linkedIssue.id)}`
      //         : undefined,
      //       manualHide: linkedIssue != undefined,
      //     });
      //   });
    };

    validateGrantee();

    return {
      state,
      database,
      allowConfigure,
      allowUpdateDataSourceMember,
      allowUpdateIssueId,
      allowCreate,
      issueLink,
      updateState,
      cancel,
      doGrant,
    };
  },
});
</script>

<style scoped>
/*  Removed the ticker in the number field  */
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Firefox */
input[type="number"] {
  -moz-appearance: textfield;
}
</style>
