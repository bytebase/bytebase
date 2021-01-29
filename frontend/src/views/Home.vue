<template>
  <div class="flex flex-col">
    <div class="px-2 py-2">
      <BBTableTabFilter
        :itemList="state.environmentFilterList"
        :selectedIndex="state.selectedFilterIndex"
        @select-index="selectFilter"
      />
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
import { watchEffect, computed, inject, reactive, PropType } from "vue";
import PipelineTable from "../components/PipelineTable.vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { Environment, Pipeline } from "../types";
import environment from "../store/modules/environment";

interface EnvironmentFilter {
  id: string;
  title: string;
}

interface LocalState {
  attentionList: Pipeline[];
  subscribeList: Pipeline[];
  closeList: Pipeline[];
  environmentFilterList: EnvironmentFilter[];
  selectedFilterIndex: number;
}

export default {
  name: "Home",
  components: {
    PipelineTable,
  },
  props: {},
  setup(props, ctx) {
    const state = reactive<LocalState>({
      attentionList: [],
      subscribeList: [],
      closeList: [],
      environmentFilterList: [],
      selectedFilterIndex: 0,
    });
    const store = useStore();
    const router = useRouter();
    const currentUser = inject<Pipeline>(UserStateSymbol);

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

    const prepareEnvironmentList = () => {
      store
        .dispatch("environment/fetchEnvironmentList")
        .then((list: Environment[]) => {
          state.environmentFilterList = [
            {
              id: "",
              title: "All",
            },
            ...list
              .map((environment) => {
                return {
                  id: environment.id,
                  title: environment.attributes.name,
                };
              })
              // Usually env is ordered by ascending importantance, thus we rervese the order to put
              // more important ones first.
              .reverse(),
          ];
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const selectFilter = (index: number) => {
      state.selectedFilterIndex = index;
    };

    const filteredList = (list: Pipeline[]) => {
      if (state.selectedFilterIndex == 0) {
        // Select "All"
        return list;
      }
      return list.filter((pipeline) => {
        return (
          pipeline.attributes.currentStageId ==
          state.environmentFilterList[state.selectedFilterIndex].id
        );
      });
    };

    watchEffect(preparePipelineList);

    watchEffect(prepareEnvironmentList);

    return {
      state,
      filteredList,
      selectFilter,
    };
  },
};
</script>
