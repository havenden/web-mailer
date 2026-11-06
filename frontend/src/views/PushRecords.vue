<template>
  <a-card>
    <div class="toolbar">
      <a-select v-model:value="query.interface_id" placeholder="全部接口" allow-clear show-search option-filter-prop="label" style="width:180px" :options="ifaceOptions" />
      <a-input v-model:value="query.phone" placeholder="手机号" allow-clear style="width:150px" @keyup.enter="load" />
      <a-input v-model:value="query.sequence" placeholder="序号" allow-clear style="width:150px" @keyup.enter="load" />
      <a-range-picker v-model:value="range" show-time format="YYYY-MM-DD HH:mm:ss" :placeholder="['开始时间', '结束时间']" style="width:360px" @change="onRangeChange" />
      <a-button type="primary" @click="load">搜索</a-button>
    </div>

    <a-table :dataSource="list" :columns="columns" :loading="loading" :pagination="pagination" rowKey="id" @change="onTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'mail_info'">
          <a-tag v-if="record.mail_total" color="default">共 {{ record.mail_total }}</a-tag>
          <a-tag color="success">成功 {{ record.mail_success }}</a-tag>
          <a-tag v-if="record.mail_failed" color="error">失败 {{ record.mail_failed }}</a-tag>
          <a-tag v-if="record.mail_pending" color="warning">待发 {{ record.mail_pending }}</a-tag>
        </template>
        <template v-if="column.dataIndex === 'created_at'">
          {{ fmt(record.created_at) }}
        </template>
        <template v-if="column.dataIndex === 'action'">
          <a-button size="small" type="link" @click="view(record)">详情</a-button>
        </template>
      </template>
    </a-table>
  </a-card>

  <a-drawer v-model:open="detailVisible" title="接收记录详情" :width="600">
    <template v-if="detail">
      <a-descriptions :column="2" bordered size="small" style="margin-bottom:14px">
        <a-descriptions-item label="记录 ID">{{ detail.record.id }}</a-descriptions-item>
        <a-descriptions-item label="序号">{{ detail.record.sequence }}</a-descriptions-item>
        <a-descriptions-item label="接口">{{ detail.interface_name }}</a-descriptions-item>
        <a-descriptions-item label="接收时间">{{ fmt(detail.record.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="手机号">{{ detail.record.phone }}</a-descriptions-item>
        <a-descriptions-item label="客户名">{{ detail.record.name }}</a-descriptions-item>
      </a-descriptions>
      <div class="section-title">原始 JSON</div>
      <pre class="json-box">{{ prettyJson(detail.record.raw_json) }}</pre>
      <div class="section-title">关联邮件任务（{{ detail.mail_records.length }}）</div>
      <a-table :dataSource="detail.mail_records" :columns="mailColumns" size="small" :pagination="false" rowKey="id">
        <template #bodyCell="{ column, record: mail }">
          <template v-if="column.dataIndex === 'status'">
            <a-tag :color="statusType(mail.status)">{{ statusText(mail.status) }}</a-tag>
          </template>
          <template v-if="column.dataIndex === 'action'">
            <a-button size="small" type="link" @click="copyMail(mail)">复制</a-button>
          </template>
        </template>
      </a-table>
    </template>
  </a-drawer>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import api from '../api'
import { fmtTime, statusText, statusType, copyToClipboard } from '../utils'

const fmt = fmtTime

const list = ref([])
const total = ref(0)
const loading = ref(false)
const ifaces = ref([])
const range = ref(null)
const query = reactive({ page: 1, page_size: 20, interface_id: '', phone: '', sequence: '' })

const pagination = reactive({
  current: 1, pageSize: 20, total: 0,
  showTotal: t => `共 ${t} 条`, showSizeChanger: false
})

const columns = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '序号', dataIndex: 'sequence', width: 160 },
  { title: '接口', dataIndex: 'interface_name', width: 120, ellipsis: true },
  { title: '号码', dataIndex: 'phone', width: 120 },
  { title: '客户名', dataIndex: 'name', width: 100, ellipsis: true },
  { title: '渠道', dataIndex: 'channel', width: 100, ellipsis: true },
  { title: '邮件任务', dataIndex: 'mail_info', width: 170 },
  { title: '接收时间', dataIndex: 'created_at', width: 165 },
  { title: '操作', dataIndex: 'action', width: 90, fixed: 'right' }
]

const mailColumns = [
  { title: '收件人', dataIndex: 'recipient_name' },
  { title: '邮箱', dataIndex: 'recipient_email', ellipsis: true },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '尝试', dataIndex: 'attempt_count', width: 60 },
  { title: '操作', dataIndex: 'action', width: 80 }
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
    const data = await api.get('/push-records', { params: query })
    list.value = data.list
    total.value = data.total
    pagination.total = data.total
  } finally { loading.value = false }
}

async function view(row) {
  detail.value = await api.get('/push-records/' + row.id)
  detailVisible.value = true
}

function prettyJson(s) {
  try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
}

async function copyMail(row) {
  const d = await api.get(`/mail-records/${row.id}/copy-text`)
  if (!d.text) { message.warning('暂无可复制内容'); return }
  await copyToClipboard(d.text)
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
.section-title { font-weight: bold; margin: 14px 0 8px; }
.json-box { background: #f5f7fa; border: 1px solid #ebeef5; border-radius: 6px; padding: 12px; font-size: 12px; max-height: 260px; overflow: auto; white-space: pre-wrap; word-break: break-all; }
</style>
