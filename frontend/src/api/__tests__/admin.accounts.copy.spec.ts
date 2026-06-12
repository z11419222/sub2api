import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post,
  },
}))

import { copyAccount } from '@/api/admin/accounts'
import type { Account } from '@/types'

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 31,
    name: 'prod-key (copy)',
    platform: 'openai',
    type: 'apikey',
    credentials: {
      base_url: 'https://api.example.com',
    },
    credentials_status: {
      has_api_key: true,
    },
    proxy_id: null,
    concurrency: 1,
    priority: 50,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-06-11T00:00:00Z',
    updated_at: '2026-06-11T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides,
  }
}

describe('admin accounts copy api', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('posts to the server-side copy endpoint and returns the redacted account', async () => {
    const copied = makeAccount()
    post.mockResolvedValue({ data: copied })

    const result = await copyAccount(3)

    expect(post).toHaveBeenCalledWith('/admin/accounts/3/copy')
    expect(result).toEqual(copied)
    expect(result.credentials).not.toHaveProperty('api_key')
    expect(result.credentials_status?.has_api_key).toBe(true)
  })
})
