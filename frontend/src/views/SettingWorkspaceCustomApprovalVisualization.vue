<template>
  <div class="custom-approval-visualization">
    <h1>审批流程可视化</h1>
    <p>这是一个用于可视化审批流程的页面。</p>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';

const approvalFlows = ref([]);

const fetchApprovalFlows = async () => {
  // TODO: 从后端获取审批流程数据
  approvalFlows.value = [];
};

onMounted(() => {
  fetchApprovalFlows();
});
</script>

<style scoped>
.custom-approval-visualization {
  padding: 24px;
}

h1 {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 16px;
}

p {
  font-size: 14px;
  color: #666;
}
</style>