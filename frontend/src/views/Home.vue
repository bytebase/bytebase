<template>
  <div class="flex flex-col">
    <div class="p-4">
      <QuickActionPanel />
    </div>
    <div class="px-2 py-1 border-t">
      <EnvironmentTabFilter @select-environment="selectEnvironment" />
    </div>
    <PipelineTable
      :pipelineSectionList="[
        {
          title: 'Attention',
          list: filteredList(state.attentionList).sort((a, b) => {
            return b.attributes.lastUpdatedTs - a.attributes.lastUpdatedTs;
          }),
        },
        {
          title: 'Subscribed',
          list: filteredList(state.subscribeList).sort((a, b) => {
            return b.attributes.lastUpdatedTs - a.attributes.lastUpdatedTs;
          }),
        },
        {
          title: 'Recently Closed',
          list: filteredList(state.closeList).sort((a, b) => {
            return b.attributes.lastUpdatedTs - a.attributes.lastUpdatedTs;
          }),
        },
      ]"
    />
  </div>
</template>

<script lang="ts">
import { watchEffect, inject, reactive } from "vue";
import EnvironmentTabFilter from "../components/EnvironmentTabFilter.vue";
import PipelineTable from "../components/PipelineTable.vue";
import QuickActionPanel from "../components/QuickActionPanel.vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { User, Environment, Pipeline } from "../types";

interface LocalState {
  attentionList: Pipeline[];
  subscribeList: Pipeline[];
  closeList: Pipeline[];
  selectedEnvironment?: Environment;
}

export default {
  name: "Home",
  components: {
    EnvironmentTabFilter,
    PipelineTable,
    QuickActionPanel,
  },
  props: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({
      attentionList: [],
      subscribeList: [],
      closeList: [],
    });
    const store = useStore();
    const router = useRouter();
    const currentUser = inject<User>(UserStateSymbol);

    const preparePipelineList = () => {
      store
        .dispatch("pipeline/fetchPipelineListForUser", currentUser!.id)
        .then((pipelineList: Pipeline[]) => {
          state.attentionList = [];
          state.subscribeList = [];
          state.closeList = [];
          for (const pipeline of pipelineList) {
            if (
              pipeline.attributes.assignee.id === currentUser!.id &&
              (pipeline.attributes.status === "PENDING" ||
                pipeline.attributes.status === "RUNNING" ||
                pipeline.attributes.status === "FAILED")
            ) {
              state.attentionList.push(pipeline);
            } else if (
              pipeline.attributes.subscriberIdList.includes(currentUser!.id) &&
              (pipeline.attributes.status === "PENDING" ||
                pipeline.attributes.status === "RUNNING" ||
                pipeline.attributes.status === "FAILED")
            ) {
              state.subscribeList.push(pipeline);
            } else if (
              pipeline.attributes.creator.id === currentUser!.id &&
              (pipeline.attributes.status === "DONE" ||
                pipeline.attributes.status === "CANCELED")
            ) {
              state.closeList.push(pipeline);
            }
          }
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const selectEnvironment = (environment: Environment) => {
      state.selectedEnvironment = environment;
    };

    const filteredList = (list: Pipeline[]) => {
      if (!state.selectedEnvironment) {
        // Select "All"
        return list;
      }
      return list.filter((pipeline) => {
        return (
          pipeline.attributes.currentStageId == state.selectedEnvironment!.id
        );
      });
    };

    watchEffect(preparePipelineList);

    return {
      state,
      filteredList,
      selectEnvironment,
    };
  },
};
</script>
