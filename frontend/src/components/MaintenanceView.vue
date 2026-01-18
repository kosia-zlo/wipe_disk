<template>
  <div class="maintenance-view">
    <div class="view-header">
      <h2>Обслуживание системы</h2>
      <el-button @click="refreshTasks" :loading="loading">
        <el-icon><Refresh /></el-icon>
        Обновить
      </el-button>
    </div>

    <!-- Task Selection -->
    <el-card class="tasks-card">
      <template #header>
        <span>Доступные задачи</span>
      </template>
      
      <div class="tasks-grid">
        <div 
          v-for="task in tasks" 
          :key="task.id"
          class="task-item"
          :class="{ 'selected': selectedTasks.includes(task.id) }"
          @click="toggleTask(task.id)"
        >
          <div class="task-header">
            <h4>{{ task.name }}</h4>
            <el-checkbox 
              :model-value="selectedTasks.includes(task.id)"
              @change="toggleTask(task.id)"
            />
          </div>
          <p class="task-description">{{ task.description }}</p>
          <div class="task-meta">
            <el-tag size="small" type="info">
              <el-icon><Clock /></el-icon>
              {{ task.estimate }}
            </el-tag>
          </div>
        </div>
      </div>

      <div class="task-actions">
        <el-button 
          type="primary" 
          @click="runSelectedTasks"
          :disabled="selectedTasks.length === 0"
          :loading="isRunning"
        >
          <el-icon><Play /></el-icon>
          Выполнить выбранные ({{ selectedTasks.length }})
        </el-button>
        
        <el-button @click="selectAll">
          <el-icon><Select /></el-icon>
          Выбрать все
        </el-button>
        
        <el-button @click="clearSelection">
          <el-icon><Close /></el-icon>
          Очистить выбор
        </el-button>
      </div>
    </el-card>

    <!-- Results -->
    <el-card v-if="results.length > 0" class="results-card">
      <template #header>
        <span>Результаты выполнения</span>
      </template>
      
      <div class="results-list">
        <div 
          v-for="result in results" 
          :key="result.taskId"
          class="result-item"
          :class="{ 'success': result.success, 'error': !result.success }"
        >
          <div class="result-header">
            <span class="task-name">{{ result.taskId }}</span>
            <el-tag 
              :type="result.success ? 'success' : 'danger'"
              size="small"
            >
              {{ result.success ? 'Успешно' : 'Ошибка' }}
            </el-tag>
          </div>
          <div class="result-details">
            <p class="message">{{ result.message }}</p>
            <p class="duration">Длительность: {{ result.duration }}</p>
            <p v-if="result.error" class="error">Ошибка: {{ result.error }}</p>
          </div>
        </div>
      </div>

      <div class="results-summary">
        <el-statistic title="Выполнено" :value="successCount" />
        <el-statistic title="Ошибок" :value="errorCount" />
        <el-statistic title="Всего" :value="results.length" />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Refresh, Play, Select, Close, Clock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  tasks: {
    type: Array,
    default: () => []
  },
  results: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['run-tasks', 'refresh-tasks'])

const selectedTasks = ref([])
const isRunning = ref(false)

const successCount = computed(() => {
  return props.results.filter(r => r.success).length
})

const errorCount = computed(() => {
  return props.results.filter(r => !r.success).length
})

const toggleTask = (taskId) => {
  const index = selectedTasks.value.indexOf(taskId)
  if (index > -1) {
    selectedTasks.value.splice(index, 1)
  } else {
    selectedTasks.value.push(taskId)
  }
}

const selectAll = () => {
  selectedTasks.value = props.tasks.map(task => task.id)
}

const clearSelection = () => {
  selectedTasks.value = []
}

const runSelectedTasks = async () => {
  if (selectedTasks.value.length === 0) {
    ElMessage.warning('Выберите хотя бы одну задачу')
    return
  }

  try {
    isRunning.value = true
    emit('run-tasks', selectedTasks.value)
    ElMessage.success(`Запущено ${selectedTasks.value.length} задач`)
  } catch (error) {
    ElMessage.error(`Ошибка запуска задач: ${error}`)
  } finally {
    isRunning.value = false
  }
}

const refreshTasks = () => {
  emit('refresh-tasks')
}
</script>

<style scoped>
.maintenance-view {
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

.tasks-card,
.results-card {
  margin-bottom: 20px;
  background-color: #1e293b;
  border: 1px solid #334155;
}

.tasks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 15px;
  margin-bottom: 20px;
}

.task-item {
  padding: 15px;
  border: 1px solid #334155;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  background-color: #0f172a;
}

.task-item:hover {
  border-color: #475569;
  transform: translateY(-2px);
}

.task-item.selected {
  border-color: #3b82f6;
  background-color: #1e3a8a;
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.task-header h4 {
  margin: 0;
  color: #ffffff;
  font-size: 16px;
}

.task-description {
  margin: 0 0 10px 0;
  color: #94a3b8;
  font-size: 14px;
  line-height: 1.4;
}

.task-meta {
  display: flex;
  justify-content: flex-end;
}

.task-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  padding-top: 15px;
  border-top: 1px solid #334155;
}

.results-list {
  margin-bottom: 20px;
}

.result-item {
  padding: 15px;
  border-radius: 8px;
  margin-bottom: 10px;
  border-left: 4px solid;
}

.result-item.success {
  background-color: #064e3b;
  border-left-color: #10b981;
}

.result-item.error {
  background-color: #7f1d1d;
  border-left-color: #ef4444;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.task-name {
  font-weight: 600;
  color: #ffffff;
}

.result-details p {
  margin: 5px 0;
  font-size: 14px;
}

.result-details .message {
  color: #e2e8f0;
}

.result-details .duration {
  color: #94a3b8;
}

.result-details .error {
  color: #fca5a5;
}

.results-summary {
  display: flex;
  justify-content: space-around;
  padding-top: 15px;
  border-top: 1px solid #334155;
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

:deep(.el-statistic__head) {
  color: #94a3b8;
}

:deep(.el-statistic__content) {
  color: #ffffff;
}
</style>
