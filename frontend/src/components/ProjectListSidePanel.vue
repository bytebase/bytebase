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
import { projectName, projectSlug } from "../utils";
import { BBOutlineItem } from "../bbkit/types";

export default {
  name: "ProjectListSidePanel",
  props: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareProjectList = () => {
      store
        .dispatch("project/fetchProjectListByUser", {
          userId: currentUser.value.id,
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareProjectList);

    const outlineItemList = computed((): BBOutlineItem[] => {
      const projectList = store.getters["project/projectListByUser"](
        currentUser.value.id,
        "NORMAL"
      );
      return projectList.map(
        (item: Project): BBOutlineItem => {
          return {
            id: item.id,
            name: projectName(item),
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
