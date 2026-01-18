<template>
  <div class="diagnostics-view">
    <div class="view-header">
      <h2>Диагностика системы</h2>
      <div class="header-actions">
        <el-select v-model="selectedLevel" @change="runDiagnostics" :disabled="loading">
          <el-option label="Быстрая" value="quick" />
          <el-option label="Полная" value="full" />
          <el-option label="Глубокая" value="deep" />
        </el-select>
        <el-button @click="runDiagnostics" :loading="loading" type="primary">
          <el-icon><Refresh /></el-icon>
          Запустить
        </el-button>
      </div>
    </div>

    <!-- Results -->
    <el-card v-if="results" class="results-card">
      <template #header>
        <div class="results-header">
          <span>Результаты диагностики</span>
          <el-tag 
            :type="overallStatusType" 
            size="large"
          >
            {{ results.overall }}
          </el-tag>
        </div>
      </template>
      
      <!-- Summary -->
      <div class="diagnostic-summary">
        <el-row :gutter="20">
          <el-col :span="6">
            <el-statistic title="Уровень" :value="results.level" />
          </el-col>
          <el-col :span="6">
            <el-statistic title="Длительность" :value="results.duration" />
          </el-col>
          <el-col :span="6">
            <el-statistic title="Всего тестов" :value="results.totalTests" />
          </el-col>
          <el-col :span="6">
            <el-statistic title="Статус" :value="results.overall" />
          </el-col>
        </el-row>

        <el-row :gutter="20" style="margin-top: 20px;">
          <el-col :span="8">
            <el-statistic title="Пройдено" :value="results.passed">
              <template #suffix>
                <span style="color: #10b981">✓</span>
              </template>
            </el-statistic>
          </el-col>
          <el-col :span="8">
            <el-statistic title="Предупреждений" :value="results.warnings">
              <template #suffix>
                <span style="color: #f59e0b">⚠</span>
              </template>
            </el-statistic>
          </el-col>
          <el-col :span="8">
            <el-statistic title="Ошибок" :value="results.failed">
              <template #suffix>
                <span style="color: #ef4444">✗</span>
              </template>
            </el-statistic>
          </el-col>
        </el-row>
      </div>

      <!-- Environment Info -->
      <el-divider content-position="left">Информация об окружении</el-divider>
      <div class="environment-info">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="ОС">
            {{ results.environment.os }}
          </el-descriptions-item>
          <el-descriptions-item label="Архитектура">
            {{ results.environment.architecture }}
          </el-descriptions-item>
          <el-descriptions-item label="Пользователь">
            {{ results.environment.user }}
          </el-descriptions-item>
          <el-descriptions-item label="Компьютер">
            {{ results.environment.computer }}
          </el-descriptions-item>
          <el-descriptions-item label="Права админа">
            <el-tag :type="results.environment.isAdmin === 'true' ? 'success' : 'danger'" size="small">
              {{ results.environment.isAdmin === 'true' ? 'Да' : 'Нет' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Серверная ОС">
            <el-tag :type="results.environment.isServer === 'true' ? 'info' : ''" size="small">
              {{ results.environment.isServer === 'true' ? 'Да' : 'Нет' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="CPU ядер" :span="2">
            {{ results.environment.cpuCount }}
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <!-- Test Results -->
      <el-divider content-position="left">Детальные результаты</el-divider>
      <div class="test-results">
        <div 
          v-for="result in results.results" 
          :key="result.test"
          class="test-result"
          :class="result.status.toLowerCase()"
        >
          <div class="test-header">
            <div class="test-info">
              <el-icon 
                :class="getTestIcon(result.status)"
                :color="getTestColor(result.status)"
              >
                <component :is="getTestIcon(result.status)" />
              </el-icon>
              <span class="test-name">{{ result.test }}</span>
            </div>
            <div class="test-meta">
              <el-tag 
                :type="getTestTagType(result.status)" 
                size="small"
              >
                {{ result.status }}
              </el-tag>
              <span class="test-duration">{{ result.duration }}</span>
            </div>
          </div>
          <div class="test-message">{{ result.message }}</div>
          <div v-if="result.details" class="test-details">
            <el-collapse>
              <el-collapse-item title="Детали">
                <pre>{{ JSON.stringify(result.details, null, 2) }}</pre>
              </el-collapse-item>
            </el-collapse>
          </div>
        </div>
      </div>

      <!-- Recommendations -->
      <el-divider content-position="left">Рекомендации</el-divider>
      <div class="recommendations">
        <el-alert
          :title="getRecommendationTitle(results.overall)"
          :type="getRecommendationType(results.overall)"
          :closable="false"
          show-icon
        >
          {{ getRecommendationText(results.overall) }}
        </el-alert>
      </div>
    </el-card>

    <!-- Loading State -->
    <el-card v-else-if="loading" class="loading-card">
      <div class="loading-content">
        <el-icon class="loading-icon"><Loading /></el-icon>
        <p>Выполняется диагностика системы...</p>
        <el-progress :percentage="loadingProgress" :show-text="false" />
      </div>
    </el-card>

    <!-- Empty State -->
    <el-empty 
      v-else
      description="Нажмите 'Запустить' для начала диагностики"
      :image-size="200"
    >
      <el-button type="primary" @click="runDiagnostics" :loading="loading">
        <el-icon><Refresh /></el-icon>
        Запустить диагностику
      </el-button>
    </el-empty>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { 
  Refresh, 
  Loading, 
  CircleCheck, 
  CircleClose, 
  Warning 
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  diagnosticResults: {
    type: Object,
    default: null
  },
  loading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['run-diagnostics'])

const selectedLevel = ref('quick')
const loadingProgress = ref(0)

const results = computed(() => props.diagnosticResults)

const overallStatusType = computed(() => {
  if (!results.value) return 'info'
  switch (results.value.overall) {
    case 'HEALTHY': return 'success'
    case 'WARNING': return 'warning'
    case 'CRITICAL': return 'danger'
    default: return 'info'
  }
})

const getTestIcon = (status) => {
  switch (status) {
    case 'PASS': return CircleCheck
    case 'FAIL': return CircleClose
    case 'WARN': return Warning
    default: return CircleCheck
  }
}

const getTestColor = (status) => {
  switch (status) {
    case 'PASS': return '#10b981'
    case 'FAIL': return '#ef4444'
    case 'WARN': return '#f59e0b'
    default: return '#6b7280'
  }
}

const getTestTagType = (status) => {
  switch (status) {
    case 'PASS': return 'success'
    case 'FAIL': return 'danger'
    case 'WARN': return 'warning'
    default: return 'info'
  }
}

const getRecommendationTitle = (overall) => {
  switch (overall) {
    case 'HEALTHY': return 'Система в хорошем состоянии'
    case 'WARNING': return 'Обнаружены предупреждения'
    case 'CRITICAL': return 'Обнаружены критические проблемы'
    default: return 'Статус неизвестен'
  }
}

const getRecommendationType = (overall) => {
  switch (overall) {
    case 'HEALTHY': return 'success'
    case 'WARNING': return 'warning'
    case 'CRITICAL': return 'danger'
    default: return 'info'
  }
}

const getRecommendationText = (overall) => {
  switch (overall) {
    case 'HEALTHY':
      return 'Система в хорошем состоянии, проблем не обнаружено. Рекомендуется регулярно выполнять диагностику для поддержания системы в оптимальном состоянии.'
    case 'WARNING':
      return 'Обнаружены предупреждения. Рекомендуется проверить систему и устранить выявленные проблемы для предотвращения возможных сбоев в будущем.'
    case 'CRITICAL':
      return 'Обнаружены критические проблемы. Требуется немедленное вмешательство для восстановления работоспособности и безопасности системы.'
    default:
      return 'Статус системы не определен. Рекомендуется повторить диагностику.'
  }
}

const runDiagnostics = () => {
  emit('run-diagnostics', selectedLevel.value)
}

// Simulate loading progress
let progressInterval = null

const startProgressSimulation = () => {
  loadingProgress.value = 0
  progressInterval = setInterval(() => {
    loadingProgress.value += Math.random() * 15
    if (loadingProgress.value >= 90) {
      loadingProgress.value = 90
      clearInterval(progressInterval)
    }
  }, 500)
}

const stopProgressSimulation = () => {
  if (progressInterval) {
    clearInterval(progressInterval)
    progressInterval = null
  }
  loadingProgress.value = 100
}

// Watch for loading state
const { watch } = require('vue')
watch(() => props.loading, (loading) => {
  if (loading) {
    startProgressSimulation()
  } else {
    stopProgressSimulation()
    setTimeout(() => {
      loadingProgress.value = 0
    }, 1000)
  }
})
</script>

<style scoped>
.diagnostics-view {
  padding: 20px;
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.view-header h2 {
  margin: 0;
  color: #ffffff;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

.results-card,
.loading-card {
  background-color: #1e293b;
  border: 1px solid #334155;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.diagnostic-summary {
  margin-bottom: 20px;
}

.environment-info {
  margin-bottom: 20px;
}

.test-results {
  margin-bottom: 20px;
}

.test-result {
  padding: 15px;
  border-radius: 8px;
  margin-bottom: 15px;
  border-left: 4px solid;
}

.test-result.pass {
  background-color: #064e3b;
  border-left-color: #10b981;
}

.test-result.fail {
  background-color: #7f1d1d;
  border-left-color: #ef4444;
}

.test-result.warn {
  background-color: #78350f;
  border-left-color: #f59e0b;
}

.test-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.test-info {
  display: flex;
  align-items: center;
  gap: 10px;
}

.test-name {
  font-weight: 600;
  color: #ffffff;
  font-size: 16px;
}

.test-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.test-duration {
  color: #94a3b8;
  font-size: 14px;
}

.test-message {
  color: #e2e8f0;
  margin-bottom: 10px;
  line-height: 1.4;
}

.test-details {
  margin-top: 10px;
}

.test-details pre {
  background-color: #0f172a;
  padding: 10px;
  border-radius: 4px;
  font-size: 12px;
  color: #94a3b8;
  overflow-x: auto;
}

.recommendations {
  margin-top: 20px;
}

.loading-content {
  text-align: center;
  padding: 40px;
}

.loading-icon {
  font-size: 48px;
  color: #3b82f6;
  margin-bottom: 20px;
  animation: spin 2s linear infinite;
}

.loading-content p {
  color: #ffffff;
  margin-bottom: 20px;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

:deep(.el-card__header) {
  background-color: #0f172a;
  border-bottom: 1px solid #334155;
  color: #ffffff;
}

:deep(.el-card__body) {
  background-color: #1e293b;
  color: #ffffff;
}

:deep(.el-descriptions__label) {
  color: #94a3b8;
}

:deep(.el-descriptions__content) {
  color: #ffffff;
}

:deep(.el-divider__text) {
  color: #ffffff;
  background-color: #1e293b;
}

:deep(.el-empty__description) {
  color: #94a3b8;
}

:deep(.el-select .el-input__inner) {
  background-color: #0f172a;
  border-color: #334155;
  color: #ffffff;
}

:deep(.el-select-dropdown) {
  background-color: #1e293b;
  border-color: #334155;
}

:deep(.el-select-dropdown__item) {
  color: #ffffff;
}

:deep(.el-select-dropdown__item:hover) {
  background-color: #334155;
}

:deep(.el-alert) {
  background-color: #1e293b;
  border-color: #334155;
}

:deep(.el-alert__title) {
  color: #ffffff;
}

:deep(.el-alert__description) {
  color: #94a3b8;
}

:deep(.el-collapse) {
  border-color: #334155;
}

:deep(.el-collapse-item__header) {
  color: #ffffff;
  background-color: #0f172a;
  border-color: #334155;
}

:deep(.el-collapse-item__content) {
  color: #ffffff;
  background-color: #1e293b;
}

:deep(.el-statistic__head) {
  color: #94a3b8;
}

:deep(.el-statistic__content) {
  color: #ffffff;
}
</style>
