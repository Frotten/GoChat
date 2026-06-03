<template>
  <div class="ai-chat-container geo-page">
    <GeoBackground />

    <!-- 左侧会话列表 -->
    <aside class="sessions-list">
      <div class="sessions-list-header">
        <div class="header-accent"></div>
        <span class="header-label">SESSIONS</span>
        <h2 class="header-title">会话列表</h2>
        <button class="new-chat-btn" @click="createNewSession">
          <span class="btn-icon">+</span>
          <span>新聊天</span>
        </button>
      </div>
      <ul class="sessions-list-ul">
        <li
          v-for="sessions in sessions"
          :key="sessions.id"
          :class="['sessions-item', { active: currentSessionId === sessions.id }]"
          @click="switchSession(sessions.id)"
        >
          <span class="session-indicator"></span>
          <span class="session-title">{{ sessions.name || `会话 ${sessions.id}` }}</span>
          <button class="delete-session-btn" @click.stop="deleteSession(sessions.id)" title="删除会话">×</button>
        </li>
      </ul>
    </aside>

    <!-- 右侧聊天区域 -->
    <main class="chat-section">
      <header class="top-bar">
        <button class="back-btn" @click="$router.push('/menu')">
          <span class="back-arrow">←</span>
          <span>返回</span>
        </button>
        <div class="top-divider"></div>
        <button class="sync-btn" @click="syncHistory" :disabled="!currentSessionId || tempSession">
          同步历史
        </button>
        <label class="stream-toggle" for="streamingMode">
          <input type="checkbox" id="streamingMode" v-model="isStreaming" />
          <span class="toggle-track">
            <span class="toggle-thumb"></span>
          </span>
          <span class="toggle-label">流式响应</span>
        </label>
      </header>

      <div class="chat-messages" ref="messagesRef">
        <div v-if="currentMessages.length === 0" class="empty-state">
          <div class="empty-geo">
            <span></span><span></span><span></span>
          </div>
          <p class="empty-title">开始对话</p>
          <p class="empty-desc">输入问题，与 AI 展开智能交流</p>
        </div>
        <div
          v-for="(message, index) in currentMessages"
          :key="index"
          :class="['message', message.role === 'user' ? 'user-message' : 'ai-message']"
        >
          <div class="message-avatar">{{ message.role === 'user' ? 'U' : 'AI' }}</div>
          <div class="message-body">
            <div class="message-header">
              <span class="message-role">{{ message.role === 'user' ? '你' : 'AI' }}</span>
              <span v-if="message.meta && message.meta.status === 'streaming'" class="streaming-indicator">
                <span></span><span></span><span></span>
              </span>
            </div>
            <div class="message-content" v-html="renderMarkdown(message.content)"></div>
          </div>
        </div>
      </div>

      <footer class="chat-input">
        <div class="input-frame">
          <div class="input-glow"></div>
          <textarea
            v-model="inputMessage"
            placeholder="请输入你的问题..."
            @keydown.enter.exact.prevent="sendMessage"
            :disabled="loading"
            ref="messageInput"
            rows="1"
          ></textarea>
          <button
            type="button"
            :disabled="!inputMessage.trim() || loading"
            @click="sendMessage"
            class="send-btn"
          >
            <span v-if="loading" class="send-loading"></span>
            <span>{{ loading ? '发送中' : '发送' }}</span>
          </button>
        </div>
      </footer>
    </main>
  </div>
</template>

<script>


import { ref, nextTick, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../utils/api'
import GeoBackground from '../components/GeoBackground.vue'

export default {
  name: 'AIChat',
  components: { GeoBackground },
  setup() {

    const sessions = ref({})               
    const currentSessionId = ref(null)    
    const tempSession = ref(false)        
    const currentMessages = ref([])      
    const inputMessage = ref('')
    const loading = ref(false)
    const messagesRef = ref(null)
    const messageInput = ref(null)
    const isStreaming = ref(false)


    const renderMarkdown = (text) => {
      if (!text && text !== '') return ''
      return String(text)
        .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
        .replace(/\*(.*?)\*/g, '<em>$1</em>')
        .replace(/`(.*?)`/g, '<code>$1</code>')
        .replace(/\n/g, '<br>')
    }

    const loadSessions = async () => {
      try {
        const response = await api.get('/AI/chat/sessions')
        if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.sessions)) {
          const sessionMap = {}
          response.data.sessions.forEach(s => {
            const sid = String(s.sessionId)
            sessionMap[sid] = {
              id: sid,
              name: s.name || `会话 ${sid}`,
              messages: [] // lazy load
            }
          })
          sessions.value = sessionMap
        }
      } catch (error) {
        console.error('Load sessions error:', error)
      }
    }

    const createNewSession = () => {
      currentSessionId.value = 'temp'
      tempSession.value = true
      currentMessages.value = []
      // focus input
      nextTick(() => {
        if (messageInput.value) messageInput.value.focus()
      })
    }

    const switchSession = async (sessionId) => {
      if (!sessionId) return
      currentSessionId.value = String(sessionId)
      tempSession.value = false

      // lazy load history if not present
      if (!sessions.value[sessionId].messages || sessions.value[sessionId].messages.length === 0) {
        try {
          const response = await api.post('/AI/chat/history', { sessionId: currentSessionId.value })
          if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.history)) {
            const messages = response.data.history.map(item => ({
              role: item.is_user ? 'user' : 'assistant',
              content: item.content
            }))
            sessions.value[sessionId].messages = messages
          }
        } catch (err) {
          console.error('Load history error:', err)
        }
      }


      currentMessages.value = [...(sessions.value[sessionId].messages || [])]
      await nextTick()
      scrollToBottom()
    }

    const syncHistory = async () => {
      if (!currentSessionId.value || tempSession.value) {
        ElMessage.warning('请选择已有会话进行同步')
        return
      }
      try {
        const response = await api.post('/AI/chat/history', { sessionId: currentSessionId.value })
        if (response.data && response.data.status_code === 1000 && Array.isArray(response.data.history)) {
          const messages = response.data.history.map(item => ({
            role: item.is_user ? 'user' : 'assistant',
            content: item.content
          }))
          sessions.value[currentSessionId.value].messages = messages
          currentMessages.value = [...messages]
          await nextTick()
          scrollToBottom()
        } else {
          ElMessage.error('无法获取历史数据')
        }
      } catch (err) {
        console.error('Sync history error:', err)
        ElMessage.error('请求历史数据失败')
      }
    }

    const deleteSession = async (sessionId) => {
      if (!sessionId) return
      if (!window.confirm('确定删除该会话吗？')) return
      try {
        const response = await api.post('/AI/chat/delete-session', { sessionId })
        if (response.data && response.data.status_code === 1000) {
          delete sessions.value[sessionId]
          sessions.value = { ...sessions.value }
          if (currentSessionId.value === sessionId) {
            currentSessionId.value = null
            currentMessages.value = []
            tempSession.value = false
          }
          ElMessage.success('会话已删除')
        } else {
          ElMessage.error(response.data?.status_msg || '删除失败')
        }
      } catch (err) {
        console.error('Delete session error:', err)
        ElMessage.error('删除会话失败')
      }
    }

    const sendMessage = async () => {
      if (!inputMessage.value || !inputMessage.value.trim()) {
        ElMessage.warning('请输入消息内容')
        return
      }

      const userMessage = {
        role: 'user',
        content: inputMessage.value
      }
      const currentInput = inputMessage.value
      inputMessage.value = ''


      currentMessages.value.push(userMessage)
      await nextTick()
      scrollToBottom()

      try {
        loading.value = true
        if (isStreaming.value) {

          await handleStreaming(currentInput)
        } else {

          await handleNormal(currentInput)
        }
      } catch (err) {
        console.error('Send message error:', err)
        ElMessage.error('发送失败，请重试')

        if (!tempSession.value && currentSessionId.value && sessions.value[currentSessionId.value] && sessions.value[currentSessionId.value].messages) {

          const sessionArr = sessions.value[currentSessionId.value].messages
          if (sessionArr && sessionArr.length) sessionArr.pop()
        }
        currentMessages.value.pop()
      } finally {
        if (!isStreaming.value) {
          loading.value = false
        }
        await nextTick()
        scrollToBottom()
      }
    }


    async function handleStreaming(question) {

      const aiMessage = {
        role: 'assistant',
        content: '',
        meta: { status: 'streaming' } // mark streaming
      }


      const aiMessageIndex = currentMessages.value.length
      currentMessages.value.push(aiMessage)

      if (!tempSession.value && currentSessionId.value && sessions.value[currentSessionId.value]) {
        if (!sessions.value[currentSessionId.value].messages) sessions.value[currentSessionId.value].messages = []
        sessions.value[currentSessionId.value].messages.push({ role: 'assistant', content: '' })
      }


      const url = tempSession.value
        ? '/api/AI/chat/send-stream-new-session'
        : '/api/AI/chat/send-stream'           

      const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token') || ''}`
      }

      const body = tempSession.value
        ? { question: question }
        : { question: question, sessionId: currentSessionId.value }

      try {
        // 创建 fetch 连接读取 SSE 流
        const response = await fetch(url, {
          method: 'POST',
          headers,
          body: JSON.stringify(body)
        })

        if (!response.ok) {
          loading.value = false
          throw new Error('Network response was not ok')
        }

        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''

        // 读取流数据
        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          const chunk = decoder.decode(value, { stream: true })
          buffer += chunk

          // 按行分割
          const lines = buffer.split('\n')
          buffer = lines.pop() || '' // 保留未完成的行

          for (const line of lines) {
            const trimmedLine = line.trim()
            if (!trimmedLine) continue

            // 处理 SSE 格式：data: <content>
            if (trimmedLine.startsWith('data:')) {
              const data = trimmedLine.slice(5).trim()
              console.log('[SSE] Received:', data) // 调试日志

              if (data === '[DONE]') {
                // 流结束
                console.log('[SSE] Stream done')
                loading.value = false
                currentMessages.value[aiMessageIndex].meta = { status: 'done' }
                currentMessages.value = [...currentMessages.value]
              } else if (data.startsWith('{')) {
                // 尝试解析 JSON（如 sessionId）
                try {
                  const parsed = JSON.parse(data)
                  if (parsed.sessionId) {
                    const newSid = String(parsed.sessionId)
                    console.log('[SSE] Session ID:', newSid)
                    if (tempSession.value) {
                      sessions.value[newSid] = {
                        id: newSid,
                        name: '新会话',
                        messages: [...currentMessages.value]
                      }
                      currentSessionId.value = newSid
                      tempSession.value = false
                    }
                  }
                } catch (e) {
                  // 不是 JSON，当作普通文本处理
                  currentMessages.value[aiMessageIndex].content += data
                  console.log('[SSE] Content updated:', currentMessages.value[aiMessageIndex].content.length)
                }
              } else {
                // 普通文本数据，直接追加
                // 使用数组索引直接更新，强制 Vue 响应式系统检测变化
                currentMessages.value[aiMessageIndex].content += data
                console.log('[SSE] Content updated:', currentMessages.value[aiMessageIndex].content.length)
              }

              // 每收到一条数据就立即更新 DOM
              // 强制更新整个数组以触发响应式
              currentMessages.value = [...currentMessages.value]
              
              // 使用 requestAnimationFrame 强制浏览器重排
              await new Promise(resolve => {
                requestAnimationFrame(() => {
                  scrollToBottom()
                  resolve()
                })
              })
            }
          }
        }

        // 流读取完成后的处理
        loading.value = false
        currentMessages.value[aiMessageIndex].meta = { status: 'done' }
        currentMessages.value = [...currentMessages.value]

        // 同步到 sessions 存储
        if (!tempSession.value && currentSessionId.value && sessions.value[currentSessionId.value]) {
          const sessMsgs = sessions.value[currentSessionId.value].messages
          if (Array.isArray(sessMsgs) && sessMsgs.length) {
            const lastIndex = sessMsgs.length - 1
            if (sessMsgs[lastIndex] && sessMsgs[lastIndex].role === 'assistant') {
              sessMsgs[lastIndex].content = currentMessages.value[aiMessageIndex].content
            }
          }
        }
      } catch (err) {
        console.error('Stream error:', err)
        loading.value = false
        currentMessages.value[aiMessageIndex].meta = { status: 'error' }
        currentMessages.value = [...currentMessages.value]
        ElMessage.error('流式传输出错')
      }
    }


    async function handleNormal(question) {
      if (tempSession.value) {

        const response = await api.post('/AI/chat/send-new-session', {
          question: question
        })
        if (response.data && response.data.status_code === 1000) {
          const sessionId = String(response.data.sessionId)
          const aiMessage = {
            role: 'assistant',
            content: response.data.Information || ''
          }

          sessions.value[sessionId] = {
            id: sessionId,
            name: '新会话',
            messages: [ { role: 'user', content: question }, aiMessage ]
          }
          currentSessionId.value = sessionId
          tempSession.value = false
          currentMessages.value = [...sessions.value[sessionId].messages]
        } else {
          ElMessage.error(response.data?.status_msg || '发送失败')

          currentMessages.value.pop()
        }
      } else {

        const sessionMsgs = sessions.value[currentSessionId.value].messages

        sessionMsgs.push({ role: 'user', content: question })

        const response = await api.post('/AI/chat/send', {
          question: question,
          sessionId: currentSessionId.value
        })
        if (response.data && response.data.status_code === 1000) {
          const aiMessage = { role: 'assistant', content: response.data.Information || '' }
          sessionMsgs.push(aiMessage)
          currentMessages.value = [...sessionMsgs]
        } else {
          ElMessage.error(response.data?.status_msg || '发送失败')
          sessionMsgs.pop() // rollback
          currentMessages.value.pop()
        }
      }
    }


    const scrollToBottom = () => {
      if (messagesRef.value) {
        try {
          messagesRef.value.scrollTop = messagesRef.value.scrollHeight
        } catch (e) {
          // ignore
        }
      }
    }

    onMounted(() => {
      loadSessions()
    })

    // expose to template
    return {
      sessions: computed(() => Object.values(sessions.value)),
      currentSessionId,
      tempSession,
      currentMessages,
      inputMessage,
      loading,
      messagesRef,
      messageInput,
      isStreaming,
      renderMarkdown,
      createNewSession,
      switchSession,
      syncHistory,
      sendMessage,
      deleteSession
    }
  }
}
</script>

<style scoped>
.ai-chat-container {
  height: 100vh;
  display: flex;
}

/* ── Sidebar ── */
.sessions-list {
  width: 272px;
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--c-surface);
  border-right: 1px solid var(--c-line);
  box-shadow: var(--shadow-soft);
  position: relative;
  z-index: 2;
  flex-shrink: 0;
}

.sessions-list-header {
  padding: 28px 24px 24px;
  position: relative;
  border-bottom: 1px solid var(--c-line);
}

.header-accent {
  position: absolute;
  top: 28px;
  left: 24px;
  width: 32px;
  height: 3px;
  background: linear-gradient(90deg, var(--c-accent), var(--c-flow-2));
  animation: geoAccentShimmer 3s ease-in-out infinite;
}

.header-label {
  display: block;
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.18em;
  color: var(--c-ink-faint);
  margin-top: 12px;
  margin-bottom: 4px;
}

.header-title {
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin: 0 0 20px;
  color: var(--c-ink);
}

.new-chat-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 11px 0;
  cursor: pointer;
  background: var(--c-ink);
  color: var(--c-surface-elevated);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.04em;
  box-shadow: var(--shadow-soft);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  position: relative;
  overflow: hidden;
}

.new-chat-btn::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(105deg, transparent 40%, rgba(255,255,255,0.12) 50%, transparent 60%);
  transform: translateX(-100%);
  animation: geoBtnShine 4s ease-in-out infinite;
}

.new-chat-btn:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-medium);
}

.btn-icon {
  font-size: 16px;
  font-weight: 300;
  line-height: 1;
}

.sessions-list-ul {
  list-style: none;
  padding: 8px 0;
  margin: 0;
  flex: 1;
  overflow-y: auto;
}

.sessions-item {
  padding: 12px 24px;
  cursor: pointer;
  transition: background 0.15s ease;
  color: var(--c-ink-muted);
  display: flex;
  align-items: center;
  gap: 10px;
  position: relative;
  border-left: 2px solid transparent;
}

.session-indicator {
  width: 6px;
  height: 6px;
  border: 1px solid var(--c-line-strong);
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.session-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.delete-session-btn {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--c-ink-faint);
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0;
  opacity: 0;
  transition: all 0.15s ease;
}

.sessions-item:hover .delete-session-btn {
  opacity: 1;
}

.delete-session-btn:hover {
  border-color: rgba(239, 68, 68, 0.3);
  color: #ef4444;
  background: rgba(239, 68, 68, 0.06);
}

.sessions-item:hover {
  background: rgba(26, 26, 26, 0.02);
}

.sessions-item.active {
  background: var(--c-accent-soft);
  color: var(--c-ink);
  font-weight: 500;
  border-left-color: var(--c-accent);
}

.sessions-item.active .session-indicator {
  background: var(--c-accent);
  border-color: var(--c-accent);
  box-shadow: 0 0 8px rgba(79, 70, 229, 0.4);
}

.sessions-item.active .delete-session-btn {
  opacity: 1;
  color: var(--c-ink-muted);
}

/* ── Chat Section ── */
.chat-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  z-index: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.top-bar {
  background: var(--c-surface-elevated);
  display: flex;
  align-items: center;
  padding: 14px 28px;
  box-shadow: var(--shadow-soft);
  border-bottom: 1px solid var(--c-line);
  gap: 16px;
  flex-shrink: 0;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: transparent;
  border: 1px solid var(--c-line);
  color: var(--c-ink-muted);
  padding: 7px 14px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.15s ease;
}

.back-btn:hover {
  border-color: var(--c-line-strong);
  color: var(--c-ink);
  background: rgba(26, 26, 26, 0.02);
}

.back-arrow {
  font-size: 14px;
}

.top-divider {
  width: 1px;
  height: 20px;
  background: var(--c-line);
  transform: rotate(8deg);
}

.sync-btn {
  background: transparent;
  color: var(--c-ink);
  padding: 7px 14px;
  border: 1px solid var(--c-line-strong);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.02em;
  transition: all 0.15s ease;
}

.sync-btn:hover:not(:disabled) {
  border-color: var(--c-accent);
  color: var(--c-accent);
  box-shadow: var(--shadow-glow);
}

.sync-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.stream-toggle {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
}

.stream-toggle input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-track {
  width: 36px;
  height: 20px;
  border: 1px solid var(--c-line-strong);
  border-radius: var(--radius-sm);
  position: relative;
  transition: all 0.2s ease;
}

.toggle-thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 12px;
  height: 12px;
  background: var(--c-ink-faint);
  border-radius: 1px;
  transition: all 0.2s ease;
}

.stream-toggle input:checked + .toggle-track {
  border-color: var(--c-accent);
  background: var(--c-accent-soft);
}

.stream-toggle input:checked + .toggle-track .toggle-thumb {
  left: 19px;
  background: var(--c-accent);
  box-shadow: 0 0 8px rgba(79, 70, 229, 0.5);
}

.toggle-label {
  font-size: 12px;
  color: var(--c-ink-muted);
  letter-spacing: 0.02em;
}

/* ── Messages ── */
.chat-messages {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 32px 40px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  opacity: 0.7;
}

.empty-geo {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.empty-geo span {
  display: block;
  border: 1px solid var(--c-line-strong);
}

.empty-geo span:nth-child(1) {
  width: 24px;
  height: 24px;
  transform: rotate(12deg);
}

.empty-geo span:nth-child(2) {
  width: 16px;
  height: 32px;
  transform: translateY(-4px);
}

.empty-geo span:nth-child(3) {
  width: 20px;
  height: 20px;
  transform: rotate(-6deg) translateY(2px);
}

.empty-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--c-ink);
  margin: 0;
}

.empty-desc {
  font-size: 13px;
  color: var(--c-ink-faint);
  margin: 0;
}

.message {
  display: flex;
  gap: 14px;
  max-width: 78%;
  animation: messageIn 0.3s ease-out;
}

@keyframes messageIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.user-message {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.message-avatar {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.05em;
  border-radius: var(--radius-sm);
  border: 1px solid var(--c-line);
}

.user-message .message-avatar {
  background: var(--c-ink);
  color: var(--c-surface-elevated);
  border-color: var(--c-ink);
}

.ai-message .message-avatar {
  background: var(--c-surface-elevated);
  color: var(--c-accent);
  border-color: var(--c-accent);
  box-shadow: var(--shadow-glow);
}

.message-body {
  flex: 1;
  min-width: 0;
  padding: 14px 18px;
  position: relative;
}

.user-message .message-body {
  background: var(--c-ink);
  color: var(--c-surface-elevated);
  border-radius: var(--radius-md) var(--radius-sm) var(--radius-md) var(--radius-md);
  box-shadow: var(--shadow-medium);
}

.ai-message .message-body {
  background: var(--c-surface-elevated);
  color: var(--c-ink);
  border: 1px solid var(--c-line);
  border-radius: var(--radius-sm) var(--radius-md) var(--radius-md) var(--radius-md);
  box-shadow: var(--shadow-soft);
}

.ai-message .message-body::before {
  content: '';
  position: absolute;
  top: -1px;
  left: 20px;
  width: 40px;
  height: 2px;
  background: linear-gradient(90deg, var(--c-accent), var(--c-flow-2), transparent);
  animation: geoFlowPulse 4s ease-in-out infinite;
}

.message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.message-role {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  opacity: 0.6;
}

.streaming-indicator {
  display: flex;
  gap: 3px;
  align-items: center;
}

.streaming-indicator span {
  width: 4px;
  height: 4px;
  background: var(--c-accent);
  border-radius: 1px;
  animation: dotPulse 1.2s ease-in-out infinite;
}

.streaming-indicator span:nth-child(2) { animation-delay: 0.2s; }
.streaming-indicator span:nth-child(3) { animation-delay: 0.4s; }

@keyframes dotPulse {
  0%, 100% { opacity: 0.3; transform: scale(0.8); }
  50% { opacity: 1; transform: scale(1); }
}

.message-content {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 14px;
  line-height: 1.7;
}

.message-content :deep(code) {
  font-family: var(--font-mono);
  font-size: 12px;
  padding: 2px 6px;
  background: rgba(26, 26, 26, 0.06);
  border-radius: var(--radius-sm);
  border: 1px solid var(--c-line);
}

.user-message .message-content :deep(code) {
  background: rgba(255, 255, 255, 0.12);
  border-color: rgba(255, 255, 255, 0.15);
}

/* ── Input Area ── */
.chat-input {
  padding: 20px 40px 28px;
  background: var(--c-surface-elevated);
  border-top: 1px solid var(--c-line);
  flex-shrink: 0;
}

.input-frame {
  position: relative;
  display: flex;
  align-items: flex-end;
  gap: 12px;
  padding: 4px;
  border: 1px solid var(--c-line);
  border-radius: var(--radius-md);
  background: var(--c-surface);
  box-shadow: var(--shadow-soft);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.input-frame:focus-within {
  border-color: var(--c-accent);
  box-shadow: var(--shadow-soft), var(--shadow-glow);
}

.input-glow {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--c-flow-1), var(--c-flow-2), var(--c-flow-3), transparent);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.input-frame:focus-within .input-glow {
  opacity: 0.8;
  animation: glowSlide 2s ease-in-out infinite;
}

@keyframes glowSlide {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}

.chat-input textarea {
  flex: 1;
  resize: none;
  border: none;
  border-radius: var(--radius-sm);
  padding: 12px 14px;
  font-size: 14px;
  outline: none;
  background: transparent;
  color: var(--c-ink);
  min-height: 20px;
  max-height: 160px;
  line-height: 1.5;
  font-family: var(--font-sans);
}

.chat-input textarea::placeholder {
  color: var(--c-ink-faint);
}

.chat-input textarea:disabled {
  opacity: 0.5;
}

.send-btn {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  margin: 4px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--c-ink);
  color: var(--c-surface-elevated);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.04em;
  cursor: pointer;
  box-shadow: var(--shadow-soft);
  transition: all 0.15s ease;
  position: relative;
  overflow: hidden;
}

.send-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(105deg, transparent 40%, rgba(255,255,255,0.1) 50%, transparent 60%);
  transform: translateX(-100%);
}

.send-btn:hover:not(:disabled)::before {
  animation: geoBtnShine 0.6s ease forwards;
}

.send-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: var(--shadow-medium);
}

.send-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.send-loading {
  width: 12px;
  height: 12px;
  border: 1.5px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ── Scrollbar ── */
.sessions-list-ul::-webkit-scrollbar,
.chat-messages::-webkit-scrollbar {
  width: 4px;
}

.sessions-list-ul::-webkit-scrollbar-thumb,
.chat-messages::-webkit-scrollbar-thumb {
  background: var(--c-line-strong);
  border-radius: 2px;
}

.sessions-list-ul::-webkit-scrollbar-track,
.chat-messages::-webkit-scrollbar-track {
  background: transparent;
}

/* ── Responsive ── */
@media (max-width: 768px) {
  .sessions-list {
    width: 220px;
  }

  .chat-messages {
    padding: 20px 16px;
  }

  .chat-input {
    padding: 16px;
  }

  .message {
    max-width: 92%;
  }

  .top-bar {
    padding: 12px 16px;
    flex-wrap: wrap;
  }
}
</style>
