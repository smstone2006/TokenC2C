import { apiClient } from './client'

export type C2CPoolStatus = 'verifying' | 'active' | 'paused' | 'rejected'
export type C2CVerificationStatus = 'pending' | 'verified' | 'failed'

export interface C2CProviderAccount {
  account_id: number
  name: string
  platform: string
  type: string
  status: string
  schedulable: boolean
  pool_status: C2CPoolStatus
  verification_status: C2CVerificationStatus
  verification_error?: string
  daily_limit_usd?: number
  daily_used_usd: number
  verified_at?: string
  created_at: string
}

export interface C2CCreateProviderAccountInput {
  name: string
  platform: string
  type: string
  credentials: Record<string, unknown>
  account_mode?: string
  api_protocol?: string
  daily_limit_usd?: number
}

export interface C2CWallet {
  available_balance: number
  pending_balance: number
  lifetime_bought: number
  lifetime_earned: number
  lifetime_platform_fees: number
}

export interface C2CStats {
  buyer_request_count: number
  buyer_token_count: number
  buyer_spend: number
  seller_request_count: number
  seller_token_count: number
  seller_pending: number
  seller_settled: number
  seller_earnings: number
}

export interface C2CLedgerEntry {
  id: number
  trade_id?: number
  wallet: 'available' | 'pending' | 'platform'
  entry_type: string
  amount: number
  created_at: string
}

export const c2cAPI = {
  async listAccounts(): Promise<C2CProviderAccount[]> {
    const { data } = await apiClient.get<C2CProviderAccount[]>('/c2c/provider-accounts')
    return data
  },
  async createAccount(input: C2CCreateProviderAccountInput): Promise<C2CProviderAccount> {
    const { data } = await apiClient.post<C2CProviderAccount>('/c2c/provider-accounts', input)
    return data
  },
  async verifyAccount(id: number): Promise<C2CProviderAccount> {
    const { data } = await apiClient.post<C2CProviderAccount>(`/c2c/provider-accounts/${id}/verify`, {})
    return data
  },
  async setPoolState(id: number, active: boolean): Promise<C2CProviderAccount> {
    const { data } = await apiClient.put<C2CProviderAccount>(`/c2c/provider-accounts/${id}/pool-state`, { active })
    return data
  },
  async deleteAccount(id: number): Promise<{ deleted: boolean }> {
    const { data } = await apiClient.delete<{ deleted: boolean }>(`/c2c/provider-accounts/${id}`)
    return data
  },
  async wallet(): Promise<C2CWallet> {
    const { data } = await apiClient.get<C2CWallet>('/c2c/wallet')
    return data
  },
  async stats(): Promise<C2CStats> {
    const { data } = await apiClient.get<C2CStats>('/c2c/stats')
    return data
  },
  async ledger(limit = 50): Promise<C2CLedgerEntry[]> {
    const { data } = await apiClient.get<C2CLedgerEntry[]>('/c2c/ledger', { params: { limit } })
    return data
  }
}
