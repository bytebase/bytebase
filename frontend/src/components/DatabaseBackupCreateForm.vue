<template>
  <form
    class="space-y-6 divide-y divide-block-border"
    @submit.prevent="$emit('create', state.backupName, state.comment)"
  >
    <div class="space-y-4">
      <div class="grid grid-cols-3 gap-y-6 gap-x-4">
        <div class="col-span-3">
          <label for="name" class="textlabel">
            Backup name <span class="text-red-600">*</span>
          </label>
          <input
            required
            id="name"
            name="name"
            type="text"
            class="textfield mt-1 w-full"
            v-model="state.backupName"
          />
        </div>
      </div>

      <div class="sm:col-span-4 w-112 min-w-full">
        <label for="comment" class="textlabel"> Comment </label>
        <div class="mt-1">
          <textarea
            ref="commentTextArea"
            rows="3"
            class="
              textarea
              block
              w-full
              resize-none
              mt-1
              text-sm text-control
              rounded-md
              whitespace-pre-wrap
            "
            placeholder="(Optional) Add a note..."
            v-model="state.comment"
            @input="
              (e) => {
                sizeToFit(e.target);
              }
            "
            @focus="
              (e) => {
                sizeToFit(e.target);
              }
            "
          ></textarea>
        </div>
      </div>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="$emit('cancel')"
      >
        Cancel
      </button>
      <button
        type="submit"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
      >
        Create backup
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, PropType, reactive, ref } from "vue";
import { Database } from "../types";
import { isEmpty } from "lodash";

interface LocalState {
  backupName: string;
  comment: string;
}

export default {
  name: "DatabaseBackupCreateForm",
  emits: ["create", "cancel"],
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
  },
  components: {},
  setup(props, ctx) {
    const commentTextArea = ref("");

    const state = reactive<LocalState>({
      backupName: `${
        props.database.name
      }-${new Date().toLocaleTimeString()}-manual`,
      comment: "",
    });

    const allowCreate = computed(() => {
      return !isEmpty(state.backupName);
    });

    return {
      commentTextArea,
      state,
      allowCreate,
    };
  },
};
</script>
