<template>
  <div id="app">
    <router-view v-slot="{ Component }">
      <transition name="page" mode="out-in">
        <component :is="Component" />
      </transition>
    </router-view>
  </div>
</template>

<script>
export default {
  name: 'App'
}
</script>

<style>
@import './styles/geo-theme.css';

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  height: 100%;
  font-family: var(--font-sans);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background: var(--c-bg);
  color: var(--c-ink);
}

#app {
  height: 100%;
}

/* 页面切换动画 */
.page-enter-active,
.page-leave-active {
  transition: opacity 0.35s ease, transform 0.35s ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(12px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* 全局滚动条 */
::-webkit-scrollbar {
  width: 4px;
  height: 4px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: var(--c-line-strong);
  border-radius: 2px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--c-ink-faint);
}

/* Element Plus 全局微调 */
.el-button {
  font-weight: 500;
  border-radius: var(--radius-sm);
}

.el-input {
  border-radius: var(--radius-sm);
}

.el-card {
  border-radius: var(--radius-md);
}

.el-message {
  border-radius: var(--radius-sm);
  border: 1px solid var(--c-line);
  box-shadow: var(--shadow-medium);
}

.el-message-box {
  border-radius: var(--radius-md);
}

@media (max-width: 768px) {
  .page-enter-from,
  .page-leave-to {
    transform: translateY(8px);
  }
}
</style>
