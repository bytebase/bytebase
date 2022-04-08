<template>
  <div class="flex-1 overflow-auto focus:outline-none" tabindex="0">
    <main class="flex-1 relative pb-8 overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="px-4 pb-4 border-b border-block-border md:flex md:items-center md:justify-between"
      >
        <div class="flex-1 min-w-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center">
                <input
                  v-if="state.editing"
                  id="name"
                  ref="editNameTextField"
                  v-model="state.editingDataSource.name"
                  required
                  name="name"
                  type="text"
                  class="textfield my-0.5 w-full"
                />
                <!-- Padding value is to prevent flickering when switching between edit/non-edit mode -->
                <h1
                  v-else
                  class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate"
                >
                  {{ dataSource.name }}
                </h1>
              </div>
              <dl
                class="flex flex-col space-y-1 sm:space-y-0 sm:flex-row sm:flex-wrap"
              >
                <dt class="sr-only">{{ $t("common.environment") }}</dt>
                <dd class="flex items-center text-sm sm:mr-4">
                  <span class="textlabel"
                    >{{ $t("common.environment") }}&nbsp;-&nbsp;</span
                  >
                  <router-link
                    :to="`/environment/${environmentSlug(
                      dataSource.instance.environment
                    )}`"
                    class="normal-link"
                  >
                    {{ environmentName(dataSource.instance.environment) }}
                  </router-link>
                </dd>
                <template v-if="isCurrentUserDBAOrOwner">
                  <dt class="sr-only">{{ $t("common.instance") }}</dt>
                  <dd class="flex items-center text-sm sm:mr-4">
                    <span class="textlabel"
                      >{{ $t("common.instance") }}&nbsp;-&nbsp;</span
                    >
                    <router-link
                      :to="`/instance/${instanceSlug(dataSource.instance)}`"
                      class="normal-link"
                    >
                      {{ instanceName(dataSource.instance) }}
                    </router-link>
                  </dd>
                </template>
                <dt class="sr-only">{{ $t("common.database") }}</dt>
                <dd class="flex items-center text-sm sm:mr-4">
                  <span class="textlabel"
                    >{{ $t("common.database") }}&nbsp;-&nbsp;</span
                  >
                  <router-link :to="`/db/${databaseSlug}`" class="normal-link">
                    {{ dataSource.database.name }}
                  </router-link>
                </dd>
                <dt class="sr-only">{{ $t("common.roletype") }}</dt>
                <dd
                  v-data-source-type
                  class="flex items-center text-sm text-control font-medium sm:mr-4"
                >
                  {{ dataSource.type }}
                </dd>
              </dl>
            </div>
          </div>
        </div>
        <div v-if="allowEdit" class="mt-6 flex space-x-3 md:mt-0 md:ml-4">
          <template v-if="state.editing">
            <button
              type="button"
              class="btn-normal"
              @click.prevent="cancelEdit"
            >
              {{ $t("common.cancel") }}
            </button>
            <button
              type="button"
              class="btn-normal"
              :disabled="!allowSave"
              @click.prevent="saveEdit"
            >
              <!-- Heroicon name: solid/save -->
              <heroicons-solid:save
                class="-ml-1 mr-2 h-5 w-5 text-control-light"
              />
              <span>{{ $t("common.save") }}</span>
            </button>
          </template>
          <template v-else>
            <button
              type="button"
              class="btn-normal"
              @click.prevent="editDataSource"
            >
              <!-- Heroicon name: solid/pencil -->
              <heroicons-solid:pencil
                class="-ml-1 mr-2 h-5 w-5 text-control-light"
              />
              <span>{{ $t("common.edit") }}</span>
            </button>
          </template>
        </div>
      </div>

      <div class="mt-6">
        <div
          class="max-w-6xl mx-auto px-6 space-y-6 divide-y divide-block-border"
        >
          <!-- Description list -->
          <DataSourceConnectionPanel
            :editing="state.editing"
            :data-source="state.editing ? state.editingDataSource : dataSource"
          />

          <!-- Guard against dataSource.id != '-1', this could happen when we delete the data source -->
          <DataSourceMemberTable
            v-if="dataSource.id != '-1'"
            class="pt-6"
            :allow-edit="allowEdit"
            :data-source="dataSource"
          />

          <!-- Hide deleting data source list for now, as we don't allow deleting data source after creating the database. -->
          <div v-if="false" class="pt-4 flex justify-start">
            <BBButtonConfirm
              v-if="allowEdit"
              :button-text="'Delete this entire data source'"
              :require-confirm="true"
              :confirm-title="`Are you sure to delete '${dataSource.name}'?`"
              :confirm-description="'All existing users using this data source to connect the database will fail. You cannot undo this action.'"
              @confirm="doDelete"
            />
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, nextTick, reactive, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import DataSourceConnectionPanel from "../components/DataSourceConnectionPanel.vue";
import DataSourceMemberTable from "../components/DataSourceMemberTable.vue";
import { idFromSlug, isDBAOrOwner } from "../utils";
import { DataSource, DataSourcePatch, Principal } from "../types";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";

interface LocalState {
  editing: boolean;
  showPassword: boolean;
  editingDataSource?: DataSource;
}

export default defineComponent({
  name: "DataSourceDetail",
  components: { DataSourceConnectionPanel, DataSourceMemberTable },
  props: {
    databaseSlug: {
      required: true,
      type: String,
    },
    dataSourceSlug: {
      required: true,
      type: String,
    },
  },
  setup(props) {
    const editNameTextField = ref();

    const store = useStore();
    const router = useRouter();

    const { t } = useI18n();

    const dataSourceId = idFromSlug(props.dataSourceSlug);

    const state = reactive<LocalState>({
      editing: false,
      showPassword: false,
    });

    const currentUser = computed(
      (): Principal => store.getters["auth/currentUser"]()
    );

    const dataSource = computed((): DataSource => {
      return store.getters["dataSource/dataSourceById"](dataSourceId);
    });

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowEdit = computed(() => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowSave = computed(() => {
      return (
        state.editingDataSource!.name &&
        !isEqual(dataSource.value, state.editingDataSource)
      );
    });

    const editDataSource = () => {
      state.editingDataSource = cloneDeep(dataSource.value);
      state.editing = true;

      nextTick(() => editNameTextField.value.focus());
    };

    const cancelEdit = () => {
      state.editingDataSource = undefined;
      state.editing = false;
    };

    const saveEdit = () => {
      const dataSourcePatch: DataSourcePatch = {
        name: state.editingDataSource?.name,
        username: state.editingDataSource?.username,
        password: state.editingDataSource?.password,
      };
      store
        .dispatch("dataSource/patchDataSource", {
          databaseId: dataSource.value.databaseId,
          dataSourceid: dataSource.value.id,
          dataSource: dataSourcePatch,
        })
        .then(() => {
          state.editingDataSource = undefined;
          state.editing = false;
        });
    };

    const doDelete = () => {
      const name = dataSource.value.name;
      store
        .dispatch("dataSource/deleteDataSourceById", {
          databaseId: dataSource.value.databaseId,
          dataSourceId,
        })
        .then(() => {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("datasource.successfully-deleted-data-source-name", [
              name,
            ]),
          });
          router.push(`/db/${props.dataSourceSlug}`);
        });
    };

    return {
      editNameTextField,
      state,
      dataSource,
      isCurrentUserDBAOrOwner,
      allowEdit,
      allowSave,
      editDataSource,
      cancelEdit,
      saveEdit,
      doDelete,
    };
  },
});
</script>
