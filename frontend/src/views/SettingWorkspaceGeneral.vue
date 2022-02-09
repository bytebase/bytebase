<template>
  <div class="mt-2 space-y-6 divide-y divide-block-border">
    <div v-if="allowEdit" class="pt-5 flex justify-end">
      <button
        type="button"
        class="btn-primary"
        :disabled="!allowSave"
        @click.prevent="doSave"
      >
        {{ $t("common.update") }}
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive } from "vue";
import { useStore } from "vuex";
import { isOwner } from "../utils";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "SettingWorkspaceGeneral",
  setup() {
    const store = useStore();

    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const allowEdit = computed((): boolean => {
      return isOwner(currentUser.value.role);
    });

    const allowSave = computed((): boolean => {
      return false;
    });

    const doSave = () => {
      // do nothing
    };

    return {
      state,
      allowEdit,
      allowSave,
      doSave,
    };
  },
  data() {
    return { placeholder: "{{ DB_NAME_PLACEHOLDER }}" };
  },
});
</script>
