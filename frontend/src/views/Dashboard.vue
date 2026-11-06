<template>
  <div v-if="d">
    <a-row :gutter="16">
      <a-col :xs="12" :sm="8" :md="6" v-for="card in cards" :key="card.label">
        <a-card class="stat-card" hoverable>
          <div class="stat-value" :style="{ color: card.color }">{{ card.value }}</div>
          <div class="stat-label">{{ card.label }}</div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top:16px">
      <a-col :xs="24" :md="14">
        <a-card title="最近 7 天数据接收 / 邮件发送趋势">
          <div class="trend">
            <div v-for="t in d.trend" :key="t.date" class="trend-col">
              <div class="trend-bars">
                <div class="bar push-bar" :style="{ height: barH(t.push_count, maxTrend) + 'px' }" :title="'接收 ' + t.push_count"></div>
                <div class="bar mail-bar" :style="{ height: barH(t.mail_count, maxTrend) + 'px' }" :title="'发送 ' + t.mail_count"></div>
              </div>
              <div class="trend-date">{{ t.date }}</div>
              <div class="trend-nums">{{ t.push_count }} / {{ t.mail_count }}</div>
            </div>
          </div>
          <div class="legend">
            <span><i class="dot" style="background:#1677ff"></i>数据接收</span>
            <span><i class="dot" style="background:#52c41a"></i>邮件发送</span>
          </div>
        </a-card>
      </a-col>
      <a-col :xs="24" :md="10">
        <a-card title="接口统计排行">
          <a-table :dataSource="d.rank" :columns="rankColumns" :pagination="false" size="small" :scroll="{ y: 320 }" rowKey="interface_name" />
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../api'

const d = ref(null)
const maxTrend = computed(() => {
  if (!d.value) return 1
  return Math.max(...d.value.trend.map(t => Math.max(t.push_count, t.mail_count)), 1)
})
const barH = (v, max) => Math.max(2, Math.round(v / max * 140))

const rankColumns = [
  { title: '接口', dataIndex: 'interface_name', ellipsis: true },
  { title: '接收量', dataIndex: 'push_count', width: 90, sorter: (a, b) => a.push_count - b.push_count },
  { title: '发送量', dataIndex: 'mail_count', width: 90, sorter: (a, b) => a.mail_count - b.mail_count }
]

const cards = computed(() => d.value ? [
  { label: '用户总数 / 启用', value: `${d.value.user_total} / ${d.value.user_active}`, color: '#1677ff' },
  { label: '接口总数 / 启用', value: `${d.value.interface_total} / ${d.value.interface_active}`, color: '#faad14' },
  { label: '今日接收 / 累计', value: `${d.value.push_today} / ${d.value.push_total}`, color: '#1f4e79' },
  { label: '今日邮件任务', value: d.value.mail_today, color: '#52c41a' },
  { label: '发送成功', value: d.value.mail_success, color: '#52c41a' },
  { label: '发送失败', value: d.value.mail_failed, color: '#ff4d4f' },
  { label: '待发送', value: d.value.mail_pending, color: '#8c8c8c' },
  { label: '发送成功率', value: d.value.mail_success_rate.toFixed(1) + '%', color: '#1677ff' }
] : [])

onMounted(async () => { d.value = await api.get('/dashboard') })
</script>

<style scoped>
.stat-card { margin-bottom: 16px; text-align: center; }
.stat-value { font-size: 26px; font-weight: bold; }
.stat-label { color: #8c8c8c; font-size: 13px; margin-top: 6px; }
.trend { display: flex; justify-content: space-around; align-items: flex-end; height: 200px; }
.trend-col { text-align: center; }
.trend-bars { display: flex; gap: 4px; align-items: flex-end; height: 160px; }
.bar { width: 18px; border-radius: 3px 3px 0 0; }
.push-bar { background: #1677ff; }
.mail-bar { background: #52c41a; }
.trend-date { color: #8c8c8c; font-size: 12px; margin-top: 6px; }
.trend-nums { color: #666; font-size: 11px; }
.legend { margin-top: 10px; color: #666; font-size: 12px; display: flex; gap: 20px; }
.dot { display: inline-block; width: 10px; height: 10px; border-radius: 2px; margin-right: 4px; }
</style>
