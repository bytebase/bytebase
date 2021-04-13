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
    <option disabled :selected="UNKNOWN_ID == state.selectedId">
      Select project
    </option>
    <!-- It may happen the selected id might not be in the project list.
         e.g. the selected project is deleted after the selection and we
         are unable to cleanup properly. In such case, the seleted project id
         is orphaned and we just display the id.  -->
    <option
      v-if="selectedIdNotInList"
      :value="state.selectedId"
      :selected="true"
    >
      {{ state.selectedId }}
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
import { Principal, Project, UNKNOWN_ID } from "../types";
import { isDBAOrOwner } from "../utils";

interface LocalState {
  selectedId: string;
}

export default {
  name: "ProjectSelect",
  emits: ["select-project-id"],
  components: {},
  props: {
    selectedId: {
      default: UNKNOWN_ID,
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

    const currentUser: ComputedRef<Principal> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const prepareProjectList = () => {
      // Fetches the entire project list if DBA or above.
      if (isDBAOrOwner(currentUser.value.role)) {
        store.dispatch("project/fetchProjectList").catch((error) => {
          console.log(error);
        });
      } else {
        store
          .dispatch("project/fetchProjectListByUser", currentUser.value.id)
          .catch((error) => {
            console.log(error);
          });
      }
    };

    watchEffect(prepareProjectList);

    const projectList = computed((): Project[] => {
      return isDBAOrOwner(currentUser.value.role)
        ? store.getters["project/projectList"]()
        : store.getters["project/projectListByUser"](currentUser.value.id);
    });

    const selectedIdNotInList = computed((): boolean => {
      if (props.selectedId == UNKNOWN_ID) {
        return false;
      }

      return (
        projectList.value.find((item) => {
          return item.id == props.selectedId;
        }) == null
      );
    });

    watch(
      () => props.selectedId,
      (cur, _) => {
        state.selectedId = cur;
      }
    );

    return {
      UNKNOWN_ID,
      state,
      projectList,
      selectedIdNotInList,
    };
  },
};
</script>
