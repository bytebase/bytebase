<template>
  <BBSelect
    :selected-item="state.selectedInstance"
    :item-list="instanceList"
    :disabled="disabled"
    :placeholder="$t('instance.select')"
    :show-prefix-item="true"
    :error="!validate()"
    @select-item="(instance: ComposedInstance) => $emit('select-instance-id', instance.uid)"
  >
    <template #menuItem="{ item: instance }: { item: ComposedInstance }">
      <div class="flex items-center gap-x-2">
        <InstanceV1EngineIcon :instance="instance" />
        <span>{{ instanceV1Name(instance) }}</span>
      </div>
    </template>
  </BBSelect>
</template>

<script lang="ts" setup>
import { computed, PropType, reactive, watch } from "vue";
import { useInstanceV1List } from "@/store";
import { State } from "@/types/proto/v1/common";
import { instanceV1Name } from "@/utils";
import { ComposedInstance, UNKNOWN_ID } from "../types";
import { InstanceV1EngineIcon } from "./v2";

interface LocalState {
  selectedInstance?: ComposedInstance;
}

const props = defineProps({
  selectedId: {
    type: String,
    default: undefined,
  },
  environmentId: {
    type: String,
    default: undefined,
  },
  filter: {
    type: Function as PropType<(instance: ComposedInstance) => boolean>,
    default: undefined,
  },
  disabled: {
    default: false,
    type: Boolean,
  },
  required: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits<{
  (event: "select-instance-id", uid: string | undefined): void;
}>();

const state = reactive<LocalState>({
  selectedInstance: undefined,
});
const { instanceList: allInstanceList, ready } = useInstanceV1List(
  true /* showDeleted */
);

const rawInstanceList = computed(() => {
  const list = [...allInstanceList.value];
  if (props.environmentId && props.environmentId !== String(UNKNOWN_ID)) {
    return list.filter(
      (instance) => instance.environmentEntity.uid === props.environmentId
    );
  }
  return list;
});

const instanceList = computed(() => {
  const list = rawInstanceList.value.filter((instance) => {
    if (instance.state === State.ACTIVE) {
      return true;
    }
    // instance.rowStatus === "ARCHIVED"
    if (instance.uid === state.selectedInstance?.uid) {
      return true;
    }
    return false;
  });

  if (!props.filter) {
    return list;
  }
  return list.filter(props.filter);
});

const validate = (): boolean => {
  if (!props.required) {
    return true;
  }
  return (
    !!state.selectedInstance &&
    state.selectedInstance.uid !== String(UNKNOWN_ID)
  );
};

// The instance list might change if environmentId changes, and the previous selected id
// might not exist in the new list. In such case, we need to invalidate the selection
// and emit the event.
const invalidateSelectionIfNeeded = () => {
  if (
    ready.value &&
    state.selectedInstance &&
    !instanceList.value.find((item) => item.uid == state.selectedInstance?.uid)
  ) {
    state.selectedInstance = undefined;
    emit("select-instance-id", undefined);
  }
};

watch(
  [() => props.selectedId, instanceList],
  ([selectedId, list]) => {
    invalidateSelectionIfNeeded();
    state.selectedInstance = list.find(
      (instance) => instance.uid === selectedId
    );
  },
  { immediate: true }
);
</script>
