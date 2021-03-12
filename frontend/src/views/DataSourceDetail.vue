<template>
  <div>DataSource Detail {{ dataSourceSlug }}</div>
</template>

<script lang="ts">
export default {
  name: "DataSourceDetail",
  props: {
    instanceSlug: {
      required: true,
      type: String,
    },
    dataSourceSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, ctx) {},
};
</script>
