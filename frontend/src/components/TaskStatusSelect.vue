<template>
  <select
    id="status"
    name="status"
    class="pl-3 pr-10 py-2 control block w-full"
    @change="
      (e) => {
        $emit('change-status', e.target.value);
      }
    "
  >
    <option value="OPEN" :selected="selectedStatus == 'OPEN'">Open</option>
    <option value="DONE" :selected="selectedStatus == 'DONE'">Done</option>
    <option value="CANCELED" :selected="selectedStatus == 'CANCELED'">
      Canceled
    </option>
  </select>
</template>

<script lang="ts">
import { PropType } from "vue";
import cloneDeep from "lodash-es/cloneDeep";
import { useStore } from "vuex";
import { TaskStatus } from "../types";

// interface LocalState {
//   environmentList: Environment[];
//   selectedId?: string;
// }

export default {
  name: "EnvironmentSelect",
  emits: ["change-status"],
  components: {},
  props: {
    selectedStatus: {
      type: String as PropType<TaskStatus>,
    },
  },
  setup(props, { emit }) {
    // const state = reactive<LocalState>({
    //   environmentList: [],
    //   selectedId: props.selectedId,
    // });
    // const store = useStore();
    // const prepareEnvironmentList = () => {
    //   store
    //     .dispatch("environment/fetchEnvironmentList")
    //     .then((list: Environment[]) => {
    //       // Usually env is ordered by ascending importantance, thus we rervese the order to put
    //       // more important ones first.
    //       state.environmentList = cloneDeep(list).reverse();
    //       if (!state.selectedId && state.environmentList.length > 0) {
    //         state.selectedId = state.environmentList[0].id;
    //         emit("select-environment-id", state.selectedId);
    //       }
    //     })
    //     .catch((error) => {
    //       console.log(error);
    //     });
    // };
    // watchEffect(prepareEnvironmentList);
    // return {
    //   state,
    // };
  },
};
</script>
