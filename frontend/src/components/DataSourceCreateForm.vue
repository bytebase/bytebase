<template>
  <form
    class="space-y-6 divide-y divide-block-border"
    @submit.prevent="$emit('create', state.dataSource)"
  >
    <div class="space-y-4">
      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3">
        <div class="sm:col-span-2">
          <label for="name" class="textlabel">
            {{ $t("common.name") }} <span class="text-red-600">*</span>
          </label>
          <input
            id="name"
            required
            name="name"
            type="text"
            class="textfield mt-1 w-full"
            :value="state.dataSource.name"
            @input="updateDataSource('name', $event.target.value)"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3 items-center">
        <div class="sm:col-span-2">
          <label for="database" class="textlabel">
            {{ $t("common.database") }} <span class="text-red-600">*</span>
          </label>
          <div v-if="database" class="flex flex-row justify-between space-x-2">
            {{ database.name }}
            <BBSwitch
              :label="$t('common.read-only')"
              :value="state.dataSource.type == 'RO'"
              @toggle="
                (on) => {
                  updateDataSource('type', on ? 'RO' : 'RW');
                }
              "
            />
          </div>
          <div v-else class="flex flex-row justify-between space-x-2">
            <!-- eslint-disable vue/attribute-hyphenation -->
            <DatabaseSelect
              class="mt-1 w-full"
              :selectedId="state.dataSource.databaseId"
              :mode="'INSTANCE'"
              :instanceId="instanceId"
              @select-database-id="
                (databaseId) => {
                  updateDataSource('databaseId', databaseId);
                }
              "
            />
            <BBSwitch
              :label="$t('common.read-only')"
              :value="state.dataSource.type == 'RO'"
              @toggle="
                (on) => {
                  updateDataSource('type', on ? 'RO' : 'RW');
                }
              "
            />
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3">
        <div class="sm:col-span-2">
          <label for="username" class="textlabel">
            {{ $t("common.username") }}
          </label>
          <input
            id="username"
            name="username"
            type="text"
            class="textfield mt-1 w-full"
            :value="state.dataSource.username"
            @input="updateDataSource('username', $event.target.value)"
          />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-3">
        <div class="sm:col-span-2">
          <div class="flex flex-row items-center space-x-1">
            <label for="username" class="textlabel">
              {{ $t("common.password") }}
            </label>
            <button
              class="btn-icon"
              @click.prevent="state.showPassword = !state.showPassword"
            >
              <heroicons-outline:eye-off
                v-if="state.showPassword"
                class="w-6 h-6"
              />
              <heroicons-outline:eye v-else class="w-6 h-6" />
            </button>
          </div>
          <input
            id="password"
            name="password"
            autocomplete="off"
            :type="state.showPassword ? 'text' : 'password'"
            class="textfield mt-1 w-full"
            :value="state.dataSource.password"
            @input="updateDataSource('password', $event.target.value)"
          />
        </div>
      </div>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="$emit('cancel')"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        type="submit"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
      >
        {{ $t("common.create") }}
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, defineComponent, PropType, reactive } from "vue";
import DatabaseSelect from "./DatabaseSelect.vue";
import { DataSourceCreate, Database, UNKNOWN_ID } from "../types";
import { useI18n } from "vue-i18n";

interface LocalState {
  dataSource: DataSourceCreate;
  showPassword: boolean;
}

export default defineComponent({
  name: "DataSourceCreateForm",
  components: { DatabaseSelect },
  props: {
    instanceId: {
      required: true,
      type: Number,
    },
    // If database is specified, then we just show that database instead of the database select
    database: {
      type: Object as PropType<Database>,
    },
  },
  emits: ["create", "cancel"],
  setup(props) {
    const { t } = useI18n();
    const state = reactive<LocalState>({
      dataSource: {
        name: t("datasource.new-data-source"),
        type: "RO",
        databaseId: props.database ? props.database.id : UNKNOWN_ID,
        instanceId: props.instanceId,
      },
      showPassword: false,
    });

    const allowCreate = computed(() => {
      return state.dataSource.name;
    });

    const updateDataSource = (field: string, value: string) => {
      (state.dataSource as any)[field] = value;
    };

    return {
      state,
      allowCreate,
      updateDataSource,
    };
  },
});
</script>
