<template>
  <RadioGrid
    :value="engine"
    :options="options"
    @update:value="$emit('update:engine', $event as Engine)"
  >
    <template #item="{ option }: RadioGridItem<Engine>">
      <div class="flex flex-row items-center gap-x-1">
        <RichEngineName
          :engine="option.value"
          tag="p"
          class="text-center text-sm !text-main"
        />
        <slot name="suffix" :engine="option.value" />
      </div>
    </template>
  </RadioGrid>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { Engine } from "@/types/proto/v1/common";
import { engineNameV1 } from "@/utils";
import {
  RadioGrid,
  type RadioGridItem,
  type RadioGridOption,
} from "../../Form";
import RichEngineName from "./RichEngineName.vue";

type EngineOption = RadioGridOption<Engine>;

const props = defineProps<{
  engine: Engine | undefined;
  engineList: Engine[];
}>();
defineEmits<{
  (event: "update:engine", engine: Engine): void;
}>();

const options = computed(() => {
  return props.engineList.map<EngineOption>((engine) => ({
    value: engine,
    label: engineNameV1(engine),
  }));
});
</script>
