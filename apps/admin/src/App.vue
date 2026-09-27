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
  /* §26 表单单列优先：inline 表单在手机上一律纵向堆叠 */
  .el-form--inline .el-form-item {
    display: block;
    margin-right: 0;
    width: 100%;
  }
  .el-form--inline .el-form-item .el-input,
  .el-form--inline .el-form-item .el-select,
  .el-form--inline .el-form-item .el-input-number,
  .el-form--inline .el-form-item .el-time-select {
    width: 100% !important;
  }
  /* 各页工具栏允许换行，查询按钮不被挤出屏幕 */
  .toolbar,
  .tab-head,
  .stat-ops {
    flex-wrap: wrap;
    row-gap: 8px;
  }
  body {
    /* 页面级禁止横向滚动（§25）；手机端列表一律卡片流 */
    overflow-x: hidden;
  }
  .el-card__body {
    padding: 12px !important;
  }
}
/* §27 底部固定操作栏（页面按需启用 .fixed-bar），适配安全区 */
.fixed-bar {
  position: sticky;
  bottom: 0;
  z-index: 10;
  display: flex;
  gap: 10px;
  padding: 10px 0 calc(10px + env(safe-area-inset-bottom));
  background: #fff;
  border-top: 1px solid #e4e7ed;
}
</style>
