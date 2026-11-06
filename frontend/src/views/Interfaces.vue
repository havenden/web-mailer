<template>
  <a-card>
    <div class="toolbar">
      <a-input v-model:value="query.name" placeholder="接口名称" allow-clear style="width:160px" @keyup.enter="load" />
      <a-input v-model:value="query.region" placeholder="地域" allow-clear style="width:140px" @keyup.enter="load" />
      <a-input v-model:value="query.channel" placeholder="渠道" allow-clear style="width:140px" @keyup.enter="load" />
      <a-select v-model:value="query.status" placeholder="状态" allow-clear style="width:110px">
        <a-select-option value="1">启用</a-select-option>
        <a-select-option value="0">禁用</a-select-option>
      </a-select>
      <a-button type="primary" @click="load">搜索</a-button>
      <a-button type="primary" @click="openForm()">新增接口</a-button>
    </div>

    <a-table :dataSource="list" :columns="columns" :loading="loading" :pagination="pagination" rowKey="id" @change="onTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'endpoint_url'">
          <div class="url-line">
            <span class="url-text">{{ record.endpoint_url }}</span>
            <a-button type="link" size="small" @click="copyUrl(record)">复制</a-button>
          </div>
        </template>
        <template v-if="column.dataIndex === 'last_push_at'">
          {{ record.last_push_at || '-' }}
        </template>
        <template v-if="column.dataIndex === 'status'">
          <a-tag :color="record.status === 1 ? 'success' : 'default'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
        </template>
        <template v-if="column.dataIndex === 'action'">
          <a-space>
            <a-button size="small" @click="openForm(record)">编辑</a-button>
            <a-button size="small" type="primary" @click="openBind(record)">绑定用户</a-button>
            <a-popconfirm title="确认删除该接口？" @confirm="del(record)">
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>
  </a-card>

  <a-modal v-model:open="formVisible" :title="form.id ? '编辑接口' : '新增接口'" @ok="save" :confirm-loading="saving" width="520px">
    <a-form :label-col="{ span: 6 }">
      <a-form-item label="接口名称" required><a-input v-model:value="form.name" /></a-form-item>
      <a-form-item label="地域"><a-input v-model:value="form.region" placeholder="将显示在邮件【地域】字段" /></a-form-item>
      <a-form-item label="资质所属地"><a-input v-model:value="form.qualification_address" /></a-form-item>
      <a-form-item label="渠道"><a-input v-model:value="form.channel" placeholder="为空时回退推送 JSON 的渠道字段" /></a-form-item>
      <a-form-item label="备注"><a-textarea v-model:value="form.remark" :rows="2" /></a-form-item>
      <a-form-item label="状态">
        <a-switch v-model:checked="formStatusBool" checked-children="启用" un-checked-children="禁用" />
      </a-form-item>
    </a-form>
  </a-modal>

  <a-modal v-model:open="bindVisible" title="绑定用户" @ok="saveBind" width="480px">
    <a-alert type="info" show-icon style="margin-bottom:12px" message="选择该接口接收数据后要推送邮件的用户，可多选" />
    <a-select v-model:value="bindUserIds" mode="multiple" show-search option-filter-prop="label" style="width:100%" placeholder="选择用户" :options="allUsersOptions" />
  </a-modal>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import api from '../api'

const list = ref([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ page: 1, page_size: 20, name: '', region: '', channel: '', status: '' })

const pagination = reactive({
  current: 1, pageSize: 20, total: 0,
  showTotal: t => `共 ${t} 条`, showSizeChanger: false
})

const columns = [
  { title: 'ID', dataIndex: 'id', width: 64 },
  { title: '接口名称', dataIndex: 'name', width: 120, ellipsis: true },
  { title: '接口地址', dataIndex: 'endpoint_url', width: 240 },
  { title: '地域', dataIndex: 'region', width: 90 },
  { title: '渠道', dataIndex: 'channel', width: 100, ellipsis: true },
  { title: '绑定用户', dataIndex: 'user_count', width: 90 },
  { title: '接收量', dataIndex: 'push_count', width: 90 },
  { title: '最近接收', dataIndex: 'last_push_at', width: 160 },
  { title: '状态', dataIndex: 'status', width: 80 },
  { title: '操作', dataIndex: 'action', width: 220, fixed: 'right' }
]

const formVisible = ref(false)
const saving = ref(false)
const form = reactive({ id: null, name: '', region: '', qualification_address: '', channel: '', remark: '', status: 1 })

const formStatusBool = computed({
  get: () => form.status === 1,
  set: v => { form.status = v ? 1 : 0 }
})

const bindVisible = ref(false)
const bindUserIds = ref([])
const bindIface = ref(null)
const allUsers = ref([])

const allUsersOptions = computed(() => allUsers.value.map(u => ({
  value: u.id,
  label: `${u.nickname || u.username}（${u.email}）${u.status === 1 ? '' : ' [已禁用]'}`
})))

function onTableChange(pag) {
  query.page = pag.current
  pagination.current = pag.current
  load()
}

async function load() {
  loading.value = true
  try {
    const data = await api.get('/interfaces', { params: query })
    list.value = data.list
    total.value = data.total
    pagination.total = data.total
  } finally { loading.value = false }
}

function openForm(row) {
  Object.assign(form, {
    id: row?.id || null, name: row?.name || '', region: row?.region || '',
    qualification_address: row?.qualification_address || '', channel: row?.channel || '',
    remark: row?.remark || '', status: row?.status ?? 1
  })
  formVisible.value = true
}

async function save() {
  if (!form.name) { message.warning('接口名称必填'); return }
  saving.value = true
  try {
    if (form.id) await api.put('/interfaces/' + form.id, form)
    else {
      const created = await api.post('/interfaces', { ...form, user_ids: [] })
      message.success('接口创建成功，接口地址：' + created.endpoint_url)
    }
    formVisible.value = false
    load()
  } finally { saving.value = false }
}

async function openBind(row) {
  bindIface.value = row
  bindUserIds.value = []
  const usersData = await api.get('/users', { params: { page: 1, page_size: 100 } })
  allUsers.value = usersData.list
  const detail = await api.get('/interfaces/' + row.id)
  bindUserIds.value = detail.users.map(u => u.id)
  bindVisible.value = true
}

async function saveBind() {
  await api.put(`/interfaces/${bindIface.value.id}/users`, { user_ids: bindUserIds.value })
  message.success('绑定用户已更新')
  bindVisible.value = false
  load()
}

async function copyUrl(row) {
  try {
    await navigator.clipboard.writeText(row.endpoint_url)
    message.success('接口地址已复制')
  } catch {
    message.info(row.endpoint_url)
  }
}

async function del(row) {
  await api.delete('/interfaces/' + row.id)
  message.success('已删除')
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 14px; flex-wrap: wrap; }
.url-line { display: flex; align-items: center; gap: 4px; }
.url-text { font-family: Consolas, monospace; font-size: 12px; word-break: break-all; }
</style>
