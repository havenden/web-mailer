<template>
  <a-card>
    <div class="toolbar">
      <a-input v-model:value="query.keyword" placeholder="用户名 / 昵称 / 邮箱" allow-clear style="width:220px" @keyup.enter="load" />
      <a-button type="primary" @click="load">搜索</a-button>
      <a-button type="primary" @click="openForm()">新增用户</a-button>
    </div>

    <a-table :dataSource="list" :columns="columns" :loading="loading" :pagination="pagination" rowKey="id" @change="onTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'status'">
          <a-tag :color="record.status === 1 ? 'success' : 'default'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
        </template>
        <template v-if="column.dataIndex === 'created_at'">
          {{ fmt(record.created_at) }}
        </template>
        <template v-if="column.dataIndex === 'action'">
          <a-space>
            <a-button size="small" @click="openForm(record)">编辑</a-button>
            <a-button size="small" :type="record.status === 1 ? 'primary' : 'default'" danger @click="toggleStatus(record)">
              {{ record.status === 1 ? '禁用' : '启用' }}
            </a-button>
            <a-popconfirm title="确认删除该用户？" @confirm="del(record)">
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>
  </a-card>

  <a-modal v-model:open="formVisible" :title="form.id ? '编辑用户' : '新增用户'" @ok="save" :confirm-loading="saving" width="480px">
    <a-form :label-col="{ span: 5 }">
      <a-form-item label="用户名" required>
        <a-input v-model:value="form.username" :disabled="!!form.id" />
      </a-form-item>
      <a-form-item label="昵称">
        <a-input v-model:value="form.nickname" placeholder="将显示在邮件【运营】字段" />
      </a-form-item>
      <a-form-item label="邮箱" required>
        <a-input v-model:value="form.email" placeholder="接收推送邮件的邮箱" />
      </a-form-item>
      <a-form-item label="手机号">
        <a-input v-model:value="form.phone" />
      </a-form-item>
      <a-form-item label="角色">
        <a-select v-model:value="form.role">
          <a-select-option value="user">普通用户</a-select-option>
          <a-select-option value="admin">管理员</a-select-option>
        </a-select>
      </a-form-item>
      <a-form-item :label="form.id ? '重置密码' : '初始密码'">
        <a-input-password v-model:value="form.password" :placeholder="form.id ? '留空则不修改' : '默认 123456'" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import api from '../api'
import { fmtTime } from '../utils'

const fmt = fmtTime

const list = ref([])
const total = ref(0)
const loading = ref(false)
const query = reactive({ page: 1, page_size: 20, keyword: '' })

const pagination = reactive({
  current: 1, pageSize: 20, total: 0,
  showTotal: t => `共 ${t} 条`, showSizeChanger: false
})

const columns = [
  { title: 'ID', dataIndex: 'id', width: 64 },
  { title: '用户名', dataIndex: 'username', width: 110 },
  { title: '昵称', dataIndex: 'nickname', width: 100 },
  { title: '邮箱', dataIndex: 'email', width: 180, ellipsis: true },
  { title: '手机号', dataIndex: 'phone', width: 120 },
  { title: '绑定接口', dataIndex: 'interface_count', width: 90 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '创建时间', dataIndex: 'created_at', width: 170 },
  { title: '操作', dataIndex: 'action', width: 240, fixed: 'right' }
]

const formVisible = ref(false)
const saving = ref(false)
const form = reactive({ id: null, username: '', nickname: '', email: '', phone: '', role: 'user', password: '' })

function onTableChange(pag) {
  query.page = pag.current
  pagination.current = pag.current
  load()
}

async function load() {
  loading.value = true
  try {
    const data = await api.get('/users', { params: query })
    list.value = data.list
    total.value = data.total
    pagination.total = data.total
  } finally { loading.value = false }
}

function openForm(row) {
  Object.assign(form, {
    id: row?.id || null,
    username: row?.username || '',
    nickname: row?.nickname || '',
    email: row?.email || '',
    phone: row?.phone || '',
    role: row?.role || 'user',
    password: ''
  })
  formVisible.value = true
}

async function save() {
  if (!form.username || !form.email) { message.warning('用户名和邮箱必填'); return }
  saving.value = true
  try {
    if (form.id) await api.put('/users/' + form.id, form)
    else await api.post('/users', form)
    message.success('保存成功')
    formVisible.value = false
    load()
  } finally { saving.value = false }
}

async function toggleStatus(row) {
  await api.put(`/users/${row.id}/status`, { status: row.status === 1 ? 0 : 1 })
  message.success(row.status === 1 ? '已禁用' : '已启用')
  load()
}

async function del(row) {
  await api.delete('/users/' + row.id)
  message.success('已删除')
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 14px; flex-wrap: wrap; }
</style>
