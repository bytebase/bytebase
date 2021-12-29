<template>
  <div class="mt-2 space-y-6 divide-y divide-block-border">
    <div v-if="allowEdit" class="pt-5 flex justify-end">
      <button
        type="button"
        class="btn-primary"
        :disabled="!allowSave"
        @click.prevent="doSave"
      >{{ $t("common.update") }}</button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { isOwner } from "../utils";
import { Setting } from "../types/setting";

interface LocalState {
}

export default {
  name: "SettingWorkspaceGeneral",
  data() {
    return { placeholder: "{{ DB_NAME_PLACEHOLDER }}" }
  },
  setup() {
    const store = useStore();

    const state = reactive<LocalState>({
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const allowEdit = computed((): boolean => {
      return isOwner(currentUser.value.role);
    });

    const allowSave = computed((): boolean => {
      return false;
    });

    const doSave = () => {
    };

    return {
      state,
      allowEdit,
      allowSave,
      doSave,
    };
  },
};
</script>
