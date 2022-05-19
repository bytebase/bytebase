<template>
  <BBSelect
    :selected-item="selectedStatus"
    :item-list="['OPEN', 'DONE', 'CANCELED']"
    :placeholder="'Unknown Status'"
    :disabled="disabled"
    @select-item="(status, didSelect) => changeStatus(status, didSelect)"
  >
    <template #menuItem="{ item }">
      <span class="flex items-center space-x-2">
        <IssueStatusIcon :issue-status="item" :size="'small'" />
        <span>
          {{ item }}
        </span>
      </span>
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import IssueStatusIcon from "./Issue/IssueStatusIcon.vue";
import { IssueStatus, IssueStatusTransitionType } from "../types";

export default defineComponent({
  name: "IssueStatusSelect",
  components: { IssueStatusIcon },
  props: {
    selectedStatus: {
      type: String as PropType<IssueStatus>,
      default: undefined,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["start-transition"],
  setup(_, { emit }) {
    const changeStatus = (newStatus: IssueStatus, didChange: () => any) => {
      let transition: IssueStatusTransitionType;
      switch (newStatus) {
        case "OPEN":
          transition = "REOPEN";
          break;
        case "DONE":
          transition = "RESOLVE";
          break;
        case "CANCELED":
          transition = "CANCEL";
          break;
      }
      emit("start-transition", transition, didChange);
    };

    return { changeStatus };
  },
});
</script>
