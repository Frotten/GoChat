<template>
  <div class="geo-page geo-page--flex-center login-container">
    <GeoBackground />

    <div class="auth-panel geo-panel geo-form-panel">
      <div class="geo-panel__accent"></div>
      <div class="auth-header">
        <span class="geo-panel__label">AUTH</span>
        <h2 class="geo-panel__title">登录</h2>
        <p class="auth-subtitle">欢迎回来，请登录您的账号</p>
      </div>

      <el-form
        ref="loginFormRef"
        :model="loginForm"
        :rules="loginRules"
        label-width="72px"
        class="auth-form"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="loginForm.username"
            placeholder="请输入用户名"
          />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="loginForm.password"
            placeholder="请输入密码"
            type="password"
            show-password
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            :loading="loading"
            @click="handleLogin"
            class="submit-btn"
          >
            登录
          </el-button>
        </el-form-item>
        <el-form-item class="link-item">
          <button type="button" class="geo-btn geo-btn--text geo-btn--block" @click="$router.push('/register')">
            还没有账号？去注册
          </button>
        </el-form-item>
      </el-form>

      <div class="panel-glow"></div>
    </div>
  </div>
</template>

<script>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../utils/api'
import GeoBackground from '../components/GeoBackground.vue'

export default {
  name: 'LoginView',
  components: { GeoBackground },
  setup() {
    const router = useRouter()
    const loginFormRef = ref()
    const loading = ref(false)
    const loginForm = ref({
      username: '',
      password: ''
    })

    const loginRules = {
      username: [
        { required: true, message: '请输入用户名', trigger: 'blur' }
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
      ]
    }

    const handleLogin = async () => {
      try {
        await loginFormRef.value.validate()
        loading.value = true
        const response = await api.post('/user/login', {
          username: loginForm.value.username,
          password: loginForm.value.password
        })
        if (response.data.status_code === 1000) {
          localStorage.setItem('token', response.data.token)
          ElMessage.success('登录成功')
          router.push('/menu')
        } else {
          ElMessage.error(response.data.status_msg || '登录失败')
        }
      } catch (error) {
        console.error('Login error:', error)
        ElMessage.error('登录失败，请重试')
      } finally {
        loading.value = false
      }
    }

    return {
      loginFormRef,
      loading,
      loginForm,
      loginRules,
      handleLogin
    }
  }
}
</script>

<style scoped>
.login-container {
  padding: 24px;
}

.auth-panel {
  width: 100%;
  max-width: 420px;
  padding: 36px 32px 32px;
}

.auth-header {
  margin-bottom: 28px;
}

.auth-subtitle {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--c-ink-muted);
}

.auth-form {
  margin-top: 8px;
}

.submit-btn {
  width: 100%;
  height: 44px;
}

.link-item {
  margin-bottom: 0 !important;
}

.link-item :deep(.el-form-item__content) {
  justify-content: center;
}

.panel-glow {
  position: absolute;
  bottom: 0;
  left: 20%;
  right: 20%;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--c-flow-1), var(--c-flow-2), transparent);
  opacity: 0.5;
}
</style>
