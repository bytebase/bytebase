<template>
  <slot />
</template>

<script lang="ts">
import { watchEffect, defineComponent } from "vue";
import { useStore } from "vuex";
// import { idFromSlug } from "../utils";

export default defineComponent({
  name: "ProvideSqlEditorContext",
  // props: {
  //   instanceSlug: {
  //     required: true,
  //     type: String,
  //   },
  // },
  async setup(props) {
    const store = useStore();

    const prepareInstanceContext = async function () {
      await Promise.all([
        store.dispatch("instance/fetchInstanceList")
        // store.dispatch(
        //   "database/fetchDatabaseListByInstanceId",
        //   idFromSlug(props.instanceSlug)
        // ),
        // store.dispatch(
        //   "instance/fetchInstanceUserListById",
        //   idFromSlug(props.instanceSlug)
        // ),
      ]);
    };

    watchEffect(prepareInstanceContext);
  },
});
</script>
