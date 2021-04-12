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

export default {
  name: "ProjectListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareProjectList = () => {
      store
        .dispatch("project/fetchProjectListByUser", currentUser.value.id)
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareProjectList);

    const outlineItemList = computed((): BBOutlineItem[] => {
      const projectList = store.getters["project/projectListByUser"](
        currentUser.value.id
      );
      return projectList.map(
        (item: Project): BBOutlineItem => {
          return {
            id: item.id,
            name: item.name,
            link: `/project/${projectSlug(item)}`,
          };
        }
      );
    });

    return {
      outlineItemList,
    };
  },
};
</script>
