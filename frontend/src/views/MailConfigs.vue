<template>
  <a-tabs default-active-key="configs">
    <a-tab-pane key="configs" tab="邮件配置（QQ 邮箱 SMTP）">
      <a-card :bordered="false">
        <div class="toolbar">
          <a-button type="primary" @click="openForm()">新增配置</a-button>
        </div>
        <a-table :dataSource="configs" :columns="configColumns" :loading="loading" :pagination="false" rowKey="id">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'has_auth_code'">
              <a-tag :color="record.has_auth_code ? 'success' : 'error'">{{ record.has_auth_code ? '已配置' : '未配置' }}</a-tag>
            </template>
            <template v-if="column.dataIndex === 'is_default'">
              <a-tag v-if="record.is_default === 1" color="warning">默认</a-tag>
              <a-button v-else type="link" size="small" @click="setDefault(record)">设为默认</a-button>
            </template>
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="record.status === 1 ? 'success' : 'default'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
            </template>
            <template v-if="column.dataIndex === 'action'">
              <a-space>
                <a-button size="small" @click="openForm(record)">编辑</a-button>
                <a-button size="small" type="primary" @click="openTest(record)">测试</a-button>
                <a-popconfirm title="确认删除该配置？" @confirm="del(record)">
                  <a-button size="small" danger>删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-card>
    </a-tab-pane>

    <a-tab-pane key="templates" tab="邮件模板">
      <a-card :bordered="false">
        <div class="toolbar">
          <a-button type="primary" @click="openTpl()">新增模板</a-button>
        </div>
        <a-table :dataSource="templates" :columns="tplColumns" :pagination="false" rowKey="id">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'is_default'">
              <a-tag v-if="record.is_default === 1" color="warning">默认</a-tag>
            </template>
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="record.status === 1 ? 'success' : 'default'">{{ record.status === 1 ? '启用' : '禁用' }}</a-tag>
            </template>
            <template v-if="column.dataIndex === 'action'">
              <a-space>
                <a-button size="small" @click="openTpl(record)">编辑</a-button>
                <a-popconfirm title="确认删除该模板？" @confirm="delTpl(record)">
                  <a-button size="small" danger>删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </template>
        </a-table>
      </a-card>
    </a-tab-pane>
  </a-tabs>

  <a-modal v-model:open="formVisible" :title="form.id ? '编辑邮件配置' : '新增邮件配置'" @ok="save" :confirm-loading="saving" width="540px">
    <a-form :label-col="{ span: 6 }">
      <a-form-item label="配置名称" required><a-input v-model:value="form.config_name" /></a-form-item>
      <a-form-item label="SMTP 服务器">
        <a-input v-model:value="form.smtp_host" placeholder="默认 smtp.qq.com" />
      </a-form-item>
      <a-form-item label="端口">
        <a-input-number v-model:value="form.smtp_port" :min="1" :max="65535" style="width:120px" />
        <a-radio-group v-model:value="form.encryption" style="margin-left:12px">
          <a-radio value="ssl">SSL（465）</a-radio>
          <a-radio value="starttls">STARTTLS（587）</a-radio>
        </a-radio-group>
      </a-form-item>
      <a-form-item label="发件邮箱" required><a-input v-model:value="form.sender_email" placeholder="QQ 邮箱地址" /></a-form-item>
      <a-form-item label="发件人名称"><a-input v-model:value="form.sender_name" /></a-form-item>
      <a-form-item label="SMTP 账号" required><a-input v-model:value="form.smtp_username" placeholder="通常为完整邮箱地址" /></a-form-item>
      <a-form-item label="SMTP 授权码" :required="!form.id">
        <a-input-password v-model:value="form.auth_code" :placeholder="form.id ? '留空则保留原授权码' : 'QQ 邮箱设置 → 账户 → 生成授权码'" />
      </a-form-item>
      <a-form-item label="设为默认"><a-switch v-model:checked="formIsDefault" /></a-form-item>
      <a-form-item label="状态"><a-switch v-model:checked="formStatusBool" /></a-form-item>
    </a-form>
  </a-modal>

  <a-modal v-model:open="testVisible" title="发送测试邮件" @ok="sendTest" :confirm-loading="testing">
    <a-form :label-col="{ span: 7 }">
      <a-form-item label="测试收件地址" required>
        <a-input v-model:value="testTo" placeholder="填写一个可收信的邮箱" />
      </a-form-item>
    </a-form>
  </a-modal>

  <a-modal v-model:open="tplVisible" :title="tpl.id ? '编辑模板' : '新增模板'" @ok="saveTpl" :confirm-loading="saving" width="760px">
    <a-form :label-col="{ span: 4 }">
      <a-form-item label="模板名称" required><a-input v-model:value="tpl.name" /></a-form-item>
      <a-form-item label="标题模板" required>
        <a-input v-model:value="tpl.subject_template" placeholder="【线索推送】{{ .Region }} - {{ .InterfaceName }}" />
      </a-form-item>
      <a-form-item label="HTML 模板" required>
        <a-textarea v-model:value="tpl.html_template" :rows="16"
          style="font-family:Consolas,monospace;font-size:12px"
          placeholder="支持变量：{{ .Region }} {{ .InterfaceName }} {{ .Sequence }} {{ .Channel }} {{ .Phone }} {{ .Condition }} {{ .Nickname }} {{ .Doctor }} {{ .QualificationAddress }}" />
      </a-form-item>
      <a-form-item label="设为默认"><a-switch v-model:checked="tplIsDefault" /></a-form-item>
    </a-form>
    <template #footer>
      <a-button @click="tplVisible = false">取消</a-button>
      <a-button :loading="previewing" @click="previewTpl">预览效果</a-button>
      <a-button type="primary" :loading="saving" @click="saveTpl">保存</a-button>
    </template>
  </a-modal>

  <a-modal v-model:open="previewVisible" title="模板预览（样例数据）" width="700px">
    <div class="pv-sub">邮件标题：{{ preview.subject }}</div>
    <iframe :srcdoc="preview.html" class="pv-frame"></iframe>
    <div class="pv-sub">复制用文本（序号 → 资质所属地）</div>
    <pre class="pv-pre">{{ preview.copy_text }}</pre>
    <template #footer>
      <a-button @click="copyText(preview.copy_text)">复制这段文本</a-button>
      <a-button type="primary" @click="previewVisible = false">关闭</a-button>
    </template>
  </a-modal>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import api from '../api'
import { copyToClipboard } from '../utils'

const configs = ref([])
const templates = ref([])
const loading = ref(false)
const saving = ref(false)

const configColumns = [
  { title: 'ID', dataIndex: 'id', width: 60 },
  { title: '配置名称', dataIndex: 'config_name' },
  { title: '发件邮箱', dataIndex: 'sender_email', ellipsis: true },
  { title: 'SMTP 服务器', dataIndex: 'smtp_host', width: 140 },
  { title: '端口', dataIndex: 'smtp_port', width: 70 },
  { title: '加密', dataIndex: 'encryption', width: 90 },
  { title: '授权码', dataIndex: 'has_auth_code', width: 90 },
  { title: '默认', dataIndex: 'is_default', width: 80 },
  { title: '状态', dataIndex: 'status', width: 80 },
  { title: '操作', dataIndex: 'action', width: 230, fixed: 'right' }
]

const tplColumns = [
  { title: 'ID', dataIndex: 'id', width: 60 },
  { title: '模板名称', dataIndex: 'name' },
  { title: '标题模板', dataIndex: 'subject_template', ellipsis: true },
  { title: '默认', dataIndex: 'is_default', width: 80 },
  { title: '状态', dataIndex: 'status', width: 80 },
  { title: '操作', dataIndex: 'action', width: 160, fixed: 'right' }
]

const formVisible = ref(false)
const form = reactive({ id: null, config_name: '', smtp_host: 'smtp.qq.com', smtp_port: 465, encryption: 'ssl', sender_email: '', sender_name: '', smtp_username: '', auth_code: '', is_default: 0, status: 1 })

const formIsDefault = computed({ get: () => form.is_default === 1, set: v => { form.is_default = v ? 1 : 0 } })
const formStatusBool = computed({ get: () => form.status === 1, set: v => { form.status = v ? 1 : 0 } })

const testVisible = ref(false)
const testing = ref(false)
const testConfig = ref(null)
const testTo = ref('')

const tplVisible = ref(false)
const tpl = reactive({ id: null, name: '', subject_template: '', html_template: '', is_default: 0 })
const tplIsDefault = computed({ get: () => tpl.is_default === 1, set: v => { tpl.is_default = v ? 1 : 0 } })

const previewVisible = ref(false)
const previewing = ref(false)
const preview = reactive({ subject: '', html: '', copy_text: '' })

async function previewTpl() {
  if (!tpl.html_template) { message.warning('请先填写 HTML 模板'); return }
  previewing.value = true
  try {
    const d = await api.post('/mail/preview', {
      subject_template: tpl.subject_template, html_template: tpl.html_template
    })
    Object.assign(preview, d)
    previewVisible.value = true
  } finally { previewing.value = false }
}

async function copyText(text) {
  if (!text) return
  await copyToClipboard(text)
  message.success('已复制')
}

async function load() {
  loading.value = true
  try {
    configs.value = (await api.get('/mail/configs')).list
    templates.value = (await api.get('/mail/templates')).list
  } finally { loading.value = false }
}

function openForm(row) {
  Object.assign(form, {
    id: row?.id || null, config_name: row?.config_name || '',
    smtp_host: row?.smtp_host || 'smtp.qq.com', smtp_port: row?.smtp_port || 465,
    encryption: row?.encryption || 'ssl', sender_email: row?.sender_email || '',
    sender_name: row?.sender_name || '', smtp_username: row?.smtp_username || '',
    auth_code: '', is_default: row?.is_default ?? 0, status: row?.status ?? 1
  })
  formVisible.value = true
}

async function save() {
  saving.value = true
  try {
    if (form.id) await api.put('/mail/configs/' + form.id, form)
    else await api.post('/mail/configs', form)
    message.success('保存成功')
    formVisible.value = false
    load()
  } finally { saving.value = false }
}

async function setDefault(row) {
  await api.put('/mail/configs/' + row.id, { ...row, is_default: 1, auth_code: '' })
  message.success('已设为默认')
  load()
}

async function del(row) {
  await api.delete('/mail/configs/' + row.id)
  message.success('已删除')
  load()
}

function openTest(row) {
  testConfig.value = row
  testTo.value = ''
  testVisible.value = true
}

async function sendTest() {
  testing.value = true
  try {
    await api.post('/mail/test', { config_id: testConfig.value.id, to: testTo.value })
    testVisible.value = false
  } finally { testing.value = false }
}

function openTpl(row) {
  Object.assign(tpl, {
    id: row?.id || null, name: row?.name || '',
    subject_template: row?.subject_template || '', html_template: row?.html_template || '',
    is_default: row?.is_default ?? 0
  })
  tplVisible.value = true
}

async function saveTpl() {
  if (!tpl.name || !tpl.subject_template || !tpl.html_template) {
    message.warning('模板名称、标题、HTML 内容均必填'); return
  }
  saving.value = true
  try {
    if (tpl.id) await api.put('/mail/templates/' + tpl.id, tpl)
    else await api.post('/mail/templates', tpl)
    message.success('保存成功')
    tplVisible.value = false
    load()
  } finally { saving.value = false }
}

async function delTpl(row) {
  await api.delete('/mail/templates/' + row.id)
  message.success('已删除')
  load()
}

onMounted(load)
</script>

<style scoped>
.toolbar { display: flex; gap: 10px; margin-bottom: 14px; }
.pv-sub { font-weight: bold; margin: 10px 0 8px; font-size: 13px; }
.pv-frame { width: 100%; height: 340px; border: 1px solid #ebeef5; border-radius: 6px; background: #fff; }
.pv-pre {
  margin: 0; padding: 12px 14px; background: #f7f8fa; border: 1px solid #ebeef5;
  border-radius: 6px; font-family: Consolas, Menlo, monospace; font-size: 13px;
  line-height: 1.9; white-space: pre-wrap; word-break: break-all; color: #303133;
}
</style>
