<template>
  <div class="wipe-view">
    <div class="view-header">
      <h2>Затирание дисков</h2>
    </div>

    <!-- Disk Selection -->
    <el-card class="selection-card">
      <template #header>
        <span>Выбор диска</span>
      </template>
      
      <el-form :model="wipeForm" label-width="120px">
        <el-form-item label="Диск:">
          <el-select 
            v-model="wipeForm.drive" 
            placeholder="Выберите диск для затирания"
            style="width: 100%"
            :disabled="isWiping"
          >
            <el-option
              v-for="disk in availableDisks"
              :key="disk.letter"
              :label="`${disk.letter} - ${disk.freeSize.toFixed(2)} GB свободно`"
              :value="disk.letter"
              :disabled="disk.isSystem || !disk.isWritable"
            >
              <div class="disk-option">
                <span class="disk-letter">{{ disk.letter }}</span>
                <span class="disk-info">{{ disk.freeSize.toFixed(2) }} GB свободно</span>
                <el-tag 
                  v-if="disk.isSystem" 
                  type="danger" 
                  size="small"
                >
                  Системный
                </el-tag>
                <el-tag 
                  v-if="!disk.isWritable" 
                  type="warning" 
                  size="small"
                >
                  Только чтение
                </el-tag>
              </div>
            </el-option>
          </el-select>
        </el-form-item>

        <el-form-item label="Метод:">
          <el-radio-group v-model="wipeForm.method" :disabled="isWiping">
            <el-radio label="standard">Стандартный</el-radio>
            <el-radio label="sdelete">SDelete совместимый</el-radio>
            <el-radio label="cipher">Шифрование</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="Паттерн:">
          <el-radio-group v-model="wipeForm.pattern" :disabled="isWiping">
            <el-radio label="random">Случайные данные</el-radio>
            <el-radio label="zeros">Нули</el-radio>
            <el-radio label="ones">Единицы</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item>
          <el-button 
            type="primary" 
            @click="startWipe"
            :disabled="!canStartWipe"
            :loading="isWiping"
          >
            <el-icon><Delete /></el-icon>
            {{ isWiping ? 'Затирание...' : 'Начать затирание' }}
          </el-button>
          
          <el-button 
            v-if="isWiping"
            type="danger"
            @click="stopWipe"
          >
            <el-icon><Close /></el-icon>
            Остановить
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- Progress -->
    <el-card v-if="isWiping || wipeProgress" class="progress-card">
      <template #header>
        <span>Прогресс затирания</span>
      </template>
      
      <div class="progress-content">
        <div class="progress-main">
          <el-progress 
            type="circle"
            :percentage="wipeProgress?.percentage || 0"
            :width="120"
            :stroke-width="8"
            :color="progressColor"
          >
            <template #default="{ percentage }">
              <span class="progress-text">{{ percentage.toFixed(1) }}%</span>
            </template>
          </el-progress>
        </div>

        <div class="progress-details">
          <div class="detail-row">
            <span class="label">Записано:</span>
            <span class="value">{{ (wipeProgress?.bytesWritten || 0).toFixed(2) }} GB</span>
          </div>
          <div class="detail-row">
            <span class="label">Скорость:</span>
            <span class="value">{{ (wipeProgress?.speedMBps || 0).toFixed(1) }} MB/s</span>
          </div>
          <div class="detail-row">
            <span class="label">Время работы:</span>
            <span class="value">{{ wipeProgress?.elapsedTime || '00:00:00' }}</span>
          </div>
          <div class="detail-row">
            <span class="label">Осталось:</span>
            <span class="value">{{ wipeProgress?.estimatedTime || 'Расчет...' }}</span>
          </div>
        </div>
      </div>
    </el-card>

    <!-- Result -->
    <el-card v-if="wipeResult" class="result-card">
      <template #header>
        <span>Результат затирания</span>
      </template>
      
      <el-alert
        :title="wipeResult.success ? 'Затирание завершено успешно' : 'Ошибка затирания'"
        :type="wipeResult.success ? 'success' : 'error'"
        :closable="false"
        show-icon
      >
        <div v-if="wipeResult.success" class="result-details">
          <p>Записано: <strong>{{ wipeResult.bytesWritten.toFixed(2) }} GB</strong></p>
          <p>Длительность: <strong>{{ wipeResult.duration }}</strong></p>
          <p>Средняя скорость: <strong>{{ wipeResult.speedMBps.toFixed(1) }} MB/s</strong></p>
        </div>
        <div v-else class="error-details">
          <p>Ошибка: <strong>{{ wipeResult.error }}</strong></p>
        </div>
      </el-alert>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Delete, Close } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps({
  disks: {
    type: Array,
    default: () => []
  },
  wipeProgress: {
    type: Object,
    default: null
  },
  wipeResult: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['start-wipe'])

const isWiping = ref(false)
const wipeForm = ref({
  drive: '',
  method: 'standard',
  pattern: 'random'
})

const availableDisks = computed(() => {
  return props.disks.filter(disk => !disk.isSystem && disk.isWritable)
})

const canStartWipe = computed(() => {
  return wipeForm.value.drive && !isWiping.value
})

const progressColor = computed(() => {
  const percentage = props.wipeProgress?.percentage || 0
  if (percentage > 90) return '#67c23a'
  if (percentage > 50) return '#e6a23c'
  return '#409eff'
})

const startWipe = async () => {
  try {
    await ElMessageBox.confirm(
      `Вы уверены, что хотите затереть диск ${wipeForm.value.drive}?`,
      'Подтверждение затирания',
      {
        confirmButtonText: 'Да, затереть',
        cancelButtonText: 'Отмена',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )

    isWiping.value = true
    emit('start-wipe', wipeForm.value.drive)
    
    ElMessage.success(`Начато затирание диска ${wipeForm.value.drive}`)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(`Ошибка запуска затирания: ${error}`)
    }
  }
}

const stopWipe = async () => {
  try {
    await ElMessageBox.confirm(
      'Вы уверены, что хотите остановить затирание?',
      'Подтверждение остановки',
      {
        confirmButtonText: 'Да, остановить',
        cancelButtonText: 'Продолжить',
        type: 'warning'
      }
    )
    
    isWiping.value = false
    ElMessage.info('Затирание остановлено пользователем')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(`Ошибка остановки: ${error}`)
    }
  }
}

// Event listeners
let wipeProgressListener = null
let wipeCompleteListener = null

const onWipeProgress = (progress) => {
  props.wipeProgress = progress
}

const onWipeComplete = (result) => {
  props.wipeResult = result
  props.wipeProgress = null
  // Update application status
}

// Setup Wails event listeners
const setupEventListeners = () => {
  if (window.runtime && window.runtime.EventsOn) {
    wipeProgressListener = window.runtime.EventsOn('wipe-progress', onWipeProgress)
    wipeCompleteListener = window.runtime.EventsOn('wipe-complete', onWipeComplete)
  }
}

// Cleanup event listeners
const cleanupEventListeners = () => {
  if (wipeProgressListener) wipeProgressListener()
  if (wipeCompleteListener) wipeCompleteListener()
}

// Setup event listeners on mount
setupEventListeners()

// Cleanup on unmount
const { onUnmounted } = require('vue')
onUnmounted(() => {
  cleanupEventListeners()
})

// Watch for wipe completion
watch(() => props.wipeResult, (result) => {
  if (result) {
    isWiping.value = false
  }
})

// Auto-select first available disk
watch(() => availableDisks.value, (disks) => {
  if (disks.length > 0 && !wipeForm.value.drive) {
    wipeForm.value.drive = disks[0].letter
  }
}, { immediate: true })
</script>

<!-- ... -->
  color: #ffffff;
}

.selection-card,
.progress-card,
.result-card {
  margin-bottom: 20px;
  background-color: #1e293b;
  border: 1px solid #334155;
}

.disk-option {
  display: flex;
  align-items: center;
  gap: 10px;
}

.disk-letter {
  font-weight: 600;
  color: #ffffff;
}

.disk-info {
  color: #94a3b8;
  flex: 1;
}

.progress-content {
  display: flex;
  align-items: center;
  gap: 40px;
}

.progress-main {
  flex-shrink: 0;
}

.progress-text {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
}

.progress-details {
  flex: 1;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  padding: 8px 0;
  border-bottom: 1px solid #334155;
}

.detail-row:last-child {
  border-bottom: none;
  margin-bottom: 0;
}

.label {
  color: #94a3b8;
  font-size: 14px;
}

.value {
  color: #ffffff;
  font-weight: 500;
}

.result-details,
.error-details {
  margin-top: 10px;
}

.result-details p,
.error-details p {
  margin: 5px 0;
  color: #ffffff;
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

:deep(.el-form-item__label) {
  color: #94a3b8;
}

:deep(.el-radio__label) {
  color: #ffffff;
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
</style>
