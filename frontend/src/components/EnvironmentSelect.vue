<template>
  <BBSelect
    :selected-item="state.selectedEnvironment"
    :item-list="environmentList"
    :disabled="disabled"
    :placeholder="$t('environment.select')"
    :show-prefix-item="true"
    @select-item="(env: Environment) => $emit('select-environment-id', env.id)"
  >
    <template #menuItem="{ item: environment }">
      <div class="flex items-center">
        {{ environmentName(environment) }}
        <ProductionEnvironmentIcon class="ml-1" :environment="environment" />
      </div>
    </template>
  </BBSelect>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, watch } from "vue";
import { Environment } from "../types";
import { useEnvironmentList } from "@/store";

interface LocalState {
  selectedEnvironment?: Environment;
}

export default defineComponent({
  name: "EnvironmentSelect",
  props: {
    selectedId: {
      type: Number,
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

    const rawEnvironmentList = useEnvironmentList(["NORMAL", "ARCHIVED"]);
    const environmentList = computed(() => {
      const list = [...rawEnvironmentList.value].reverse();

      return list.filter((env) => {
        if (env.rowStatus === "NORMAL") {
          return true;
        }

        // env.rowStatus === "ARCHIVED"
        if (env.id === state.selectedEnvironment?.id) {
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
          !list.find((item: Environment) => item.id == props.selectedId)
        ) {
          // auto select the first NORMAL environment
          const defaultEnvironment = list.find(
            (env) => env.rowStatus === "NORMAL"
          );
          state.selectedEnvironment = defaultEnvironment;
          emit("select-environment-id", defaultEnvironment?.id);
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
          (item: Environment) => item.id == state.selectedEnvironment?.id
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
          (env) => env.id === selectedId
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
