<template>
  <div class="flex flex-col">
    <FeatureAttention
      v-if="remainingInstanceCount <= 3"
      custom-class="m-5"
      feature="bb.feature.instance-count"
      :description="instanceCountAttention"
    />
    <div class="px-5 py-2 flex justify-between items-center">
      <EnvironmentTabFilter
        :environment="state.selectedEnvironment?.uid ?? String(UNKNOWN_ID)"
        :include-all="true"
        @update:environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('instance.search-instance-name')"
        @change-text="(text: string) => changeSearchText(text)"
      />
    </div>
    <InstanceTable :instance-list="filteredList(instanceList)" />
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, ref, defineComponent } from "vue";
import { useRouter } from "vue-router";
import { cloneDeep } from "lodash-es";
import { useI18n } from "vue-i18n";

import { EnvironmentTabFilter } from "@/components/v2";
import InstanceTable from "../components/InstanceTable.vue";
import { Instance, UNKNOWN_ID } from "../types";
import { sortInstanceListByEnvironmentV1 } from "../utils";
import {
  useUIStateStore,
  useSubscriptionStore,
  useEnvironmentV1Store,
  useInstanceList,
  useInstanceStore,
  useEnvironmentV1List,
} from "@/store";
import { Environment } from "@/types/proto/v1/environment_service";

interface LocalState {
  searchText: string;
  selectedEnvironment?: Environment;
}

export default defineComponent({
  name: "InstanceDashboard",
  components: {
    EnvironmentTabFilter,
    InstanceTable,
  },
  setup() {
    const searchField = ref();

    const instanceStore = useInstanceStore();
    const subscriptionStore = useSubscriptionStore();
    const uiStateStore = useUIStateStore();
    const router = useRouter();
    const { t } = useI18n();

    const environmentList = useEnvironmentV1List(false /* !showDeleted */);

    const state = reactive<LocalState>({
      searchText: "",
      selectedEnvironment: router.currentRoute.value.query.environment
        ? useEnvironmentV1Store().getEnvironmentByUID(
            router.currentRoute.value.query.environment as string
          )
        : undefined,
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!uiStateStore.getIntroStateByKey("instance.visit")) {
        uiStateStore.saveIntroStateByKey({
          key: "instance.visit",
          newState: true,
        });
      }
    });

    const selectEnvironment = (environmentId: string | undefined) => {
      if (environmentId && environmentId !== String(UNKNOWN_ID)) {
        router.replace({
          name: "workspace.instance",
          query: { environment: environmentId },
        });
        state.selectedEnvironment =
          useEnvironmentV1Store().getEnvironmentByUID(environmentId);
      } else {
        state.selectedEnvironment = undefined;
        router.replace({ name: "workspace.instance" });
      }
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const rawInstanceList = useInstanceList();

    const instanceList = computed(() => {
      return sortInstanceListByEnvironmentV1(
        cloneDeep(rawInstanceList.value),
        environmentList.value
      );
    });

    const filteredList = (list: Instance[]) => {
      const environment = state.selectedEnvironment;
      if (environment && environment.uid !== String(UNKNOWN_ID)) {
        list = list.filter(
          (instance) => String(instance.environment.id) === environment.uid
        );
      }

      const keyword = state.searchText.trim().toLowerCase();
      if (keyword) {
        list = list.filter((instance) =>
          instance.name.toLowerCase().includes(keyword)
        );
      }
      return list;
    };

    const instanceQuota = computed((): number => {
      return subscriptionStore.instanceCount;
    });

    const remainingInstanceCount = computed((): number => {
      const instanceList = instanceStore.getInstanceList(["NORMAL"]);
      return Math.max(0, instanceQuota.value - instanceList.length);
    });

    const instanceCountAttention = computed((): string => {
      const upgrade = t(
        "subscription.features.bb-feature-instance-count.upgrade"
      );
      let status = "";
      if (remainingInstanceCount.value > 0) {
        status = t(
          "subscription.features.bb-feature-instance-count.remaining",
          {
            total: instanceQuota.value,
            count: remainingInstanceCount.value,
          }
        );
      } else {
        status = t("subscription.features.bb-feature-instance-count.runoutof", {
          total: instanceQuota.value,
        });
      }

      return `${status} ${upgrade}`;
    });

    return {
      UNKNOWN_ID,
      searchField,
      state,
      instanceList,
      filteredList,
      selectEnvironment,
      changeSearchText,
      remainingInstanceCount,
      instanceCountAttention,
    };
  },
});
</script>
