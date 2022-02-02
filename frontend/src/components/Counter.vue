<template>
  <div class="h-8 w-24">
    <div
      class="flex flex-row h-8 w-full rounded-lg relative bg-transparent mt-1 bg-gray-200 text-gray-600 items-center"
    >
      <button
        class="hover:text-gray-700 hover:bg-gray-300 h-full w-14 rounded-l cursor-pointer outline-none"
        :disabled="state.count <= minimum"
        @click="onClick(-5)"
      >
        <span class="m-auto text-xl font-thin">−</span>
      </button>
      <div class="w-full text-center">
        {{ state.count }}
      </div>
      <button
        class="hover:text-gray-700 hover:bg-gray-300 h-full w-14 rounded-r cursor-pointer outline-none"
        @click="onClick(5)"
      >
        <span class="m-auto text-xl font-thin">+</span>
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { watch, reactive } from "vue";

export default {
  props: {
    count: {
      default: 0,
      type: Number,
    },
    minimum: {
      default: 1,
      type: Number,
    },
  },
  emits: ["on-change"],
  setup(props, { emit }) {
    const state = reactive<{ count: number }>({
      count: props.count,
    });
    const onClick = (diff: number) => {
      const count = Math.max(props.minimum, state.count + diff);
      state.count = count;
    };

    watch(
      () => state.count,
      (val) => {
        emit("on-change", val);
      }
    );

    watch(
      () => props.count,
      (val) => {
        state.count = val;
      }
    );

    return {
      state,
      onClick,
    };
  },
};
</script>

<style scoped>
input[type="number"]::-webkit-inner-spin-button,
input[type="number"]::-webkit-outer-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
</style>
