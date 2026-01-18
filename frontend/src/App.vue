<template>
  <el-container class="app-container">
    <!-- Header -->
    <el-header class="app-header">
      <div class="header-content">
        <div class="logo">
          <el-icon><Delete /></el-icon>
          <span class="title">WipeDisk Enterprise</span>
          <span class="version">v1.3.0</span>
        </div>
        <div class="header-actions">
          <el-button type="primary" @click="showDiagnostics" :loading="diagnosticsLoading">
            <el-icon><Monitor /></el-icon>
            Диагностика
          </el-button>
        </div>
      </div>
    </el-header>

    <!-- Main Content -->
    <el-main class="app-main">
      <el-tabs v-model="activeTab" type="border-card">
        <!-- Disks Tab -->
        <el-tab-pane label="Диски" name="disks">
          <DisksView @wipe-disk="handleWipeDisk" />
        </el-tab-pane>

        <!-- Wipe Tab -->
        <el-tab-pane label="Затирание" name="wipe">
          <WipeView 
            :disks="disks" 
            :wipe-progress="wipeProgress"
            :wipe-result="wipeResult"
            @start-wipe="handleStartWipe"
          />
        </el-tab-pane>

        <!-- Maintenance Tab -->
        <el-tab-pane label="Обслуживание" name="maintenance">
          <MaintenanceView 
            :tasks="maintenanceTasks"
            :results="maintenanceResults"
            @run-tasks="handleRunMaintenanceTasks"
          />
        </el-tab-pane>

        <!-- Diagnostics Tab -->
        <el-tab-pane label="Диагностика" name="diagnostics">
          <DiagnosticsView 
            :diagnostic-results="diagnosticResults"
            :loading="diagnosticsLoading"
            @run-diagnostics="handleRunDiagnostics"
          />
        </el-tab-pane>
      </el-tabs>
    </el-main>

    <!-- Status Bar -->
    <el-footer class="app-footer">
      <div class="status-bar">
        <span class="status-item">
          <el-icon><CircleCheck /></el-icon>
          Статус: {{ applicationStatus }}
        </span>
        <span class="status-item">
          <el-icon><Clock /></el-icon>
          Время работы: {{ uptime }}
        </span>
      </div>
    </el-footer>
  </el-container>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, Monitor, CircleCheck, Clock } from '@element-plus/icons-vue'
import DisksView from './components/DisksView.vue'
import WipeView from './components/WipeView.vue'
import MaintenanceView from './components/MaintenanceView.vue'
import DiagnosticsView from './components/DiagnosticsView.vue'

// Reactive data
const activeTab = ref('disks')
const disks = ref([])
const wipeProgress = ref(null)
const wipeResult = ref(null)
const maintenanceTasks = ref([])
const maintenanceResults = ref([])
const diagnosticResults = ref(null)
const diagnosticsLoading = ref(false)
const applicationStatus = ref('Готов к работе')
const uptime = ref('00:00:00')

// Wails bindings
const { GetDisks, StartWipe, GetMaintenanceTasks, RunMaintenanceTasks, GetDiagnostics } = window.go.wipedisk_enterprise.internal.app.App

// Event listeners
let wipeProgressListener = null
let wipeCompleteListener = null

// Methods
const loadDisks = async () => {
  try {
    const diskList = await GetDisks()
    disks.value = diskList
  } catch (error) {
    ElMessage.error(`Ошибка загрузки дисков: ${error}`)
  }
}

const handleWipeDisk = (disk) => {
  activeTab.value = 'wipe'
  // WipeView will handle the actual wipe
}

const handleStartWipe = async (drive) => {
  try {
    applicationStatus.value = 'Затирание диска...'
    await StartWipe(drive)
    ElMessage.success(`Начато затирание диска ${drive}`)
  } catch (error) {
    ElMessage.error(`Ошибка запуска затирания: ${error}`)
    applicationStatus.value = 'Ошибка'
  }
}

const loadMaintenanceTasks = async () => {
  try {
    const tasks = await GetMaintenanceTasks()
    maintenanceTasks.value = tasks
  } catch (error) {
    ElMessage.error(`Ошибка загрузки задач: ${error}`)
  }
}

const handleRunMaintenanceTasks = async (taskIds) => {
  try {
    applicationStatus.value = 'Выполнение обслуживания...'
    const results = await RunMaintenanceTasks(taskIds)
    maintenanceResults.value = results
    
    const successCount = results.filter(r => r.success).length
    ElMessage.success(`Выполнено ${successCount}/${results.length} задач`)
    applicationStatus.value = 'Готов к работе'
  } catch (error) {
    ElMessage.error(`Ошибка выполнения задач: ${error}`)
    applicationStatus.value = 'Ошибка'
  }
}

const handleRunDiagnostics = async (level) => {
  try {
    diagnosticsLoading.value = true
    applicationStatus.value = 'Диагностика системы...'
    const results = await GetDiagnostics(level)
    diagnosticResults.value = results
    ElMessage.success('Диагностика завершена')
    applicationStatus.value = 'Готов к работе'
  } catch (error) {
    ElMessage.error(`Ошибка диагностики: ${error}`)
    applicationStatus.value = 'Ошибка'
  } finally {
    diagnosticsLoading.value = false
  }
}

const showDiagnostics = () => {
  activeTab.value = 'diagnostics'
  handleRunDiagnostics('quick')
}

const updateUptime = () => {
  const now = Date.now()
  const start = window.startTime || now
  const elapsed = Math.floor((now - start) / 1000)
  const hours = Math.floor(elapsed / 3600)
  const minutes = Math.floor((elapsed % 3600) / 60)
  const seconds = elapsed % 60
  uptime.value = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
}

// Event handlers
const onWipeProgress = (progress) => {
  wipeProgress.value = progress
}

const onWipeComplete = (result) => {
  wipeResult.value = result
  wipeProgress.value = null
  applicationStatus.value = result.success ? 'Затирание завершено' : 'Ошибка затирания'
  
  if (result.success) {
    ElMessage.success(`Затирание завершено. Записано: ${result.bytesWritten.toFixed(2)} GB`)
  } else {
    ElMessage.error(`Ошибка затирания: ${result.error}`)
  }
}

// Lifecycle
onMounted(async () => {
  window.startTime = Date.now()
  
  // Load initial data
  await loadDisks()
  await loadMaintenanceTasks()
  
  // Setup event listeners
  if (window.runtime) {
    wipeProgressListener = window.runtime.EventsOn('wipe-progress', onWipeProgress)
    wipeCompleteListener = window.runtime.EventsOn('wipe-complete', onWipeComplete)
  }
  
  // Start uptime counter
  const interval = setInterval(updateUptime, 1000)
  
  // Cleanup on unmount
  onUnmounted(() => {
    clearInterval(interval)
    if (wipeProgressListener) wipeProgressListener()
    if (wipeCompleteListener) wipeCompleteListener()
  })
})
</script>

<style scoped>
.app-container {
  height: 100vh;
  background-color: #1b2634;
  color: #ffffff;
}

.app-header {
  background-color: #0f172a;
  border-bottom: 1px solid #334155;
  padding: 0;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 100%;
  padding: 0 20px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo .el-icon {
  font-size: 24px;
  color: #ef4444;
}

.title {
  font-size: 18px;
  font-weight: 600;
  color: #ffffff;
}

.version {
  font-size: 12px;
  color: #64748b;
  background-color: #1e293b;
  padding: 2px 8px;
  border-radius: 4px;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.app-main {
  padding: 20px;
  background-color: #1b2634;
}

.app-footer {
  background-color: #0f172a;
  border-top: 1px solid #334155;
  padding: 0;
  height: 40px;
}

.status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 100%;
  padding: 0 20px;
  font-size: 12px;
  color: #94a3b8;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 5px;
}

.status-item .el-icon {
  font-size: 14px;
}

:deep(.el-tabs__header) {
  background-color: #1e293b;
  margin: 0;
}

:deep(.el-tabs__nav-wrap) {
  background-color: #1e293b;
}

:deep(.el-tabs__item) {
  color: #94a3b8;
  background-color: #1e293b;
  border-color: #334155;
}

:deep(.el-tabs__item.is-active) {
  color: #ffffff;
  background-color: #334155;
  border-color: #475569;
}

:deep(.el-tabs__content) {
  background-color: #1b2634;
  border: 1px solid #334155;
  border-top: none;
  padding: 20px;
}
</style>
