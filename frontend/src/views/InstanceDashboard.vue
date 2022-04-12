<template>
  <div class="flex flex-col">
    <FeatureAttention
      v-if="remainingInstanceCount <= 3"
      custom-class="m-5"
      feature="bb.feature.instance-count"
      :description="instanceCountAttention"
    />
    <div class="px-5 py-2 flex justify-between items-center">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('instance.search-instance-name')"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <InstanceTable :instance-list="filteredList(instanceList)" />
  </div>

  <BBAlert
    v-if="state.showGuide"
    :style="'INFO'"
    :ok-text="$t('common.do-not-show-again')"
    :cancel-text="$t('common.dismiss')"
    :title="$t('instance.how-to-setup-instance')"
    :description="$t('instance.how-to-setup-instance-description')"
    @ok="
      () => {
        doDismissGuide();
      }
    "
    @cancel="state.showGuide = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, onMounted, reactive, ref, defineComponent } from "vue";
import { useRouter } from "vue-router";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import InstanceTable from "../components/InstanceTable.vue";
import { Environment, Instance } from "../types";
import { cloneDeep } from "lodash-es";
import { sortInstanceList } from "../utils";
import { useI18n } from "vue-i18n";
import {
  useUIStateStore,
  useSubscriptionStore,
  useEnvironmentStore,
  useEnvironmentList,
  useInstanceList,
  useInstanceStore,
} from "@/store";

interface LocalState {
  searchText: string;
  selectedEnvironment?: Environment;
  showGuide: boolean;
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

    const environmentList = useEnvironmentList(["NORMAL"]);

    const state = reactive<LocalState>({
      searchText: "",
      selectedEnvironment: router.currentRoute.value.query.environment
        ? useEnvironmentStore().getEnvironmentById(
            parseInt(router.currentRoute.value.query.environment as string, 10)
          )
        : undefined,
      showGuide: false,
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!uiStateStore.getIntroStateByKey("guide.instance")) {
        setTimeout(() => {
          state.showGuide = true;
          uiStateStore.saveIntroStateByKey({
            key: "instance.visit",
            newState: true,
          });
        }, 1000);
      }
    });

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
      if (environment) {
        router.replace({
          name: "workspace.instance",
          query: { environment: environment.id },
        });
      } else {
        router.replace({ name: "workspace.instance" });
      }
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const doDismissGuide = () => {
      uiStateStore.saveIntroStateByKey({
        key: "guide.instance",
        newState: true,
      });
      state.showGuide = false;
    };

    const rawInstanceList = useInstanceList();

    const instanceList = computed(() => {
      return sortInstanceList(
        cloneDeep(rawInstanceList.value),
        environmentList.value
      );
    });

    const filteredList = (list: Instance[]) => {
      if (!state.selectedEnvironment && !state.searchText) {
        // Select "All"
        return list;
      }
      return list.filter((instance) => {
        return (
          (!state.selectedEnvironment ||
            instance.environment.id == state.selectedEnvironment.id) &&
          (!state.searchText ||
            instance.name
              .toLowerCase()
              .includes(state.searchText.toLowerCase()))
        );
      });
    };

    const instanceQuota = computed((): number => {
      const { subscription } = subscriptionStore;
      return subscription?.instanceCount ?? 5;
    });

    const remainingInstanceCount = computed((): number => {
      const instanceList: Instance[] = instanceStore.getInstanceList([
        "NORMAL",
      ]);
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
      searchField,
      state,
      instanceList,
      filteredList,
      selectEnvironment,
      changeSearchText,
      doDismissGuide,
      remainingInstanceCount,
      instanceCountAttention,
    };
  },
});
</script>
