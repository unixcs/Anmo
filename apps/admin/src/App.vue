<template>
  <router-view />
</template>

<script setup lang="ts">
</script>

<style>
body {
  margin: 0;
  font-family:
    -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB',
    'Microsoft YaHei', sans-serif;
  background: #f5f7fa;
}

/* 手机端全局紧凑化 */
@media (max-width: 768px) {
  .el-main {
    padding: 10px !important;
  }
  .el-table {
    font-size: 12px;
  }
  .el-table .cell {
    padding: 0 6px;
  }
  .el-dialog {
    margin-top: 6vh !important;
  }
  .el-message-box {
    width: 92vw;
    max-width: 420px;
  }
  .el-pagination {
    flex-wrap: wrap;
    row-gap: 6px;
  }
  .el-date-editor.el-input,
  .el-date-editor.el-input__wrapper {
    width: 140px !important;
  }
}
</style>
