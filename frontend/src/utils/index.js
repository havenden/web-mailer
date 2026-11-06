export function copyToClipboard(text) {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text)
    }
  } catch (e) { /* fallback */ }
  return new Promise((resolve, reject) => {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.top = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      resolve()
    } catch (err) {
      reject(err)
    } finally {
      document.body.removeChild(ta)
    }
  })
}

export function fmtTime(t) {
  return t ? t.replace('T', ' ').slice(0, 19) : ''
}

export const statusList = ['待发送', '发送中', '发送成功', '重试等待', '发送失败']
export function statusText(s) {
  return statusList[s] || s
}

const statusTypes = ['warning', 'processing', 'success', 'warning', 'error']
export function statusType(s) {
  return statusTypes[s] || 'default'
}
