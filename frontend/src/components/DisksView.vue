<template>
  <div class="disks-view">
    <div class="view-header">
      <h2>Информация о дисках</h2>
      <el-button @click="refreshDisks" :loading="loading">
        <el-icon><Refresh /></el-icon>
        Обновить
      </el-button>
    </div>

    <div class="disks-grid" v-if="disks.length > 0">
      <el-card 
        v-for="disk in disks" 
        :key="disk.letter"
        class="disk-card"
        :class="{ 'system-disk': disk.isSystem, 'writable': disk.isWritable }"
      >
        <template #header>
          <div class="disk-header">
            <span class="disk-letter">{{ disk.letter }}</span>
            <el-tag 
              :type="disk.isSystem ? 'danger' : 'success'"
              size="small"
            >
              {{ disk.isSystem ? 'Системный' : 'Доступный' }}
            </el-tag>
          </div>
        </template>

        <div class="disk-info">
          <div class="info-row">
            <span class="label">Тип:</span>
            <span class="value">{{ disk.type }}</span>
          </div>
          <div class="info-row">
            <span class="label">Всего:</span>
            <span class="value">{{ disk.totalSize.toFixed(2) }} GB</span>
          </div>
          <div class="info-row">
            <span class="label">Свободно:</span>
            <span class="value">{{ disk.freeSize.toFixed(2) }} GB</span>
          </div>
          <div class="info-row">
            <span class="label">Использовано:</span>
            <span class="value">{{ disk.usedSize.toFixed(2) }} GB</span>
          </div>
          <div class="info-row">
            <span class="label">Запись:</span>
            <el-tag 
              :type="disk.isWritable ? 'success' : 'danger'"
              size="small"
            >
              {{ disk.isWritable ? 'Разрешена' : 'Запрещена' }}
            </el-tag>
          </div>
        </div>

        <div class="disk-actions">
          <el-button 
            type="primary" 
            size="small"
            :disabled="disk.isSystem || !disk.isWritable"
            @click="$emit('wipe-disk', disk)"
          >
            <el-icon><Delete /></el-icon>
            Затереть
          </el-button>
        </div>

        <!-- Progress bar -->
        <div class="disk-usage">
          <el-progress 
            :percentage="usagePercentage(disk)"
            :color="progressColor(disk)"
            :show-text="false"
            :stroke-width="8"
          />
          <span class="usage-text">{{ usagePercentage(disk).toFixed(1) }}% использовано</span>
        </div>
      </el-card>
    </div>

    <el-empty 
      v-else
      description="Диски не найдены"
      :image-size="200"
    >
      <el-button type="primary" @click="refreshDisks">
        <el-icon><Refresh /></el-icon>
        Обновить
      </el-button>
    </el-empty>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Refresh, Delete } from '@element-plus/icons-vue'

defineProps({
  disks: {
    type: Array,
    default: () => []
  },
  loading: {
    type: Boolean,
    default: false
  }
})

defineEmits(['wipe-disk', 'refresh-disks'])

const usagePercentage = (disk) => {
  return (disk.usedSize / disk.totalSize) * 100
}

const progressColor = (disk) => {
  const percentage = usagePercentage(disk)
  if (percentage > 90) return '#f56565'
  if (percentage > 75) return '#ed8936'
  if (percentage > 50) return '#ecc94b'
  return '#48bb78'
}
</script>

<style scoped>
.disks-view {
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

.disks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 20px;
}

.disk-card {
  background-color: #1e293b;
  border: 1px solid #334155;
  transition: all 0.3s ease;
}

.disk-card:hover {
  border-color: #475569;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.disk-card.system-disk {
  border-color: #ef4444;
}

.disk-card.writable {
  border-color: #10b981;
}

.disk-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.disk-letter {
  font-size: 18px;
  font-weight: 600;
  color: #ffffff;
}

.disk-info {
  margin-bottom: 20px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  padding: 4px 0;
}

.label {
  color: #94a3b8;
  font-size: 14px;
}

.value {
  color: #ffffff;
  font-weight: 500;
}

.disk-actions {
  margin-bottom: 15px;
}

.disk-usage {
  text-align: center;
}

.usage-text {
  display: block;
  margin-top: 8px;
  font-size: 12px;
  color: #94a3b8;
}

:deep(.el-card__header) {
  background-color: #0f172a;
  border-bottom: 1px solid #334155;
}

:deep(.el-card__body) {
  background-color: #1e293b;
  color: #ffffff;
}

:deep(.el-progress-bar__outer) {
  background-color: #334155;
}

:deep(.el-empty__description) {
  color: #94a3b8;
}
</style>
