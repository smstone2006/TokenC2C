<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('c2c.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('c2c.description') }}</p>
      </div>

      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <StatCard :title="t('c2c.available')" :value="money(wallet?.available_balance)" />
        <StatCard :title="t('c2c.pending')" :value="money(wallet?.pending_balance)" />
        <StatCard :title="t('c2c.buyerRequests')" :value="number(stats?.buyer_request_count)" />
        <StatCard :title="t('c2c.buyerSpend')" :value="money(stats?.buyer_spend)" />
        <StatCard :title="t('c2c.sellerRequests')" :value="number(stats?.seller_request_count)" />
        <StatCard :title="t('c2c.sellerEarnings')" :value="money(stats?.seller_earnings)" />
      </div>

      <div class="grid gap-6 lg:grid-cols-[380px_minmax(0,1fr)]">
        <section class="card p-6">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('c2c.addAccount') }}</h2>
          <form class="mt-5 space-y-4" @submit.prevent="createAccount">
            <div>
              <label class="input-label" for="c2c-name">{{ t('c2c.accountName') }}</label>
              <input id="c2c-name" v-model.trim="form.name" class="input mt-1" required maxlength="100" />
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="input-label" for="c2c-platform">Platform</label>
                <select id="c2c-platform" v-model="form.platform" class="input mt-1" @change="normalizeType">
                  <option v-for="platform in platforms" :key="platform" :value="platform">{{ platform }}</option>
                </select>
              </div>
              <div>
                <label class="input-label" for="c2c-type">Type</label>
                <select id="c2c-type" v-model="form.type" class="input mt-1">
                  <option v-for="kind in accountTypes" :key="kind" :value="kind">{{ kind }}</option>
                </select>
              </div>
            </div>
            <div v-if="isCNProvider" class="grid grid-cols-2 gap-3">
              <select v-model="form.account_mode" class="input" aria-label="Account mode">
                <option value="payg">Pay as you go</option>
                <option value="coding">Coding plan</option>
              </select>
              <select v-model="form.api_protocol" class="input" aria-label="API protocol">
                <option value="chat_completions">OpenAI</option>
                <option value="anthropic">Anthropic</option>
              </select>
            </div>
            <div>
              <label class="input-label" for="c2c-credentials">{{ t('c2c.credentials') }}</label>
              <textarea id="c2c-credentials" v-model="form.credentials" class="input mt-1 min-h-32 font-mono text-xs" required spellcheck="false" />
              <p class="input-hint">{{ t('c2c.credentialsHint') }}</p>
            </div>
            <div>
              <label class="input-label" for="c2c-limit">{{ t('c2c.dailyLimit') }}</label>
              <input id="c2c-limit" v-model.number="form.daily_limit_usd" class="input mt-1" type="number" min="0.00000001" step="0.01" />
            </div>
            <button class="btn btn-primary w-full" type="submit" :disabled="saving">
              {{ saving ? '...' : t('c2c.create') }}
            </button>
          </form>
        </section>

        <section class="card overflow-hidden">
          <div class="border-b border-gray-200 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('c2c.mySupply') }}</h2>
          </div>
          <div v-if="loading" class="p-8 text-center text-gray-500">...</div>
          <div v-else-if="accounts.length === 0" class="p-8 text-center text-gray-500">{{ t('c2c.noAccounts') }}</div>
          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <article v-for="account in accounts" :key="account.account_id" class="p-5">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div class="flex items-center gap-2">
                    <h3 class="font-semibold text-gray-900 dark:text-white">{{ account.name }}</h3>
                    <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="statusClass(account.pool_status)">{{ account.pool_status }}</span>
                  </div>
                  <p class="mt-1 text-xs text-gray-500">{{ account.platform }} · {{ account.type }} · #{{ account.account_id }}</p>
                  <p v-if="account.verification_error" class="mt-2 text-sm text-red-600 dark:text-red-400">{{ account.verification_error }}</p>
                  <p v-if="account.daily_limit_usd" class="mt-2 text-xs text-gray-500">
                    {{ money(account.daily_used_usd) }} / {{ money(account.daily_limit_usd) }}
                  </p>
                </div>
                <div class="flex flex-wrap gap-2">
                  <button v-if="account.verification_status !== 'verified'" class="btn btn-secondary btn-sm" @click="verify(account)">{{ t('c2c.verify') }}</button>
                  <button v-else class="btn btn-secondary btn-sm" @click="togglePool(account)">
                    {{ account.pool_status === 'active' ? t('c2c.pause') : t('c2c.resume') }}
                  </button>
                  <button class="btn btn-danger btn-sm" @click="remove(account)">{{ t('c2c.remove') }}</button>
                </div>
              </div>
            </article>
          </div>
        </section>
      </div>

      <section class="card overflow-hidden">
        <div class="border-b border-gray-200 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('c2c.ledger') }}</h2>
        </div>
        <div v-if="ledger.length === 0" class="p-8 text-center text-gray-500">{{ t('c2c.noLedger') }}</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800"><tr><th class="px-5 py-3 text-left">Time</th><th class="px-5 py-3 text-left">Type</th><th class="px-5 py-3 text-left">Wallet</th><th class="px-5 py-3 text-right">Amount</th></tr></thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="entry in ledger" :key="entry.id">
                <td class="whitespace-nowrap px-5 py-3 text-gray-500">{{ new Date(entry.created_at).toLocaleString() }}</td>
                <td class="px-5 py-3 text-gray-700 dark:text-dark-200">{{ entry.entry_type }}</td>
                <td class="px-5 py-3 text-gray-500">{{ entry.wallet }}</td>
                <td class="px-5 py-3 text-right font-mono" :class="entry.amount >= 0 ? 'text-emerald-600' : 'text-red-600'">{{ entry.amount >= 0 ? '+' : '' }}{{ money(entry.amount) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import StatCard from '@/components/common/StatCard.vue'
import { useAppStore } from '@/stores/app'
import { c2cAPI, type C2CLedgerEntry, type C2CProviderAccount, type C2CStats, type C2CWallet } from '@/api/c2c'

const { t } = useI18n()
const appStore = useAppStore()
const platforms = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'kimi', 'zhipu', 'deepseek']
const loading = ref(true)
const saving = ref(false)
const accounts = ref<C2CProviderAccount[]>([])
const wallet = ref<C2CWallet>()
const stats = ref<C2CStats>()
const ledger = ref<C2CLedgerEntry[]>([])
const form = reactive({
  name: '', platform: 'openai', type: 'apikey', credentials: '{\n  "api_key": ""\n}',
  account_mode: 'payg', api_protocol: 'chat_completions', daily_limit_usd: undefined as number | undefined
})

const isCNProvider = computed(() => ['kimi', 'zhipu', 'deepseek'].includes(form.platform))
const accountTypes = computed(() => {
  if (isCNProvider.value) return ['apikey']
  if (form.platform === 'antigravity') return ['oauth']
  if (form.platform === 'anthropic') return ['apikey', 'oauth', 'setup-token', 'bedrock', 'service_account']
  return ['apikey', 'oauth']
})

function normalizeType() {
  if (!accountTypes.value.includes(form.type)) form.type = accountTypes.value[0]
}
function money(value?: number) { return `$${Number(value || 0).toFixed(4)}` }
function number(value?: number) { return Number(value || 0).toLocaleString() }
function statusClass(status: string) {
  if (status === 'active') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'rejected') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

async function load() {
  loading.value = true
  try {
    const [accountRows, walletData, statsData, ledgerRows] = await Promise.all([
      c2cAPI.listAccounts(), c2cAPI.wallet(), c2cAPI.stats(), c2cAPI.ledger()
    ])
    accounts.value = accountRows
    wallet.value = walletData
    stats.value = statsData
    ledger.value = ledgerRows
  } catch (error: any) {
    appStore.showError(error?.message || t('c2c.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function createAccount() {
  let credentials: Record<string, unknown>
  try {
    const parsed = JSON.parse(form.credentials)
    if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') throw new Error('object required')
    credentials = parsed
  } catch {
    appStore.showError(t('c2c.invalidJSON'))
    return
  }
  saving.value = true
  try {
    await c2cAPI.createAccount({
      name: form.name, platform: form.platform, type: form.type, credentials,
      account_mode: isCNProvider.value ? form.account_mode : undefined,
      api_protocol: isCNProvider.value ? form.api_protocol : undefined,
      daily_limit_usd: form.daily_limit_usd || undefined
    })
    form.name = ''
    await load()
  } catch (error: any) {
    appStore.showError(error?.message || t('c2c.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function verify(account: C2CProviderAccount) {
  await c2cAPI.verifyAccount(account.account_id)
  await load()
}
async function togglePool(account: C2CProviderAccount) {
  await c2cAPI.setPoolState(account.account_id, account.pool_status !== 'active')
  await load()
}
async function remove(account: C2CProviderAccount) {
  if (!window.confirm(t('c2c.confirmDelete'))) return
  await c2cAPI.deleteAccount(account.account_id)
  await load()
}

onMounted(load)
</script>
