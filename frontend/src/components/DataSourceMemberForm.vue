<template>
  <form class="mx-4 space-y-6 divide-y divide-block-border">
    <div class="">
      <!-- Instance Name -->
      <div class="grid gap-y-6 gap-x-4 grid-cols-4">
        <div class="col-span-2 col-start-2 w-64">
          <label for="environment" class="textlabel">
            Environment <span style="color: red">*</span>
          </label>
          <!-- eslint-disable vue/attribute-hyphenation -->
          <EnvironmentSelect
            id="environment"
            class="mt-1 w-full"
            name="environment"
            :disabled="!allowConfigure"
            :selectedID="state.environmentID"
            @select-environment-id="
              (environmentID) => {
                updateState('environmentID', environmentID);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
          <label for="instance" class="textlabel">
            Instance <span class="text-red-600">*</span>
          </label>
          <!-- eslint-disable vue/attribute-hyphenation -->
          <InstanceSelect
            id="instance"
            class="mt-1 w-full"
            name="instance"
            :disabled="!allowConfigure"
            :selectedID="state.instanceID"
            @select-instance-id="
              (instanceID) => {
                updateState('instanceID', instanceID);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
          <label for="database" class="textlabel">
            Database <span class="text-red-600">*</span>
          </label>
          <!-- eslint-disable vue/attribute-hyphenation -->
          <DatabaseSelect
            id="database"
            class="mt-1 w-full"
            name="database"
            :disabled="!allowConfigure"
            :mode="'INSTANCE'"
            :instanceID="state.instanceID"
            :selectedID="state.databaseID"
            @select-database-id="
              (databaseID) => {
                updateState('databaseID', databaseID);
              }
            "
          />
        </div>

        <div class="col-span-2 col-start-2 w-64">
          <label for="datasource" class="textlabel">
            Data Source <span class="text-red-600">*</span>
          </label>
          <!-- eslint-disable vue/attribute-hyphenation -->
          <DataSourceSelect
            id="datasource"
            class="mt-1 w-full"
            name="datasource"
            :disabled="!allowConfigure"
            :database="database"
            :selectedID="state.dataSourceID"
            @select-data-source-id="
              (dataSourceID) => {
                updateState('dataSourceID', dataSourceID);
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
          <!-- eslint-disable vue/attribute-hyphenation -->
          <MemberSelect
            id="user"
            class="mt-1"
            name="user"
            :allowed-role-list="['DEVELOPER']"
            :disabled="!allowUpdateDataSourceMember"
            :selectedID="state.granteeID"
            @select-principal-id="
              (principalID) => {
                updateState('granteeID', principalID);
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
                id="issue"
                class="textfield w-full pl-12"
                name="issue"
                type="number"
                placeholder="Your issue id (e.g. 1234)"
                :disabled="!allowUpdateIssueID"
                :value="state.issueID"
                @input="updateState('issueID', $event.target.value)"
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
import MemberSelect from "../components/MemberSelect.vue";
import {
  DatabaseID,
  DataSource,
  DataSourceID,
  DataSourceMember,
  DataSourceMemberCreate,
  EnvironmentID,
  InstanceID,
  PrincipalID,
  Issue,
  IssueID,
} from "../types";
import { issueSlug } from "../utils";

interface LocalState {
  environmentID?: EnvironmentID;
  instanceID?: InstanceID;
  databaseID?: DatabaseID;
  dataSourceID?: DataSourceID;
  granteeID?: PrincipalID;
  granteeError: string;
  issueID?: IssueID;
}

export default {
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
    principalID: {
      type: Number,
    },
    issueID: {
      type: Number,
    },
  },
  emits: ["submit", "cancel"],
  setup(props, { emit }) {
    const store = useStore();

    const state = reactive<LocalState>({
      environmentID: props.dataSource
        ? props.dataSource.instance.environment.id
        : undefined,
      instanceID: props.dataSource ? props.dataSource.instance.id : undefined,
      databaseID: props.dataSource ? props.dataSource.database.id : undefined,
      dataSourceID: props.dataSource ? props.dataSource.id : undefined,
      granteeID: props.principalID,
      granteeError: "",
      issueID: props.issueID,
    });

    const database = computed(() => {
      return state.databaseID
        ? store.getters["database/databaseByID"](state.databaseID)
        : undefined;
    });

    const dataSource = computed(() => {
      if (props.dataSource) {
        return props.dataSource;
      }

      if (state.dataSourceID) {
        if (state.databaseID) {
          const database = store.getters["database/databaseByID"](
            state.databaseID
          );
          if (database) {
            return database.dataSourceList.find(
              (item: DataSource) => item.id == state.dataSourceID
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
      return !props.principalID && state.dataSourceID;
    });

    const allowUpdateIssueID = computed(() => {
      return !props.issueID;
    });

    const allowCreate = computed(() => {
      return state.dataSourceID && state.granteeID && !state.granteeError;
    });

    const issueLink = computed((): string => {
      if (state.issueID) {
        // We intentionally not to validate whether the issueID is legit, we will do the validation
        // when actually trying to create the database.
        return `/issue/${state.issueID}`;
      }
      return "";
    });

    const validateGrantee = () => {
      if (state.granteeID) {
        const member = dataSource.value.memberList.find(
          (item: DataSourceMember) => item.principal.id == state.granteeID
        );
        if (member) {
          state.granteeError = `${member.principal.name} already exists`;
        } else {
          state.granteeError = "";
        }
      }
    };

    const updateState = (field: string, value: number) => {
      if (field == "environmentID") {
        state.environmentID = value;
      } else if (field == "instanceID") {
        state.instanceID = value;
      } else if (field == "databaseID") {
        state.databaseID = value;
      } else if (field == "dataSourceID") {
        state.dataSourceID = value;
      } else if (field == "granteeID") {
        state.granteeID = value;
        validateGrantee();
      } else if (field == "issueID") {
        state.issueID = value;
      }
    };

    const cancel = () => {
      emit("cancel");
    };

    const doGrant = async () => {
      // If issueID id provided, we check its existence first.
      // We only set the issueID if it's valid.
      let linkedIssue: Issue | undefined = undefined;
      if (state.issueID) {
        try {
          linkedIssue = await store.dispatch(
            "issue/fetchIssueByID",
            state.issueID
          );
        } catch (err) {
          console.warn(`Unable to fetch linked issue id ${state.issueID}`, err);
        }
      }

      const newDataSourceMember: DataSourceMemberCreate = {
        principalID: state.granteeID!,
        issueID: linkedIssue?.id,
      };
      store
        .dispatch("dataSource/createDataSourceMember", {
          dataSourceID: state.dataSourceID,
          databaseID: state.databaseID,
          newDataSourceMember,
        })
        .then((dataSource: DataSource) => {
          emit("submit", dataSource);

          const addedMember = dataSource.memberList.find(
            (item: DataSourceMember) => {
              return item.principal.id == state.granteeID;
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
        });
    };

    validateGrantee();

    return {
      state,
      database,
      allowConfigure,
      allowUpdateDataSourceMember,
      allowUpdateIssueID,
      allowCreate,
      issueLink,
      updateState,
      cancel,
      doGrant,
    };
  },
};
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
