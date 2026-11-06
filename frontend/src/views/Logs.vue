<template>
  <a-card>
    <div class="toolbar">
      <a-select v-model:value="query.module" placeholder="全部模块" allow-clear style="width:160px">
        <a-select-option v-for="m in modules" :key="m" :value="m">{{ m }}</a-select-option>
      </a-select>
      <a-button type="primary" @click="load">筛选</a-button>
    </div>

    <a-table :dataSource="list" :columns="columns" :loading="loading" :pagination="pagination" rowKey="id" @change="onTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'action_type'">
          <a-tag :color="actionType(record.action)">{{ record.action }}</a-tag>
        </template>
        <template v-if="column.dataIndex === 'result'">
          <a-tag :color="record.result === 'success' ? 'success' : 'error'">
            {{ record.result === 'success' ? '成功' : '失败' }}
          </a-tag>
        </template>
        <template v-if="column.dataIndex === 'created_at'">
          {{ fmt(record.created_at) }}
        </template>
      </template>
    </a-table>
  </a-card>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api'
import { fmtTime } from '../utils'

const fmt = fmtTime

const list = ref([])
const total = ref(0)
const loading = ref(false)
const modules = ['用户管理', '接口管理', '邮件配置', '邮件模板', '邮件记录', '接收记录', '系统']
const query = reactive({ page: 1, page_size: 20, module: '' })

const pagination = reactive({
  current: 1, pageSize: 20, total: 0,
  showTotal: t => `共 ${t} 条`, showSizeChanger: false
})

const columns = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '操作人', dataIndex: 'operator', width: 100 },
  { title: '模块', dataIndex: 'module', width: 110 },
  { title: '操作类型', dataIndex: 'action_type', width: 90 },
  { title: '描述', dataIndex: 'description', ellipsis: true },
  { title: '结果', dataIndex: 'result', width: 90 },
  { title: '来源 IP', dataIndex: 'source_ip', width: 130 },
  { title: '时间', dataIndex: 'created_at', width: 165 }
]

const actionType = a => ({ POST: 'success', PUT: 'warning', DELETE: 'danger' }[a] || 'processing')

function onTableChange(pag) {
  query.page = pag.current
  pagination.current = pag.current
  load()
}

async function load() {
  loading.value = true
  try {
    const data = await api.get('/logs', { params: query })
    list.value = data.list
    total.value = data.total
    pagination.total = data.total
  } finally { loading.value = false }
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 14px; }
</style>
