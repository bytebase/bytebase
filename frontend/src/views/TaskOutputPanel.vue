<template>
  <h2 class="px-4 text-lg font-medium">
    Output
    <span class="text-base font-normal text-control-light">
      (these fields must be filled by the assignee before resolving the task)
    </span>
  </h2>

  <div class="my-2 mx-4 space-y-2">
    <template v-for="(field, index) in fieldList" :key="index">
      <div class="flex flex-col space-y-1">
        <div class="textlabel">
          {{ field.name }}
          <span v-if="field.required" class="text-red-600">*</span>
          <template v-if="allowEditDatabase">
            <router-link
              :to="databaseActionLink(field)"
              class="ml-2 normal-link"
            >
              {{
                task.type == "bytebase.database.create" ? "+ Create" : "+ Grant"
              }}
            </router-link>
          </template>
        </div>
        <template v-if="field.type == 'String'">
          <div class="flex flex-row">
            <input
              type="text"
              class="flex-1 min-w-0 block w-full px-3 py-2 rounded-l-md border border-r border-control-border focus:mr-0.5 focus:ring-control focus:border-control sm:text-sm disabled:bg-gray-50"
              :disabled="!allowEdit"
              :name="field.id"
              :value="fieldValue(field)"
              autocomplete="off"
              @blur="$emit('update-custom-field', field, $event.target.value)"
            />
            <!-- Disallow tabbing since the focus ring is partially covered by the text field due to overlaying -->
            <button
              tabindex="-1"
              :disabled="!fieldValue(field)"
              class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium text-control-light disabled:text-gray-300 bg-gray-50 hover:bg-gray-100 disabled:bg-gray-50 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1 disabled:cursor-not-allowed"
              @click.prevent="copyText(field)"
            >
              <svg
                class="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                ></path>
              </svg>
            </button>
            <button
              tabindex="-1"
              :disabled="!isValidLink(fieldValue(field))"
              class="-ml-px px-2 py-2 border border-gray-300 text-sm font-medium rounded-r-md text-control-light disabled:text-gray-300 bg-gray-50 hover:bg-gray-100 disabled:bg-gray-50 focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-1"
              @click.prevent="goToLink(fieldValue(field))"
            >
              <svg
                class="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                ></path>
              </svg>
            </button>
          </div>
        </template>
        <div
          v-if="field.type == 'Database'"
          class="flex flex-row items-center space-x-2"
        >
          <DatabaseSelect
            class="mt-1 w-64"
            :disabled="!allowEditDatabase"
            :mode="'ENVIRONMENT'"
            :environmentId="environmentId"
            :selectedId="fieldValue(field)"
            @select-database-id="
              (databaseId) => {
                trySaveCustomField(field, databaseId);
              }
            "
          />
          <template v-if="databaseViewLink(field)">
            <router-link
              :to="databaseViewLink(field)"
              class="ml-2 normal-link text-sm"
            >
              View
            </router-link>
          </template>
          <template v-if="task.type == 'bytebase.database.create'">
            <div
              v-if="field.resolved(taskContext)"
              class="text-sm text-success"
            >
              (Created)
            </div>
            <div v-else class="text-sm text-error">(To be created)</div>
          </template>
          <template v-else-if="task.type == 'bytebase.database.grant'">
            <div
              v-if="field.resolved(taskContext)"
              class="text-sm text-success"
            >
              (Granted)
            </div>
            <div v-else class="text-sm text-error">(To be granted)</div>
          </template>
        </div>
      </div>
    </template>
  </div>
</template>

<script lang="ts">
import { PropType, computed, reactive } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import isEqual from "lodash-es/isEqual";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import DatabaseSelect from "../components/DatabaseSelect.vue";
import { fullDatabasePath } from "../utils";
import { TaskField, TaskBuiltinFieldId, TaskContext } from "../plugins";
import { DatabaseId, DataSource, EnvironmentId, Task } from "../types";

interface LocalState {}

export default {
  name: "TaskOutputPanel",
  emits: ["update-custom-field"],
  props: {
    task: {
      required: true,
      type: Object as PropType<Task>,
    },
    fieldList: {
      required: true,
      type: Object as PropType<TaskField[]>,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  components: { DatabaseSelect },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const environmentId = computed(
      (): EnvironmentId => {
        return props.task.payload[TaskBuiltinFieldId.ENVIRONMENT];
      }
    );

    const fieldValue = (field: TaskField): string => {
      return props.task.payload[field.id];
    };

    const taskContext = computed(
      (): TaskContext => {
        return {
          store,
          currentUser: currentUser.value,
          new: false,
          task: props.task,
        };
      }
    );

    const allowEditDatabase = computed((): boolean => {
      if (!props.allowEdit) {
        return false;
      }
      return (
        props.task.type == "bytebase.database.create" ||
        props.task.type == "bytebase.database.grant"
      );
    });

    const isValidLink = (link: string): boolean => {
      return link?.trim().length > 0;
    };

    const databaseActionLink = (field: TaskField): string => {
      const queryParamList: string[] = [];

      if (props.task.type == "bytebase.database.create") {
        if (environmentId.value) {
          queryParamList.push(`environment=${environmentId.value}`);
        }

        const databaseName = props.task.payload[TaskBuiltinFieldId.DATABASE];
        queryParamList.push(`name=${databaseName}`);

        queryParamList.push(`owner=${props.task.creator.id}`);

        queryParamList.push(`task=${props.task.id}`);

        queryParamList.push(`from=${props.task.type}`);

        return "/db/new?" + queryParamList.join("&");
      }

      if (props.task.type == "bytebase.database.grant") {
        const databaseId = props.task.payload[TaskBuiltinFieldId.DATABASE];
        if (databaseId) {
          const database = store.getters["database/databaseById"](databaseId, {
            environmentId: environmentId.value,
          });
          if (database) {
            // TODO: Hard-code from DatabaseGrantTemplate
            const READ_ONLY_ID = 100;
            const readOnly = props.task.payload[READ_ONLY_ID];
            let dataSourceId;
            for (const dataSource of database.dataSourceList) {
              if (readOnly && dataSource.type == "RO") {
                dataSourceId = dataSource.id;
                break;
              } else if (!readOnly && dataSource.type == "RW") {
                dataSourceId = dataSource.id;
                break;
              }
            }

            if (dataSourceId) {
              queryParamList.push(`database=${databaseId}`);

              queryParamList.push(`datasource=${dataSourceId}`);

              queryParamList.push(`grantee=${props.task.creator.id}`);

              queryParamList.push(`task=${props.task.id}`);

              return "/db/grant?" + queryParamList.join("&");
            }
          }
        }
      }

      return "";
    };

    const databaseViewLink = (field: TaskField): string => {
      if (field.type == "Database") {
        const databaseId = fieldValue(field);
        if (databaseId) {
          const database = store.getters["database/databaseById"](databaseId, {
            environmentId: environmentId.value,
          });
          if (database) {
            return fullDatabasePath(database);
          }
        }
      }
      return "";
    };

    const copyText = (field: TaskField) => {
      toClipboard(props.task.payload[field.id]).then(() => {
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "INFO",
          title: `${field.name} copied to clipboard.`,
        });
      });
    };

    const goToLink = (link: string) => {
      const myLink = link.trim();
      const parts = myLink.split("://");
      if (parts.length > 1) {
        window.open(myLink, "_blank");
      } else {
        if (!myLink.startsWith("/")) {
          router.push("/" + myLink);
        } else {
          router.push(myLink);
        }
      }
    };

    const trySaveCustomField = (
      field: TaskField,
      value: string | DatabaseId
    ) => {
      if (!isEqual(value, fieldValue(field))) {
        emit("update-custom-field", field, value);
      }
    };

    return {
      state,
      environmentId,
      fieldValue,
      taskContext,
      allowEditDatabase,
      isValidLink,
      databaseActionLink,
      databaseViewLink,
      copyText,
      goToLink,
      trySaveCustomField,
    };
  },
};
</script>
