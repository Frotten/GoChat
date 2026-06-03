<template>
  <div class="geo-page geo-page--flex-center register-container">
    <GeoBackground extra />

    <div class="auth-panel geo-panel geo-form-panel">
      <div class="geo-panel__accent"></div>
      <div class="auth-header">
        <span class="geo-panel__label">REGISTER</span>
        <h2 class="geo-panel__title">注册</h2>
        <p class="auth-subtitle">创建账号，开启 AI 体验</p>
      </div>

      <el-form
        ref="registerFormRef"
        :model="registerForm"
        :rules="registerRules"
        label-width="72px"
        class="auth-form"
      >
        <el-form-item label="邮箱" prop="email">
          <el-input
            v-model="registerForm.email"
            placeholder="请输入邮箱"
            type="email"
          />
        </el-form-item>
        <el-form-item label="验证码" prop="captcha">
          <div class="captcha-row">
            <el-input
              v-model="registerForm.captcha"
              placeholder="请输入验证码"
              class="captcha-input"
            />
            <el-button
              type="primary"
              :loading="codeLoading"
              :disabled="countdown > 0"
              @click="sendCode"
              class="captcha-btn"
            >
              {{ countdown > 0 ? `${countdown}s` : '发送验证码' }}
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="registerForm.password"
            placeholder="请输入密码"
            type="password"
            show-password
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input
            v-model="registerForm.confirmPassword"
            placeholder="请再次输入密码"
            type="password"
            show-password
          />
        </el-form-item>
        <el-form-item>
          <el-button
            type="primary"
            :loading="loading"
            @click="handleRegister"
            class="submit-btn"
          >
            注册
          </el-button>
        </el-form-item>
        <el-form-item class="link-item">
          <button type="button" class="geo-btn geo-btn--text geo-btn--block" @click="$router.push('/login')">
            已有账号？去登录
          </button>
        </el-form-item>
      </el-form>

      <div class="panel-glow"></div>
    </div>
  </div>
</template>

<script>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../utils/api'
import GeoBackground from '../components/GeoBackground.vue'

export default {
  name: 'RegisterView',
  components: { GeoBackground },
  setup() {
    const router = useRouter()
    const registerFormRef = ref()
    const loading = ref(false)
    const codeLoading = ref(false)
    const countdown = ref(0)

    const registerForm = reactive({
      email: '',
      captcha: '',
      password: '',
      confirmPassword: ''
    })

    const validateConfirmPassword = (rule, value, callback) => {
      if (value !== registerForm.password) {
        callback(new Error('两次输入密码不一致'))
      } else {
        callback()
      }
    }

    const registerRules = {
      email: [
        { required: true, message: '请输入邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
      ],
      captcha: [
        { required: true, message: '请输入验证码', trigger: 'blur' }
      ],
      password: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
      ],
      confirmPassword: [
        { required: true, message: '请确认密码', trigger: 'blur' },
        { validator: validateConfirmPassword, trigger: 'blur' }
      ]
    }

    const sendCode = async () => {
      if (!registerForm.email) {
        ElMessage.warning('请先输入邮箱')
        return
      }
      try {
        codeLoading.value = true
        const response = await api.post('/user/captcha', { email: registerForm.email })
        if (response.data.status_code === 1000) {
          ElMessage.success('验证码发送成功')
          countdown.value = 60
          const timer = setInterval(() => {
            countdown.value--
            if (countdown.value <= 0) {
              clearInterval(timer)
            }
          }, 1000)
        } else {
          ElMessage.error(response.data.status_msg || '验证码发送失败')
        }
      } catch (error) {
        console.error('Send code error:', error)
        const msg = error.response?.data?.status_msg
        ElMessage.error(msg || '验证码发送失败：请确认后端已启动(9090)且 Redis 可连接')
      } finally {
        codeLoading.value = false
      }
    }

    const handleRegister = async () => {
      try {
        await registerFormRef.value.validate()
        loading.value = true
        const response = await api.post('/user/register', {
              email: registerForm.email,
              captcha: registerForm.captcha,
              password: registerForm.password
        })
        if (response.data.status_code === 1000) {
          ElMessage.success('注册成功，请登录')
          router.push('/login')
        } else {
          ElMessage.error(response.data.status_msg || '注册失败')
        }
      } catch (error) {
        console.error('Register error:', error)
        ElMessage.error('注册失败，请重试')
      } finally {
        loading.value = false
      }
    }

    return {
      registerFormRef,
      loading,
      codeLoading,
      countdown,
      registerForm,
      registerRules,
      sendCode,
      handleRegister
    }
  }
}
</script>

<style scoped>
.register-container {
  padding: 24px;
}

.auth-panel {
  width: 100%;
  max-width: 440px;
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

.captcha-row {
  display: flex;
  gap: 10px;
  width: 100%;
}

.captcha-input {
  flex: 1;
  min-width: 0;
}

.captcha-btn {
  flex-shrink: 0;
  white-space: nowrap;
  height: 32px;
  padding: 0 14px;
  font-size: 12px;
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
