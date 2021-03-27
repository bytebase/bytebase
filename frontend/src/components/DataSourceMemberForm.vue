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
  <form class="space-y-6 divide-y divide-block-border">
    <div class="">
      <!-- Instance Name -->
      <div class="grid gap-y-6 gap-x-4 grid-cols-1">
        <div class="col-span-1 col-start-1">
          <label for="environment" class="textlabel">
            Environment <span style="color: red">*</span>
          </label>
          <EnvironmentSelect
            class="mt-1"
            id="environment"
            name="environment"
            :disabled="true"
            :selectedId="state.environmentId"
          />
        </div>

        <div class="col-span-1 col-start-1">
          <label for="instance" class="textlabel">
            Instance <span class="text-red-600">*</span>
          </label>
          <InstanceSelect
            class="mt-1"
            id="instance"
            name="instance"
            :disabled="true"
            :selectedId="dataSource.instanceId"
          />
        </div>

        <div class="col-span-1 col-start-1">
          <label for="database" class="textlabel">
            Database <span class="text-red-600">*</span>
          </label>
          <DatabaseSelect
            class="mt-1"
            id="database"
            name="database"
            :disabled="true"
            :mode="'INSTANCE'"
            :instanceId="dataSource.instanceId"
            :selectedId="dataSource.databaseId"
          />
        </div>

        <div class="col-span-1 col-start-1">
          <label for="datasource" class="textlabel">
            Data Source <span class="text-red-600">*</span>
          </label>
          <DataSourceSelect
            class="mt-1"
            id="datasource"
            name="datasource"
            :disabled="true"
            :instanceId="dataSource.instanceId"
            :selectedId="dataSource.id"
          />
        </div>

        <div class="col-span-1 col-start-1">
          <label for="user" class="textlabel">
            User <span class="text-red-600">*</span>
            <span class="ml-2 text-error text-xs">
              {{ state.principalError }}
            </span>
          </label>
          <PrincipalSelect
            class="mt-1"
            id="user"
            name="user"
            :selectedId="principalId"
            @select-principal-id="
              (principalId) => {
                updateDataSourceMember('principalId', principalId);
              }
            "
          />
        </div>

        <div class="col-span-1 col-start-1">
          <label for="task" class="textlabel"> Task </label>
          <div class="mt-1 relative rounded-md shadow-sm">
            <div
              class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"
            >
              <span class="text-accent font-semibold sm:text-sm">task/</span>
            </div>
            <input
              class="textfield w-full pl-12"
              id="task"
              name="task"
              type="number"
              placeholder="Your task id (e.g. 1234)"
              :value="state.taskId"
              @input="updateDataSourceMember('taskId', $event.target.value)"
            />
          </div>
        </div>
      </div>
    </div>
    <!-- Action Button Group -->
    <div class="pt-4">
      <!-- Create button group -->
      <div class="flex justify-end">
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
    </div>
  </form>
</template>

<script lang="ts">
import { computed, reactive, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import EnvironmentSelect from "../components/EnvironmentSelect.vue";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import DataSourceSelect from "../components/DataSourceSelect.vue";
import InstanceSelect from "../components/InstanceSelect.vue";
import PrincipalSelect from "../components/PrincipalSelect.vue";
import { instanceSlug } from "../utils";
import {
  DataSource,
  DataSourceMember,
  DataSourceMemberNew,
  EnvironmentId,
  PrincipalId,
  TaskId,
} from "../types";

interface LocalState {
  environmentId?: EnvironmentId;
  principalId?: PrincipalId;
  principalError: string;
  taskId?: TaskId;
}

export default {
  name: "DataSourceMemberNewForm",
  emits: ["dismiss"],
  props: {
    dataSource: {
      required: true,
      type: Object as PropType<DataSource>,
    },
    principalId: {
      type: String,
    },
    taskId: {
      type: String,
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
    const router = useRouter();

    const databaseName = computed(() => {
      const database = store.getters["database/databaseById"](
        props.dataSource.databaseId,
        { instanceId: props.dataSource.instanceId }
      );
      return database.name;
    });

    const instance = computed(() => {
      return store.getters["instance/instanceById"](
        props.dataSource.instanceId
      );
    });

    const state = reactive<LocalState>({
      environmentId: instance.value.environmentId,
      principalId: props.principalId,
      principalError: "",
      taskId: props.taskId,
    });

    const allowCreate = computed(() => {
      return state.principalId && !state.principalError;
    });

    const updateDataSourceMember = (field: string, value: string) => {
      if (field == "principalId") {
        state.principalId = value;
        const member = props.dataSource.memberList.find(
          (item: DataSourceMember) => item.principal.id == value
        );
        if (member) {
          state.principalError = `${member.principal.name} already exists`;
        } else {
          state.principalError = "";
        }
      } else if (field == "taskId") {
        state.taskId = value;
      }
    };

    const cancel = () => {
      emit("dismiss");
    };

    const doGrant = () => {
      store
        .dispatch("dataSource/createDataSourceMember", {
          instanceId: props.dataSource.instanceId,
          dataSourceId: props.dataSource.id,
          newDataSourceMember: {
            principalId: state.principalId,
            taskId: state.taskId,
          },
        })
        .then((dataSourceMember: DataSourceMember) => {
          emit("dismiss");
          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: `Successfully granted '${props.dataSource.name}' to '${dataSourceMember.principal.name}'.`,
          });
        })
        .catch((error) => {
          console.error(error);
        });
    };

    return {
      state,
      allowCreate,
      databaseName,
      instance,
      updateDataSourceMember,
      cancel,
      doGrant,
    };
  },
};
</script>
