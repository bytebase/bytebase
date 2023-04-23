<template>
  <BBSelect
    :selected-item="selectedInstanceUser"
    :item-list="filteredInstanceUserList"
    :placeholder="$t('instance.select-database-user')"
    :show-prefix-item="true"
    @select-item="handleSelectItem"
  >
    <template #menuItem="{ item: instance }">
      {{ instance.name }}
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { PropType, computed, defineComponent, ref, watch } from "vue";

import { useInstanceStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { InstanceUser } from "@/types/InstanceUser";

export default defineComponent({
  name: "InstanceUserSelect",
  components: {},
  props: {
    selectedId: {
      type: String,
      default: undefined,
    },
    instanceId: {
      type: Number,
      default: UNKNOWN_ID,
    },
    filter: {
      type: Function as PropType<(user: InstanceUser) => boolean>,
      default: undefined,
    },
  },
  emits: ["select"],
  setup(props, { emit }) {
    const selectedInstanceUser = ref<InstanceUser>();
    const instanceUserList = ref<InstanceUser[]>([]);

    watch(
      () => props.instanceId,
      async () => {
        selectedInstanceUser.value = undefined;
        instanceUserList.value =
          await useInstanceStore().fetchInstanceUserListById(props.instanceId);
        emit("select", undefined);
      },
      { immediate: true }
    );

    watch(
      () => props.selectedId,
      () => {
        selectedInstanceUser.value = instanceUserList.value.find(
          (user) => user.id === props.selectedId
        );
      }
    );

    const filteredInstanceUserList = computed(() => {
      if (!props.filter) return instanceUserList.value;
      return instanceUserList.value.filter(props.filter);
    });

    const handleSelectItem = (instanceUser: InstanceUser) => {
      emit("select", instanceUser.id);
    };

    return {
      selectedInstanceUser,
      filteredInstanceUserList,
      handleSelectItem,
    };
  },
});
</script>
