<template>
  <div class="geo-page geo-page--column menu-container">
    <GeoBackground extra />

    <header class="menu-header">
      <div class="header-brand">
        <div class="brand-accent"></div>
        <span class="brand-label">PLATFORM</span>
        <h1 class="brand-title">AI 应用平台</h1>
      </div>
      <button class="geo-btn geo-btn--danger" @click="handleLogout">退出登录</button>
    </header>

    <main class="menu-main">
      <div class="menu-intro">
        <span class="intro-label">MODULES</span>
        <p class="intro-desc">选择功能模块开始体验</p>
      </div>

      <div class="menu-grid">
        <article class="menu-card" @click="$router.push('/ai-chat')">
          <div class="card-geo">
            <span></span><span></span>
          </div>
          <div class="card-icon">
            <el-icon :size="28"><ChatDotRound /></el-icon>
          </div>
          <div class="card-body">
            <span class="card-tag">CHAT</span>
            <h3 class="card-title">AI 聊天</h3>
            <p class="card-desc">与 AI 进行智能对话，支持 RAG 知识库增强</p>
          </div>
          <div class="card-arrow">→</div>
          <div class="card-glow"></div>
        </article>
      </div>
    </main>
  </div>
</template>

<script>
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ChatDotRound } from '@element-plus/icons-vue'
import GeoBackground from '../components/GeoBackground.vue'

export default {
  name: 'MenuView',
  components: {
    ChatDotRound,
    GeoBackground
  },
  setup() {
    const router = useRouter()

    const handleLogout = async () => {
      try {
        await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
        localStorage.removeItem('token')
        ElMessage.success('退出登录成功')
        router.push('/login')
      } catch {
        // 用户取消操作
      }
    }

    return {
      handleLogout
    }
  }
}
</script>

<style scoped>
.menu-container {
  min-height: 100vh;
}

.menu-header {
  position: relative;
  z-index: 2;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 40px;
  background: var(--c-surface-elevated);
  border-bottom: 1px solid var(--c-line);
  box-shadow: var(--shadow-soft);
}

.header-brand {
  position: relative;
}

.brand-accent {
  width: 32px;
  height: 2px;
  background: linear-gradient(90deg, var(--c-accent), var(--c-flow-2));
  margin-bottom: 6px;
  animation: geoAccentShimmer 3s ease-in-out infinite;
}

.brand-label {
  display: block;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  color: var(--c-ink-faint);
  margin-bottom: 2px;
}

.brand-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--c-ink);
}

.menu-main {
  flex: 1;
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 40px;
  gap: 40px;
}

.menu-intro {
  text-align: center;
  animation: geoFadeIn 0.6s ease-out;
}

.intro-label {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  color: var(--c-ink-faint);
}

.intro-desc {
  margin: 8px 0 0;
  font-size: 14px;
  color: var(--c-ink-muted);
}

.menu-grid {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 24px;
  max-width: 900px;
  width: 100%;
}

.menu-card {
  position: relative;
  width: 100%;
  max-width: 380px;
  display: flex;
  align-items: flex-start;
  gap: 20px;
  padding: 28px;
  background: var(--c-surface-elevated);
  border: 1px solid var(--c-line);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-soft);
  cursor: pointer;
  overflow: hidden;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
  animation: geoSlideIn 0.5s ease-out;
}

.menu-card:hover {
  border-color: var(--c-accent);
  box-shadow: var(--shadow-medium), var(--shadow-glow);
  transform: translateY(-2px);
}

.card-geo {
  position: absolute;
  top: 16px;
  right: 16px;
  display: flex;
  gap: 4px;
  opacity: 0.4;
}

.card-geo span {
  display: block;
  border: 1px solid var(--c-line-strong);
}

.card-geo span:nth-child(1) {
  width: 12px;
  height: 12px;
  transform: rotate(8deg);
}

.card-geo span:nth-child(2) {
  width: 8px;
  height: 16px;
  transform: translateY(-2px);
}

.card-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--c-line);
  border-radius: var(--radius-sm);
  color: var(--c-accent);
  background: var(--c-accent-soft);
  transition: box-shadow 0.2s ease;
}

.menu-card:hover .card-icon {
  box-shadow: var(--shadow-glow);
}

.card-body {
  flex: 1;
  min-width: 0;
}

.card-tag {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  color: var(--c-ink-faint);
}

.card-title {
  margin: 6px 0 8px;
  font-size: 18px;
  font-weight: 600;
  color: var(--c-ink);
  letter-spacing: -0.01em;
}

.card-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--c-ink-muted);
}

.card-arrow {
  flex-shrink: 0;
  align-self: center;
  font-size: 18px;
  color: var(--c-ink-faint);
  transition: color 0.2s ease, transform 0.2s ease;
}

.menu-card:hover .card-arrow {
  color: var(--c-accent);
  transform: translateX(4px);
}

.card-glow {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--c-flow-1), var(--c-flow-2), transparent);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.menu-card:hover .card-glow {
  opacity: 0.8;
}

@media (max-width: 768px) {
  .menu-header {
    padding: 16px 20px;
  }

  .menu-main {
    padding: 32px 20px;
  }

  .menu-card {
    max-width: 100%;
  }
}
</style>
