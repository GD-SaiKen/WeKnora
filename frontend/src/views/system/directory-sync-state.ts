export type DirectorySyncStatus = {
  status: string
  synchronized_users?: number
  error?: string
}

export function syncStatusText(state: DirectorySyncStatus): string {
  if (state.status === 'failed' && state.error) {
    return `同步失败：${state.error}`
  }
  if (state.status === 'succeeded') {
    const count = state.synchronized_users ?? 0
    return `同步成功，已处理 ${count} 人`
  }
  if (state.status === 'running') {
    return '同步进行中…'
  }
  return '尚未同步'
}
