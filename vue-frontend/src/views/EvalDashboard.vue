<template>
  <div class="eval-page geo-page">
    <GeoBackground />

    <aside class="eval-sidebar">
      <div class="sidebar-header">
        <div class="header-accent"></div>
        <span class="header-label">EVALUATION</span>
        <h2 class="header-title">Agent 测评</h2>
        <button class="back-btn" @click="$router.push('/menu')">← 返回</button>
      </div>

      <section class="dataset-section">
        <h3 class="section-title">测评集</h3>
        <div v-if="loadingDatasets" class="loading-hint">加载中...</div>
        <ul v-else class="dataset-list">
          <li
            v-for="ds in datasets"
            :key="ds.id"
            :class="['dataset-item', { active: selectedDataset?.id === ds.id }]"
            @click="selectDataset(ds)"
          >
            <span class="dataset-name">{{ ds.name }}</span>
            <span class="dataset-meta">{{ ds.case_count }} 条用例 · v{{ ds.version }}</span>
          </li>
        </ul>
        <button
          class="run-btn"
          :disabled="!selectedDataset || startingRun"
          @click="startRun"
        >
          {{ startingRun ? '启动中...' : '开始测评' }}
        </button>
      </section>

      <section class="runs-section">
        <div class="section-head">
          <h3 class="section-title">历史记录</h3>
          <button class="refresh-btn" @click="loadRuns" :disabled="loadingRuns">刷新</button>
        </div>
        <ul class="runs-list">
          <li
            v-for="run in runs"
            :key="run.id"
            :class="['run-item', { active: selectedRun?.id === run.id }]"
            @click="selectRun(run)"
          >
            <div class="run-top">
              <span class="run-name">{{ run.dataset_name }}</span>
              <span :class="['run-status', run.status]">{{ statusLabel(run.status) }}</span>
            </div>
            <div class="run-meta">
              <span>{{ formatTime(run.created_at) }}</span>
              <span v-if="run.status === 'completed'">
                {{ run.passed_cases }}/{{ run.total_cases }} 通过
              </span>
            </div>
          </li>
        </ul>
      </section>
    </aside>

    <main class="eval-main">
      <header class="main-header">
        <div v-if="selectedRun">
          <span class="main-label">RUN DETAIL</span>
          <h1 class="main-title">{{ selectedRun.dataset_name }}</h1>
          <p class="main-sub">
            模型 {{ selectedRun.model_type || '—' }} ·
            {{ statusLabel(selectedRun.status) }}
          </p>
        </div>
        <div v-else-if="selectedDataset">
          <span class="main-label">DATASET</span>
          <h1 class="main-title">{{ selectedDataset.name }}</h1>
          <p class="main-sub">{{ selectedDataset.description }}</p>
        </div>
        <div v-else class="empty-header">
          <span class="main-label">OVERVIEW</span>
          <h1 class="main-title">选择测评集或历史记录</h1>
        </div>
      </header>

      <div v-if="selectedRun && selectedRun.status === 'completed'" class="metrics-row">
        <article class="metric-card">
          <span class="metric-label">通过率</span>
          <strong class="metric-value">
            {{ passRate(selectedRun) }}%
          </strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">平均得分</span>
          <strong class="metric-value">{{ (selectedRun.avg_score * 100).toFixed(1) }}%</strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">平均延迟</span>
          <strong class="metric-value">{{ selectedRun.avg_latency_ms }} ms</strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">用例数</span>
          <strong class="metric-value">{{ selectedRun.total_cases }}</strong>
        </article>
      </div>

      <div v-if="selectedRun?.status === 'completed' && selectedRun.tool_cases" class="metrics-row">
        <article class="metric-card">
          <span class="metric-label">Tool 精确匹配率</span>
          <strong class="metric-value">{{ percent(selectedRun.tool_exact_match_rate) }}</strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">Tool Call F1</span>
          <strong class="metric-value">{{ percent(selectedRun.tool_call_f1) }}</strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">参数准确率</span>
          <strong class="metric-value">{{ percent(selectedRun.tool_argument_accuracy) }}</strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">Tool P / R</span>
          <strong class="metric-value small">{{ percent(selectedRun.tool_call_precision) }} / {{ percent(selectedRun.tool_call_recall) }}</strong>
        </article>
      </div>

      <div v-if="selectedRun?.status === 'completed' && selectedRun.rag_cases" class="metrics-row">
        <article class="metric-card">
          <span class="metric-label">RAG Hit Rate</span>
          <strong class="metric-value">{{ percent(selectedRun.rag_hit_rate) }}</strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">Recall@K</span>
          <strong class="metric-value">{{ percent(selectedRun.rag_recall_at_k) }}</strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">Precision@K</span>
          <strong class="metric-value">{{ percent(selectedRun.rag_precision_at_k) }}</strong>
        </article>
        <article class="metric-card">
          <span class="metric-label">MRR</span>
          <strong class="metric-value">{{ selectedRun.rag_mrr.toFixed(3) }}</strong>
        </article>
      </div>

      <div v-if="selectedRun?.status === 'running' || selectedRun?.status === 'pending'" class="running-banner">
        <span class="pulse-dot"></span>
        测评运行中，请稍候...
        <button class="refresh-btn inline" @click="refreshSelectedRun">刷新状态</button>
      </div>

      <div v-if="selectedRun?.status === 'failed'" class="error-banner">
        测评失败：{{ selectedRun.error_message || '未知错误' }}
      </div>

      <section v-if="selectedDataset && !selectedRun" class="cases-preview">
        <h3 class="section-title">用例预览</h3>
        <div v-if="loadingCases" class="loading-hint">加载用例...</div>
        <div v-else class="cases-grid">
          <article v-for="c in cases" :key="c.id" class="case-card">
            <span class="case-tag">{{ c.category || 'answer' }} · {{ c.score_type || 'none' }}</span>
            <h4 class="case-name">{{ c.name }}</h4>
            <p class="case-input">{{ c.input }}</p>
          </article>
        </div>
      </section>

      <section v-if="selectedRun" class="results-section">
        <h3 class="section-title">用例结果</h3>
        <div v-if="loadingResults" class="loading-hint">加载结果...</div>
        <div v-else-if="results.length === 0" class="loading-hint">暂无结果</div>
        <div v-else class="results-list">
          <article
            v-for="r in results"
            :key="r.id"
            :class="['result-card', r.passed ? 'passed' : 'failed']"
          >
            <div class="result-head">
              <h4 class="result-name">{{ r.case_name }}</h4>
              <span class="case-tag">{{ r.category || 'answer' }}</span>
              <span :class="['result-badge', r.passed ? 'pass' : 'fail']">
                {{ r.passed ? '通过' : '失败' }}
              </span>
              <span class="result-latency">{{ r.latency_ms }} ms</span>
            </div>
            <div class="result-block">
              <span class="block-label">输入</span>
              <p>{{ r.input }}</p>
            </div>
            <div v-if="r.metrics?.tool" class="result-block">
              <span class="block-label">Tool Calling 指标</span>
              <p>
                F1 {{ percent(r.metrics.tool.call_f1) }} · 参数 {{ percent(r.metrics.tool.argument_accuracy) }} ·
                顺序 {{ percent(r.metrics.tool.order_accuracy) }} · 实际 {{ r.metrics.tool.actual_calls }} 次
              </p>
              <p v-for="(call, index) in r.tool_calls || []" :key="`${r.id}-tool-${index}`">
                #{{ index + 1 }} {{ call.name }} {{ call.arguments }}
              </p>
            </div>
            <div v-if="r.metrics?.rag" class="result-block">
              <span class="block-label">RAG 指标</span>
              <p>
                Recall@{{ r.metrics.rag.k }} {{ percent(r.metrics.rag.recall_at_k) }} ·
                Precision@{{ r.metrics.rag.k }} {{ percent(r.metrics.rag.precision_at_k) }} ·
                MRR {{ r.metrics.rag.mrr.toFixed(3) }}
              </p>
              <p v-for="hit in r.retrieval_hits || []" :key="`${r.id}-hit-${hit.rank}`">
                #{{ hit.rank }} {{ hit.source }}#{{ hit.chunk_id }} · {{ hit.score.toFixed(3) }}
              </p>
            </div>
            <div v-if="r.retrieval_error" class="result-block error">
              <span class="block-label">检索错误</span>
              <p>{{ r.retrieval_error }}</p>
            </div>
            <div class="result-block">
              <span class="block-label">实际输出</span>
              <p>{{ r.actual_output || '—' }}</p>
            </div>
            <div v-if="r.error_message" class="result-block error">
              <span class="block-label">错误</span>
              <p>{{ r.error_message }}</p>
            </div>
          </article>
        </div>
      </section>
    </main>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../utils/api'
import GeoBackground from '../components/GeoBackground.vue'

export default {
  name: 'EvalDashboard',
  components: { GeoBackground },
  setup() {
    const datasets = ref([])
    const cases = ref([])
    const runs = ref([])
    const results = ref([])
    const selectedDataset = ref(null)
    const selectedRun = ref(null)
    const loadingDatasets = ref(false)
    const loadingCases = ref(false)
    const loadingRuns = ref(false)
    const loadingResults = ref(false)
    const startingRun = ref(false)
    let pollTimer = null

    const statusLabel = (status) => {
      const map = {
        pending: '等待中',
        running: '运行中',
        completed: '已完成',
        failed: '失败'
      }
      return map[status] || status
    }

    const formatTime = (t) => {
      if (!t) return ''
      return new Date(t).toLocaleString('zh-CN')
    }

    const passRate = (run) => {
      if (!run.total_cases) return '0.0'
      return ((run.passed_cases / run.total_cases) * 100).toFixed(1)
    }

    const percent = (value) => `${((value || 0) * 100).toFixed(1)}%`

    const loadDatasets = async () => {
      loadingDatasets.value = true
      try {
        const { data } = await api.get('/AI/eval/datasets')
        if (data.status_code === 1000) {
          datasets.value = data.datasets || []
          if (datasets.value.length && !selectedDataset.value) {
            selectDataset(datasets.value[0])
          }
        }
      } catch (e) {
        ElMessage.error('加载测评集失败')
      } finally {
        loadingDatasets.value = false
      }
    }

    const loadCases = async (datasetId) => {
      loadingCases.value = true
      cases.value = []
      try {
        const { data } = await api.get(`/AI/eval/datasets/${datasetId}/cases`)
        if (data.status_code === 1000) {
          cases.value = data.cases || []
        }
      } catch (e) {
        ElMessage.error('加载用例失败')
      } finally {
        loadingCases.value = false
      }
    }

    const loadRuns = async () => {
      loadingRuns.value = true
      try {
        const { data } = await api.get('/AI/eval/runs')
        if (data.status_code === 1000) {
          runs.value = data.runs || []
        }
      } catch (e) {
        ElMessage.error('加载历史记录失败')
      } finally {
        loadingRuns.value = false
      }
    }

    const loadResults = async (runId) => {
      loadingResults.value = true
      results.value = []
      try {
        const { data } = await api.get(`/AI/eval/runs/${runId}/results`)
        if (data.status_code === 1000) {
          results.value = data.results || []
        }
      } catch (e) {
        ElMessage.error('加载结果失败')
      } finally {
        loadingResults.value = false
      }
    }

    const fetchRunDetail = async (runId) => {
      const { data } = await api.get(`/AI/eval/runs/${runId}`)
      if (data.status_code === 1000) {
        selectedRun.value = data.run
        return data.run
      }
      return null
    }

    const selectDataset = (ds) => {
      selectedDataset.value = ds
      selectedRun.value = null
      results.value = []
      loadCases(ds.id)
    }

    const selectRun = async (run) => {
      selectedRun.value = run
      await fetchRunDetail(run.id)
      if (selectedRun.value?.status === 'completed') {
        await loadResults(run.id)
      } else {
        results.value = []
      }
      setupPolling()
    }

    const refreshSelectedRun = async () => {
      if (!selectedRun.value) return
      const run = await fetchRunDetail(selectedRun.value.id)
      if (run?.status === 'completed') {
        await loadResults(run.id)
        stopPolling()
        await loadRuns()
      } else if (run?.status === 'failed') {
        stopPolling()
        await loadRuns()
      }
    }

    const startRun = async () => {
      if (!selectedDataset.value) return
      startingRun.value = true
      try {
        const { data } = await api.post('/AI/eval/runs', {
          dataset_id: selectedDataset.value.id
        })
        if (data.status_code === 1000) {
          ElMessage.success('测评已启动')
          await loadRuns()
          const newRun = runs.value.find(r => r.id === data.run_id)
          if (newRun) {
            await selectRun(newRun)
          }
        } else {
          ElMessage.error(data.status_msg || '启动失败')
        }
      } catch (e) {
        ElMessage.error('启动测评失败')
      } finally {
        startingRun.value = false
      }
    }

    const setupPolling = () => {
      stopPolling()
      if (!selectedRun.value) return
      if (selectedRun.value.status === 'running' || selectedRun.value.status === 'pending') {
        pollTimer = setInterval(refreshSelectedRun, 3000)
      }
    }

    const stopPolling = () => {
      if (pollTimer) {
        clearInterval(pollTimer)
        pollTimer = null
      }
    }

    onMounted(async () => {
      await Promise.all([loadDatasets(), loadRuns()])
    })

    onUnmounted(stopPolling)

    return {
      datasets,
      cases,
      runs,
      results,
      selectedDataset,
      selectedRun,
      loadingDatasets,
      loadingCases,
      loadingRuns,
      loadingResults,
      startingRun,
      statusLabel,
      formatTime,
      passRate,
      percent,
      selectDataset,
      selectRun,
      startRun,
      loadRuns,
      refreshSelectedRun
    }
  }
}
</script>

<style scoped>
.eval-page {
  display: flex;
  min-height: 100vh;
}

.eval-sidebar {
  position: relative;
  z-index: 2;
  width: 320px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 24px;
  background: var(--c-surface-elevated);
  border-right: 1px solid var(--c-line);
}

.sidebar-header .header-accent {
  width: 32px;
  height: 2px;
  background: linear-gradient(90deg, var(--c-accent), var(--c-flow-2));
  margin-bottom: 6px;
}

.header-label, .main-label, .section-title, .metric-label, .case-tag, .block-label {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.14em;
  color: var(--c-ink-faint);
}

.header-title {
  margin: 4px 0 12px;
  font-size: 18px;
  font-weight: 600;
}

.back-btn, .refresh-btn, .run-btn {
  border: 1px solid var(--c-line);
  background: var(--c-surface);
  color: var(--c-ink);
  border-radius: var(--radius-sm);
  padding: 8px 12px;
  cursor: pointer;
  font-size: 13px;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.back-btn:hover, .refresh-btn:hover, .run-btn:hover:not(:disabled) {
  border-color: var(--c-accent);
}

.run-btn {
  width: 100%;
  margin-top: 12px;
  background: var(--c-accent-soft);
  color: var(--c-accent);
  font-weight: 600;
}

.run-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.dataset-list, .runs-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.dataset-item, .run-item {
  padding: 12px;
  border: 1px solid var(--c-line);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}

.dataset-item:hover, .run-item:hover,
.dataset-item.active, .run-item.active {
  border-color: var(--c-accent);
  background: var(--c-accent-soft);
}

.dataset-name, .run-name {
  display: block;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 4px;
}

.dataset-meta, .run-meta {
  font-size: 12px;
  color: var(--c-ink-muted);
  display: flex;
  justify-content: space-between;
  gap: 8px;
}

.section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.runs-section {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.runs-list {
  overflow-y: auto;
  max-height: 360px;
}

.run-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.run-status {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid var(--c-line);
}

.run-status.completed { color: #2d8a4e; border-color: #2d8a4e; }
.run-status.running, .run-status.pending { color: var(--c-accent); border-color: var(--c-accent); }
.run-status.failed { color: #c0392b; border-color: #c0392b; }

.eval-main {
  flex: 1;
  position: relative;
  z-index: 1;
  padding: 32px 40px;
  overflow-y: auto;
}

.main-title {
  margin: 6px 0;
  font-size: 24px;
  font-weight: 600;
}

.main-sub {
  color: var(--c-ink-muted);
  font-size: 14px;
}

.metrics-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin: 24px 0;
}

.metric-card {
  padding: 16px;
  border: 1px solid var(--c-line);
  border-radius: var(--radius-md);
  background: var(--c-surface-elevated);
}

.metric-value {
  display: block;
  margin-top: 8px;
  font-size: 22px;
}

.running-banner, .error-banner {
  margin: 16px 0;
  padding: 12px 16px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  gap: 10px;
}

.running-banner {
  border: 1px solid var(--c-accent);
  background: var(--c-accent-soft);
}

.error-banner {
  border: 1px solid #c0392b;
  background: rgba(192, 57, 43, 0.08);
  color: #c0392b;
}

.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--c-accent);
  animation: pulse 1.2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.85); }
}

.refresh-btn.inline {
  margin-left: auto;
}

.cases-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
  margin-top: 12px;
}

.case-card, .result-card {
  padding: 16px;
  border: 1px solid var(--c-line);
  border-radius: var(--radius-md);
  background: var(--c-surface-elevated);
}

.case-name, .result-name {
  margin: 8px 0;
  font-size: 15px;
}

.case-input {
  font-size: 13px;
  color: var(--c-ink-muted);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 12px;
}

.result-card.passed { border-left: 3px solid #2d8a4e; }
.result-card.failed { border-left: 3px solid #c0392b; }

.result-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.result-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
}

.result-badge.pass { background: rgba(45, 138, 78, 0.15); color: #2d8a4e; }
.result-badge.fail { background: rgba(192, 57, 43, 0.12); color: #c0392b; }

.result-latency {
  margin-left: auto;
  font-size: 12px;
  color: var(--c-ink-faint);
}

.result-block p {
  margin-top: 4px;
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.result-block.error p {
  color: #c0392b;
}

.loading-hint {
  font-size: 13px;
  color: var(--c-ink-muted);
  padding: 12px 0;
}

@media (max-width: 960px) {
  .eval-page {
    flex-direction: column;
  }

  .eval-sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--c-line);
  }

  .metrics-row {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
