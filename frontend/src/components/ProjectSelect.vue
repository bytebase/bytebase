<template>
  <select
    class="btn-select w-full disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        $emit('select-project-id', e.target.value);
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      Select project
    </option>
    <template v-for="(project, index) in projectList" :key="index">
      <option
        v-if="project.rowStatus == 'NORMAL'"
        :value="project.id"
        :selected="project.id == state.selectedId"
      >
        {{ project.name }}
      </option>
      <option
        v-else-if="project.id == state.selectedId"
        :value="project.id"
        :selected="true"
      >
        {{ projectName(project) }}
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { computed, ComputedRef, reactive, watch, watchEffect } from "vue";
import { useStore } from "vuex";
import { Project, User } from "../types";

interface LocalState {
  selectedId?: string;
}

export default {
  name: "ProjectSelect",
  emits: ["select-project-id"],
  components: {},
  props: {
    selectedId: {
      type: String,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const currentUser: ComputedRef<User> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const prepareProjectList = () => {
      store
        .dispatch("project/fetchProjectListByUser", currentUser.value.id)
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareProjectList);

    const projectList = computed(() => {
      return store.getters["project/projectListByUser"](currentUser.value.id);
    });

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedId &&
        !projectList.value.find((item: Project) => item.id == state.selectedId)
      ) {
        state.selectedId = undefined;
        emit("select-project-id", state.selectedId);
      }
    };

    watch(
      () => projectList.value,
      () => {
        invalidateSelectionIfNeeded();
      }
    );

    watch(
      () => props.selectedId,
      (cur, _) => {
        state.selectedId = cur;
      }
    );

    return {
      state,
      projectList,
    };
  },
};
</script>
