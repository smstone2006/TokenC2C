import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import * as XLSX from 'xlsx'
import { i18n, loadLocaleMessages } from '@/i18n'
import UsageFilters from '@/components/admin/usage/UsageFilters.vue'
import UsageView from '../UsageView.vue'

// Only API responses and the browser download boundary are replaced. The real
// UsageView event handler, row formatting, Blob and SheetJS writer/reader run.
const { list, exportList, getStats, getSnapshotV2, getModelStats, saveAs, showError, showSuccess } = vi.hoisted(() => ({
  list: vi.fn(),
  exportList: vi.fn(),
  getStats: vi.fn(),
  getSnapshotV2: vi.fn(),
  getModelStats: vi.fn(),
  saveAs: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: { list, getStats },
    dashboard: { getSnapshotV2, getModelStats },
    users: { getById: vi.fn() },
    groups: { list: vi.fn(async () => ({ items: [] })) },
  },
}))
vi.mock('@/api/admin/usage', () => ({ adminUsageAPI: { list: exportList } }))
vi.mock('@/api/admin/ops', () => ({ listErrorLogs: vi.fn() }))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess, showWarning: vi.fn(), showInfo: vi.fn() }),
}))
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }) }))
vi.mock('file-saver', () => ({ saveAs }))

const ProgressStub = {
  props: ['show', 'current', 'total', 'progress'],
  emits: ['cancel'],
  template: '<button v-if="show" data-testid="cancel-export" @click="$emit(\'cancel\')">Cancel</button>',
}

const headerKeys = [
  'usage.time', 'admin.usage.user', 'usage.apiKeyFilter', 'admin.usage.account',
  'usage.requestedModel', 'usage.sentUpstreamModel', 'usage.upstreamResponseModel',
  'usage.upstreamModelMismatch', 'usage.requestedReasoningEffort', 'usage.reasoningEffort',
  'admin.usage.group', 'usage.inboundEndpoint', 'usage.upstreamEndpoint', 'usage.type',
  'admin.usage.inputTokens', 'admin.usage.outputTokens', 'admin.usage.cacheReadTokens',
  'admin.usage.cacheCreationTokens', 'admin.usage.inputCost', 'admin.usage.outputCost',
  'admin.usage.cacheReadCost', 'admin.usage.cacheCreationCost', 'usage.rate',
  'usage.accountMultiplier', 'usage.original', 'usage.userBilled', 'usage.accountBilled',
  'usage.firstToken', 'usage.duration', 'admin.usage.requestId', 'usage.userAgent', 'admin.usage.ipAddress',
]

const usageRows = Array.from({ length: 101 }, (_, index) => ({
  id: index + 1,
  created_at: '2026-10-29T00:00:00Z',
  user: { email: '测试用户@example.test' },
  api_key: { name: '=SUM(1,2)' },
  account: { name: '供应账号 "A" & <B>' },
  model: 'requested-model',
  upstream_model: 'sent-model',
  upstream_response_model: 'response-model',
  upstream_model_mismatch: true,
  reasoning_effort: 'high',
  upstream_reasoning_effort: 'low',
  group: { name: '日本語・中文分组' },
  inbound_endpoint: '/v1/responses',
  upstream_endpoint: '/upstream/responses',
  request_type: 'sync',
  input_tokens: 1234 + index,
  output_tokens: 56,
  cache_read_tokens: 78,
  cache_creation_tokens: 90,
  input_cost: 0.1234567,
  output_cost: 0.2345678,
  cache_read_cost: 0.0000004,
  cache_creation_cost: 0.0000015,
  rate_multiplier: 1.5,
  account_rate_multiplier: 1.25,
  total_cost: 0.3,
  actual_cost: 0.45,
  account_stats_cost: 0.2,
  first_token_ms: 0,
  duration_ms: 1250,
  request_id: String(index + 1).padStart(6, '0'),
  user_agent: '客户端\n"quoted", <&> 😀',
  ip_address: '2001:db8::1',
}))

let wrapper: VueWrapper | undefined
let exportDir: string
const requestedArtifactDir = process.env.TOKENC2C_XLSX_ARTIFACT_DIR

const exportButton = (view: VueWrapper) => view.findComponent(UsageFilters).get('button.btn-primary')

// Vitest uses vue-i18n's runtime-only bundle. These labels have no interpolation;
// wrap the actual locale strings as message functions instead of mocking t().
function staticMessageFunctions(messages: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(Object.entries(messages).map(([key, value]) => [
    key, typeof value === 'string' ? () => value : staticMessageFunctions(value as Record<string, unknown>),
  ]))
}

async function mountExportView() {
  wrapper = mount(UsageView, {
    global: {
      plugins: [i18n],
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        UsageExportProgress: ProgressStub,
        UsageStatsCards: true, UsageTable: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true,
        EndpointDistributionChart: true, UserTokenRanking: true,
        OpsErrorLogTable: true, OpsErrorDetailModal: true,
      },
    },
  })
  await vi.advanceTimersByTimeAsync(120)
  await flushPromises()
  return wrapper
}

function readBlob(blob: Blob): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(reader.error)
    reader.onload = () => resolve(Buffer.from(reader.result as ArrayBuffer))
    reader.readAsArrayBuffer(blob)
  })
}

async function readDownloadedWorkbook(scenario: string) {
  await vi.waitFor(() => expect(saveAs).toHaveBeenCalledTimes(1))
  const [blob, filename] = saveAs.mock.calls[0] as [Blob, string]
  expect(blob.type).toBe('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')
  expect(filename).toMatch(/^usage_\d{4}-\d{2}-\d{2}_to_\d{4}-\d{2}-\d{2}\.xlsx$/)
  const bytes = await readBlob(blob)
  expect(bytes.subarray(0, 2).toString()).toBe('PK')
  const path = join(exportDir, `usage-${scenario}-${XLSX.version}.xlsx`)
  await writeFile(path, bytes)
  const workbook = XLSX.read(await readFile(path), { type: 'buffer', cellNF: true })
  expect(workbook.SheetNames).toEqual(['Usage'])
  expect(showError).not.toHaveBeenCalled()
  expect(showSuccess).toHaveBeenCalledTimes(1)

  // Persist a version-independent semantic representation for the one-off
  // 0.18.5 -> official SheetJS comparison (ZIP metadata can legitimately differ).
  if (requestedArtifactDir) {
    await writeFile(path.replace(/\.xlsx$/, '.json'), JSON.stringify({
      filename,
      sheetNames: workbook.SheetNames,
      sheets: workbook.Sheets,
    }, null, 2))
  }
  return workbook.Sheets.Usage
}

describe('admin UsageView real XLSX export', () => {
  beforeAll(async () => {
    exportDir = requestedArtifactDir || await mkdtemp(join(tmpdir(), 'tokenc2c-usage-xlsx-'))
    await mkdir(exportDir, { recursive: true })
    await loadLocaleMessages('en')
    i18n.global.setLocaleMessage('en', staticMessageFunctions(i18n.global.getLocaleMessage('en')))
    i18n.global.locale.value = 'en'
  })

  beforeEach(() => {
    // Leave setImmediate real: jsdom FileReader dispatches its I/O events with it.
    vi.useFakeTimers({ toFake: ['Date', 'setTimeout', 'clearTimeout'] })
    vi.setSystemTime(new Date('2026-10-29T12:00:00Z'))
    vi.clearAllMocks()
    list.mockResolvedValue({ items: [], total: 101, pages: 2 })
    getStats.mockResolvedValue({})
    getSnapshotV2.mockResolvedValue({ trend: [], models: [], groups: [] })
    getModelStats.mockResolvedValue({ models: [] })
    exportList.mockReset().mockImplementation(async ({ page }: { page: number }) => ({
      items: usageRows.slice((page - 1) * 100, page * 100), total: 101, pages: 2,
    }))
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = undefined
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  afterAll(async () => {
    if (!requestedArtifactDir) await rm(exportDir, { recursive: true, force: true })
  })

  it('pins the verified official release and its tarball integrity', async () => {
    const frontendRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../../..')
    const manifest = JSON.parse(await readFile(join(frontendRoot, 'package.json'), 'utf8'))
    const source = 'https://cdn.sheetjs.com/xlsx-0.20.3/xlsx-0.20.3.tgz'
    expect(manifest.dependencies.xlsx).toBe(source)
    expect(XLSX.version).toBe('0.20.3')
    const lockfile = await readFile(join(frontendRoot, 'pnpm-lock.yaml'), 'utf8')
    const lockedPackage = lockfile.split(`  xlsx@${source}:\n`)[1]?.split('\n\n')[0]
    expect(lockedPackage).toContain(`tarball: ${source}`)
    // pnpm 9 can omit integrity when resolving URL dependencies. Fail the
    // regression suite if a future lockfile rewrite drops this verified pin.
    expect(lockedPackage).toContain('integrity: sha512-oLDq3jw7AcLqKWH2AhCpVTZl8mf6X2YReP+Neh0SJUzV/BdZYjth94tG5toiMB1PPrYtxOCfaoUCkvtuH+3AJA==')
  })

  it('downloads a real paginated workbook with unchanged columns, cell types and values', async () => {
    const view = await mountExportView()
    await exportButton(view).trigger('click')
    const sheet = await readDownloadedWorkbook('paginated')

    expect(exportList).toHaveBeenCalledTimes(2)
    for (const [index, [params, options]] of exportList.mock.calls.entries()) {
      expect(params).toMatchObject({ page: index + 1, page_size: 100, exact_total: true })
      expect(options.signal).toBeInstanceOf(AbortSignal)
    }
    expect(sheet['!ref']).toBe('A1:AF102')
    const expectedRows = usageRows.map((_, index) => [
      '2026-10-29T00:00:00Z', '测试用户@example.test', '=SUM(1,2)', '供应账号 "A" & <B>',
      'requested-model', 'sent-model', 'response-model', 'Yes', 'High', 'Low', '日本語・中文分组',
      '/v1/responses', '/upstream/responses', i18n.global.t('usage.sync'),
      1234 + index, 56, 78, 90, '0.123457', '0.234568', '0.000000', '0.000002',
      '1.500', '1.250', '0.300000', '0.450000', '0.250000', 0, 1250,
      String(index + 1).padStart(6, '0'), '客户端\n"quoted", <&> 😀', '2001:db8::1',
    ])
    expect(XLSX.utils.sheet_to_json(sheet, { header: 1, raw: true, defval: '' })).toEqual([
      headerKeys.map(key => i18n.global.t(key)), ...expectedRows,
    ])
    expect(sheet.C2).toMatchObject({ t: 's', v: '=SUM(1,2)' })
    expect(sheet.C2.f).toBeUndefined()
    expect(sheet.O2).toMatchObject({ t: 'n', v: 1234 })
    expect(sheet.S2).toMatchObject({ t: 's', v: '0.123457' })
    expect(sheet.AD2).toMatchObject({ t: 's', v: '000001' })
    expect(view.findComponent(ProgressStub).props()).toMatchObject({
      show: false, current: 101, total: 101, progress: 100,
    })
    expect(exportButton(view).attributes('disabled')).toBeUndefined()
  })

  it('keeps empty results as a valid workbook with the original 32-column header', async () => {
    exportList.mockResolvedValue({ items: [], total: 0, pages: 0 })
    const view = await mountExportView()
    await exportButton(view).trigger('click')
    const sheet = await readDownloadedWorkbook('empty')
    expect(sheet['!ref']).toBe('A1:AF1')
    expect(XLSX.utils.sheet_to_json(sheet, { header: 1 })).toEqual([
      headerKeys.map(key => i18n.global.t(key)),
    ])
    expect(exportList).toHaveBeenCalledTimes(1)
  })

  it('retains cancellation, progress and duplicate-export protection during pagination', async () => {
    let finishSecondPage!: (value: unknown) => void
    exportList.mockImplementation(({ page }: { page: number }) => page === 1
      ? Promise.resolve({ items: usageRows.slice(0, 100), total: 101, pages: 2 })
      : new Promise(resolve => { finishSecondPage = resolve }))
    const view = await mountExportView()
    await exportButton(view).trigger('click')
    await vi.waitFor(() => expect(exportList).toHaveBeenCalledTimes(2))
    expect(view.findComponent(ProgressStub).props()).toMatchObject({
      show: true, current: 100, total: 101, progress: 99,
    })
    // Emit directly as well as checking the disabled button to exercise the handler guard.
    expect(exportButton(view).attributes('disabled')).toBeDefined()
    view.findComponent(UsageFilters).vm.$emit('export')
    await flushPromises()
    expect(exportList).toHaveBeenCalledTimes(2)
    await view.get('[data-testid="cancel-export"]').trigger('click')
    expect(exportList.mock.calls[1][1].signal.aborted).toBe(true)
    finishSecondPage({ items: usageRows.slice(100), total: 101, pages: 2 })
    await flushPromises()
    expect(saveAs).not.toHaveBeenCalled()
    expect(showSuccess).not.toHaveBeenCalled()
    expect(showError).not.toHaveBeenCalled()
    expect(view.findComponent(ProgressStub).props('show')).toBe(false)
    expect(exportButton(view).attributes('disabled')).toBeUndefined()
  })
})
