<template>
  <BBSelect
    :selected-item="selectedInstanceUser"
    :item-list="instanceUserList"
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
import { useInstanceStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { InstanceUser } from "@/types/InstanceUser";
import { defineComponent, ref, watch } from "vue";

export default defineComponent({
  name: "InstanceUserSelect",
  components: {},
  props: {
    selectedId: {
      type: Number,
      default: UNKNOWN_ID,
    },
    instanceId: {
      type: Number,
      default: UNKNOWN_ID,
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

    const handleSelectItem = (instanceUser: InstanceUser) => {
      emit("select", instanceUser.id);
    };

    return {
      selectedInstanceUser,
      instanceUserList,
      handleSelectItem,
    };
  },
});
</script>
