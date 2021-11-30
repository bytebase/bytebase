<template>
  <BBOutline
    :id="'project'"
    :title="'Projects'"
    :item-list="outlineItemList"
    :allow-collapse="false"
  />
</template>

<script lang="ts">
import { computed, watchEffect } from "vue";
import { useStore } from "vuex";

import { Project, UNKNOWN_ID } from "../types";
import { projectName, projectSlug } from "../utils";
import { BBOutlineItem } from "../bbkit/types";

export default {
  name: "ProjectListSidePanel",
  props: {},
  setup() {
    const store = useStore();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareProjectList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch("project/fetchProjectListByUser", {
          userId: currentUser.value.id,
        });
      }
    };

    watchEffect(prepareProjectList);

    const outlineItemList = computed((): BBOutlineItem[] => {
      const projectList = store.getters["project/projectListByUser"](
        currentUser.value.id,
        "NORMAL"
      );
      return projectList
        .map((item: Project): BBOutlineItem => {
          return {
            id: item.id.toString(),
            name: projectName(item),
            link: `/project/${projectSlug(item)}#overview`,
          };
        })
        .sort((a: any, b: any) => {
          return a.name.localeCompare(b.name);
        });
    });

    return {
      outlineItemList,
    };
  },
};
</script>
