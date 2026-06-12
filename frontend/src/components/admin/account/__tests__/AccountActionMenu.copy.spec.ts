import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountActionMenu from '../AccountActionMenu.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 3,
    name: 'prod-key',
    platform: 'openai',
    type: 'apikey',
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

describe('AccountActionMenu copy action', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('emits copy for apikey accounts', async () => {
    const account = makeAccount({ type: 'apikey' })
    const wrapper = mount(AccountActionMenu, {
      attachTo: document.body,
      props: {
        show: true,
        account,
        position: { top: 10, left: 10 },
      },
      global: {
        stubs: {
          Icon: true,
          Teleport: true,
        },
      },
    })

    await wrapper.get('button[data-test="copy-account"]').trigger('click')

    expect(wrapper.emitted('copy')?.[0]).toEqual([account])
    expect(wrapper.emitted('close')).toBeTruthy()
    wrapper.unmount()
  })

  it('hides copy for non-apikey accounts', () => {
    const wrapper = mount(AccountActionMenu, {
      attachTo: document.body,
      props: {
        show: true,
        account: makeAccount({ type: 'oauth' }),
        position: { top: 10, left: 10 },
      },
      global: {
        stubs: {
          Icon: true,
          Teleport: true,
        },
      },
    })

    expect(wrapper.find('button[data-test="copy-account"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
