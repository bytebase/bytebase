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

<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="">
      <!-- Instance Name -->
      <div class="grid gap-y-6 gap-x-4 grid-cols-4">
        <div class="col-span-2 col-start-2 w-64">
          <label for="environment" class="textlabel">
            Environment <span style="color: red">*</span>
          </label>
          <EnvironmentSelect
            class="mt-1 w-full"
            id="environment"
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
            Instance <span class="text-red-600">*</span>
          </label>
          <InstanceSelect
            class="mt-1 w-full"
            id="instance"
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
            Database <span class="text-red-600">*</span>
          </label>
          <DatabaseSelect
            class="mt-1 w-full"
            id="database"
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
            Data Source <span class="text-red-600">*</span>
          </label>
          <DataSourceSelect
            class="mt-1 w-full"
            id="datasource"
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
            User <span class="text-red-600">*</span>
            <span class="ml-2 text-error text-xs">
              {{ state.granteeError }}
            </span>
          </label>
          <!-- DBA and Owner always have all access, so we only need to grant to developer -->
          <PrincipalSelect
            class="mt-1"
            id="user"
            name="user"
            :allowedRoleList="['DEVELOPER']"
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
          <label for="issue" class="textlabel"> Issue </label>
          <div class="mt-1 relative rounded-md shadow-sm">
            <div
              class="
                absolute
                inset-y-0
                left-0
                pl-3
                flex
                items-center
                pointer-events-none
              "
            >
              <span class="text-accent font-semibold sm:text-sm">issue/</span>
            </div>
            <div class="flex flex-row space-x-2 items-center">
              <input
                class="textfield w-full pl-12"
                id="issue"
                name="issue"
                type="number"
                placeholder="Your issue id (e.g. 1234)"
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
                  View
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
        Cancel
      </button>
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="doGrant"
      >
        Grant
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import DataSourceSelect from "../components/DataSourceSelect.vue";
import InstanceSelect from "../components/InstanceSelect.vue";
import PrincipalSelect from "../components/PrincipalSelect.vue";
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
  UNKNOWN_ID,
} from "../types";
import { issueSlug } from "../utils";

interface LocalState {
  environmentId?: EnvironmentId;
  instanceId?: InstanceId;
  databaseId?: DatabaseId;
  dataSourceId?: DataSourceId;
  granteeId?: PrincipalId;
  granteeError: string;
  issueId?: IssueId;
}

export default {
  name: "DataSourceMemberCreateForm",
  emits: ["submit", "cancel"],
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
  components: {
    EnvironmentSelect,
    DatabaseSelect,
    DataSourceSelect,
    PrincipalSelect,
    InstanceSelect,
  },
  setup(props, { emit }) {
    const store = useStore();

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

    const database = computed(() => {
      return state.databaseId
        ? store.getters["database/databaseById"](state.databaseId)
        : undefined;
    });

    const dataSource = computed(() => {
      if (props.dataSource) {
        return props.dataSource;
      }

      if (state.dataSourceId) {
        if (state.databaseId) {
          const database = store.getters["database/databaseById"](
            state.databaseId
          );
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
          state.granteeError = `${member.principal.name} already exists`;
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
          linkedIssue = await store.dispatch(
            "issue/fetchIssueById",
            state.issueId
          );
        } catch (err) {
          console.warn(`Unable to fetch linked issue id ${state.issueId}`, err);
        }
      }

      const newDataSouceMember: DataSourceMemberCreate = {
        principalId: state.granteeId!,
        issueId: linkedIssue?.id,
      };
      store
        .dispatch("dataSource/createDataSourceMember", {
          dataSourceId: state.dataSourceId,
          databaseId: state.databaseId,
          newDataSouceMember,
        })
        .then((dataSource: DataSource) => {
          emit("submit", dataSource);

          const addedMember = dataSource.memberList.find(
            (item: DataSourceMember) => {
              return item.principal.id == state.granteeId;
            }
          );
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully granted '${dataSource.name}' to '${
              addedMember!.principal.name
            }'.`,
            description: linkedIssue
              ? `We also linked the granted database to the requested issue '${linkedIssue.name}'.`
              : "",
            link: linkedIssue
              ? `/issue/${issueSlug(linkedIssue.name, linkedIssue.id)}`
              : undefined,
            manualHide: linkedIssue != undefined,
          });
        })
        .catch((error) => {
          console.error(error);
        });
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
};
</script>
