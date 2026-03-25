/**
 * 组件统一导出
 * 提供所有自定义组件的集中访问
 */

// 布局组件
export { default as AppShell } from "./AppShell.vue";

// 数据展示组件
export { default as DataTable } from "./DataTable.vue";
export { default as SimpleTable } from "./SimpleTable.vue";
export { default as EmptyState } from "./EmptyState.vue";
export { default as StatusTag } from "./StatusTag.vue";

// 页面组件
export { default as PageHeader } from "./PageHeader.vue";

// 功能组件
export { default as LogTerminal } from "./LogTerminal.vue";

// UX优化组件
export { default as QuickActionsPanel } from "./QuickActionsPanel.vue";
export { default as SmartListBar } from "./SmartListBar.vue";
export { default as DetailDrawer } from "./DetailDrawer.vue";
export { default as RowActionsMenu } from "./RowActionsMenu.vue";
export { default as SmartForm } from "./SmartForm.vue";
export { default as FormActionBar } from "./FormActionBar.vue";
export { default as FormPageLayout } from "./FormPageLayout.vue";
