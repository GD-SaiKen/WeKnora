import assert from 'node:assert/strict'
import test from 'node:test'

import { syncStatusText } from '../../../views/system/directory-sync-state.ts'

test('directory-sync-state failed sync shows the server reason', () => {
  assert.equal(
    syncStatusText({ status: 'failed', error: 'AI Hub Platform API 当前不可用', synchronized_users: 3 }),
    '同步失败：AI Hub Platform API 当前不可用',
  )
})
