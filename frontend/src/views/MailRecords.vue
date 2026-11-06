<template>
  <a-card>
    <div class="toolbar">
      <a-input v-model:value="query.email" placeholder="收件邮箱" allow-clear style="width:180px" @keyup.enter="load" />
      <a-select v-model:value="query.interface_id" placeholder="全部接口" allow-clear show-search option-filter-prop="label" style="width:170px" :options="ifaceOptions" />
      <a-select v-model:value="query.status" placeholder="发送状态" allow-clear style="width:130px">
        <a-select-option v-for="(t, i) in statusList" :key="i" :value="i">{{ t }}</a-select-option>
      </a-select>
      <a-range-picker v-model:value="range" show-time format="YYYY-MM-DD HH:mm:ss" :placeholder="['开始时间', '结束时间']" style="width:360px" @change="onRangeChange" />
      <a-button type="primary" @click="load">搜索</a-button>
    </div>

    <a-table :dataSource="list" :columns="columns" :loading="loading" :pagination="pagination" rowKey="id" @change="onTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'status'">
          <a-tag :color="statusType(record.status)">{{ statusText(record.status) }}</a-tag>
        </template>
        <template v-if="column.dataIndex === 'last_sent_at'">
          {{ fmt(record.last_sent_at || record.created_at) }}
        </template>
        <template v-if="column.dataIndex === 'action'">
          <a-space>
            <a-button size="small" type="link" @click="view(record)">查看</a-button>
            <a-button size="small" type="link" @click="copyRow(record)">复制</a-button>
            <a-button v-if="record.status === 4 || record.status === 3" size="small" type="link" danger @click="retry(record)">重发</a-button>
          </a-space>
        </template>
      </template>
    </a-table>
  </a-card>

  <a-drawer v-model:open="detailVisible" title="邮件详情" :width="640">
    <template v-if="detail">
      <a-descriptions :column="1" bordered size="small" style="margin-bottom:14px">
        <a-descriptions-item label="收件邮箱">{{ detail.recipient_email }}</a-descriptions-item>
        <a-descriptions-item label="收件人（发送时昵称快照）">{{ detail.recipient_name }}</a-descriptions-item>
        <a-descriptions-item label="标题">{{ detail.subject || '-' }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="statusType(detail.status)">{{ statusText(detail.status) }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="尝试次数">{{ detail.attempt_count }}</a-descriptions-item>
        <a-descriptions-item v-if="detail.last_error" label="失败原因">
          <span style="color:#ff4d4f">{{ detail.last_error }}</span>
        </a-descriptions-item>
      </a-descriptions>
      <div class="section-title">
        复制用文本（序号 → 资质所属地）
        <a-button size="small" type="primary" ghost style="margin-left:8px" @click="copyRow(detail)">一键复制</a-button>
      </div>
      <pre class="copy-block">{{ detail.copy_text || '暂无数据' }}</pre>
      <div class="section-title">HTML 内容（发送时快照）</div>
      <iframe v-if="detail.html_content" :srcdoc="detail.html_content" class="mail-frame"></iframe>
      <a-empty v-else description="尚未渲染（任务未发送）" />
      <div v-if="detail.status === 4 || detail.status === 3" style="margin-top:14px;text-align:right">
        <a-button type="primary" danger @click="retry(detail); detailVisible = false">手动重发</a-button>
      </div>
    </template>
  </a-drawer>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import api from '../api'
import { fmtTime, statusText, statusType, statusList, copyToClipboard } from '../utils'

const fmt = fmtTime

const list = ref([])
const total = ref(0)
const loading = ref(false)
const ifaces = ref([])
const range = ref(null)
const query = reactive({ page: 1, page_size: 20, email: '', interface_id: '', status: '' })

const pagination = reactive({
  current: 1, pageSize: 20, total: 0,
  showTotal: t => `共 ${t} 条`, showSizeChanger: false
})

const columns = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '收件人', dataIndex: 'recipient_name', width: 100, ellipsis: true },
  { title: '收件邮箱', dataIndex: 'recipient_email', width: 170, ellipsis: true },
  { title: '接口', dataIndex: 'interface_name', width: 110, ellipsis: true },
  { title: '邮件标题', dataIndex: 'subject', ellipsis: true },
  { title: '状态', dataIndex: 'status', width: 96 },
  { title: '尝试', dataIndex: 'attempt_count', width: 64 },
  { title: '最近发送', dataIndex: 'last_sent_at', width: 160 },
  { title: '操作', dataIndex: 'action', width: 208, fixed: 'right' }
]

const ifaceOptions = computed(() => ifaces.value.map(i => ({ value: i.id, label: i.name })))

const detailVisible = ref(false)
const detail = ref(null)

function onRangeChange(dates) {
  if (dates && dates.length === 2) {
    query.start_time = dates[0].format('YYYY-MM-DD HH:mm:ss')
    query.end_time = dates[1].format('YYYY-MM-DD HH:mm:ss')
  } else {
    query.start_time = ''
    query.end_time = ''
  }
}

function onTableChange(pag) {
  query.page = pag.current
  pagination.current = pag.current
  load()
}

async function load() {
  loading.value = true
  try {
    const data = await api.get('/mail-records', { params: query })
    list.value = data.list
    total.value = data.total
    pagination.total = data.total
  } finally { loading.value = false }
}

async function view(row) {
  detail.value = await api.get('/mail-records/' + row.id)
  detailVisible.value = true
}

async function retry(row) {
  Modal.confirm({
    title: '手动重发',
    content: '确认重发该邮件？尝试次数将重新计数。',
    okType: 'danger',
    onOk: async () => {
      await api.post(`/mail-records/${row.id}/retry`)
      message.success('已重置为待发送，Worker 将尽快处理')
      load()
    }
  })
}

async function copyRow(row) {
  let text = row.copy_text
  if (!text) {
    const d = await api.get(`/mail-records/${row.id}/copy-text`)
    text = d.text
  }
  if (!text) { message.warning('暂无可复制内容'); return }
  await copyToClipboard(text)
  message.success('已复制：序号 → 资质所属地')
}

onMounted(async () => {
  load()
  try {
    const d = await api.get('/interfaces', { params: { page: 1, page_size: 100 } })
    ifaces.value = d.list
  } catch { /* ignore */ }
})
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 14px; flex-wrap: wrap; }
.section-title { font-weight: bold; margin: 10px 0 8px; }
.mail-frame { width: 100%; height: 420px; border: 1px solid #ebeef5; border-radius: 6px; background: #fff; }
.copy-block {
  margin: 0; padding: 12px 14px; background: #f7f8fa; border: 1px solid #ebeef5;
  border-radius: 6px; font-family: Consolas, Menlo, monospace; font-size: 13px;
  line-height: 1.9; white-space: pre-wrap; word-break: break-all; color: #303133;
}
</style>
