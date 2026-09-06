import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('../client', () => ({ apiClient: client }))

import { c2cAPI } from '../c2c'

describe('c2cAPI', () => {
  beforeEach(() => vi.clearAllMocks())

  it('creates a user-owned provider account without admin-only fields', async () => {
    const input = {
      name: 'My OpenAI',
      platform: 'openai',
      type: 'apikey',
      credentials: { api_key: 'secret' }
    }
    client.post.mockResolvedValueOnce({ data: { account_id: 7, ...input } })

    const result = await c2cAPI.createAccount(input)

    expect(client.post).toHaveBeenCalledWith('/c2c/provider-accounts', input)
    expect(result.account_id).toBe(7)
    expect(input).not.toHaveProperty('owner_user_id')
    expect(input).not.toHaveProperty('priority')
    expect(input).not.toHaveProperty('group_ids')
  })

  it('uses explicit pool-state and verification endpoints', async () => {
    client.post.mockResolvedValueOnce({ data: { account_id: 7 } })
    client.put.mockResolvedValueOnce({ data: { account_id: 7, pool_status: 'paused' } })

    await c2cAPI.verifyAccount(7)
    await c2cAPI.setPoolState(7, false)

    expect(client.post).toHaveBeenCalledWith('/c2c/provider-accounts/7/verify', {})
    expect(client.put).toHaveBeenCalledWith('/c2c/provider-accounts/7/pool-state', { active: false })
  })

  it('requests the seller and buyer ledger with a bounded limit', async () => {
    client.get.mockResolvedValueOnce({ data: [] })

    await c2cAPI.ledger(25)

    expect(client.get).toHaveBeenCalledWith('/c2c/ledger', { params: { limit: 25 } })
  })
})
