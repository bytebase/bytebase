<template>
  <BBOutline
    :id="'project'"
    :title="'Projects'"
    :itemList="outlineItemList"
    :allowCollapse="false"
  />
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue";
import { useStore } from "vuex";

import { Project } from "../types";
import { projectSlug } from "../utils";
import { BBOutlineItem } from "../bbkit/types";

interface LocalState {
  projectList: Project[];
}

export default {
  name: "ProjectListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      projectList: [],
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareProjectList = () => {
      store
        .dispatch("project/fetchProjectListByUser", currentUser.value.id)
        .then((projectList: Project[]) => {
          state.projectList = projectList;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    const outlineItemList = computed((): BBOutlineItem[] => {
      return state.projectList.map(
        (item: Project): BBOutlineItem => {
          return {
            id: item.id,
            name: item.name,
            link: `/project/${projectSlug(item)}`,
          };
        }
      );
    });

    watchEffect(prepareProjectList);

    return {
      outlineItemList,
    };
  },
};
</script>
