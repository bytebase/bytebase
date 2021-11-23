<template>
  <BBSelect
    :selectedItem="selectedStatus"
    :itemList="['OPEN', 'DONE', 'CANCELED']"
    :placeholder="'Unknown Status'"
    :disabled="disabled"
    @select-item="(status, didSelect) => changeStatus(status, didSelect)"
  >
    <template #menuItem="{ item }">
      <span class="flex items-center space-x-2">
        <IssueStatusIcon :issueStatus="item" :size="'small'" />
        <span>
          {{ item }}
        </span>
      </span>
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { PropType } from "vue";
import IssueStatusIcon from "../components/IssueStatusIcon.vue";
import { IssueStatus, IssueStatusTransitionType } from "../types";

export default {
  name: "IssueStatusSelect",
  components: { IssueStatusIcon },
  emits: ["start-transition"],
  props: {
    selectedStatus: {
      type: String as PropType<IssueStatus>,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  setup(_, { emit }) {
    const changeStatus = (newStatus: IssueStatus, didChange: () => {}) => {
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
};
</script>
