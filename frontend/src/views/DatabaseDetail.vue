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
                <h1 class="text-xl font-bold leading-6 text-main truncate">
                  {{ database.name }}
                </h1>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="mt-6">
        <div class="max-w-6xl mx-auto px-6 space-y-6">
          <!-- Description list -->
          <dl class="grid grid-cols-1 gap-x-4 gap-y-4 sm:grid-cols-2">
            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">
                Environment
              </dt>
              <dd class="mt-1 text-sm text-main">
                <div class="mt-2.5 mb-3">
                  {{ database.instance.environment.name }}
                </div>
              </dd>
            </div>

            <div class="sm:col-span-1">
              <dt class="text-sm font-medium text-control-light">Instance</dt>
              <dd class="mt-1 text-sm text-main">
                <div class="mt-2.5 mb-3">
                  {{ database.instance.name }}
                </div>
              </dd>
            </div>
          </dl>
        </div>
      </div>
    </main>
  </div>
</template>

<script lang="ts">
import { computed, nextTick, reactive, ref } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import cloneDeep from "lodash-es/cloneDeep";
import isEqual from "lodash-es/isEqual";
import { toClipboard } from "@soerenmartius/vue3-clipboard";
import DataSourceMemberTable from "../components/DataSourceMemberTable.vue";
import { idFromSlug } from "../utils";
import { ALL_DATABASE_NAME, DataSource } from "../types";

interface LocalState {
  editing: boolean;
  showPassword: boolean;
  editingDataSource?: DataSource;
}

export default {
  name: "DatabaseDetail",
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
    databaseSlug: {
      required: true,
      type: String,
    },
  },
  components: {},
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      editing: false,
      showPassword: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const database = computed(() => {
      return store.getters["database/databaseById"](
        idFromSlug(props.databaseSlug),
        idFromSlug(props.instanceSlug)
      );
    });

    const instance = computed(() => {
      return store.getters["instance/instanceById"](
        idFromSlug(props.instanceSlug)
      );
    });

    return {
      state,
      database,
      instance,
    };
  },
};
</script>
