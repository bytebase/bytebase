<template>
  <BBSelect
    :selected-item="state.selectedEnvironment"
    :item-list="environmentList"
    :disabled="disabled"
    :placeholder="$t('environment.select')"
    :show-prefix-item="true"
    @select-item="(env: Environment) => $emit('select-environment-id', env.uid)"
  >
    <template #menuItem="{ item: environment }">
      <EnvironmentV1Name :environment="environment" :link="false" />
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, watch } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useEnvironmentV1List } from "@/store";
import { State } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";

interface LocalState {
  selectedEnvironment?: Environment;
}

export default defineComponent({
  name: "EnvironmentSelect",
  components: { EnvironmentV1Name },
  props: {
    selectedId: {
      type: String,
      default: undefined,
    },
    selectDefault: {
      default: true,
      type: Boolean,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["select-environment-id"],
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      selectedEnvironment: undefined,
    });

    const rawEnvironmentList = useEnvironmentV1List(true /* showDeleted */);
    const environmentList = computed(() => {
      const list = [...rawEnvironmentList.value].reverse();

      return list.filter((env) => {
        if (env.state === State.ACTIVE) {
          return true;
        }

        // env.rowStatus === "ARCHIVED"
        if (env.uid === state.selectedEnvironment?.uid) {
          return true;
        }
        return false;
      });
    });

    const autoSelectDefaultIfNeeded = () => {
      if (!props.selectDefault) {
        return;
      }

      const list = environmentList.value;
      if (list.length > 0) {
        if (
          !props.selectedId ||
          !list.find((item: Environment) => item.uid === props.selectedId)
        ) {
          // auto select the first NORMAL environment
          const defaultEnvironment = list.find(
            (env) => env.state === State.ACTIVE
          );
          state.selectedEnvironment = defaultEnvironment;
          emit("select-environment-id", defaultEnvironment?.uid);
        }
      }
    };

    onMounted(() => {
      autoSelectDefaultIfNeeded();
    });

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedEnvironment &&
        !environmentList.value.find(
          (item: Environment) => item.uid == state.selectedEnvironment?.uid
        )
      ) {
        state.selectedEnvironment = undefined;
        emit("select-environment-id", undefined);
      }
    };

    watch(
      () => props.selectedId,
      (selectedId) => {
        invalidateSelectionIfNeeded();
        state.selectedEnvironment = environmentList.value.find(
          (env) => env.uid === String(selectedId)
        );
      },
      { immediate: true }
    );

    return {
      state,
      environmentList,
    };
  },
});
</script>
