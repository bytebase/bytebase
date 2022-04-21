<template>
  <BBDialog
    ref="dialog"
    :title="'Migration mode'"
    :positive-text="$t('common.next')"
  >
    <div class="w-[24rem] space-y-4 pl-8 pr-2 pb-4">
      <div class="flex items-start space-x-2">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent mt-0.5"
          value="normal"
        />
        <div @click="state.mode = 'normal'">
          <div class="textlabel">Normal migration</div>
          <div class="textinfolabel">
            Perform schema change directly on target table.
          </div>
        </div>
      </div>
      <div class="flex items-start space-x-2">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent mt-0.5"
          value="online"
        />
        <div @click="state.mode = 'online'">
          <div class="textlabel">Online migration</div>
          <div class="textinfolabel">
            <p>
              Perform schema change on a duplicated table. Switch table names
              after data synchronization.
            </p>
            <p>
              <a href="#" class="normal-link inline-flex items-center"
                >Learn more.
                <heroicons-outline:external-link class="inline-block"
              /></a>
            </p>
          </div>
        </div>
      </div>
    </div>
  </BBDialog>
</template>

<script lang="ts" setup>
import { reactive, ref } from "vue";
import { BBDialog } from "@/bbkit";

type Mode = "normal" | "online";

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);

const state = reactive({
  mode: "normal" as Mode,
});

const open = (): Promise<{ result: boolean; mode: Mode }> => {
  state.mode = "normal"; // reset state

  return dialog.value!.open().then(
    (result) => {
      return {
        result,
        mode: state.mode,
      };
    },
    (_error) => {
      return {
        result: false,
        mode: "normal",
      };
    }
  );
};

defineExpose({ open });
</script>
