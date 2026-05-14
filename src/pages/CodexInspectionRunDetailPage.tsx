import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useLocation, useParams } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import {
  IconChevronLeft,
  IconChevronRight,
  IconCopy,
  IconExternalLink,
  IconEye,
  IconFileText,
  IconRefreshCw,
  IconSend,
  IconShield,
  IconUsers,
  IconWebhook,
} from '@/components/ui/icons';
import {
  isUsageServiceId,
  normalizeUsageServiceBase,
  usageServiceApi,
} from '@/services/api/usageService';
import { useAuthStore, useNotificationStore, useUsageServiceStore } from '@/stores';
import type {
  CodexInspectionAccountResult,
  CodexInspectionActionRecord,
  CodexInspectionRun,
  CodexInspectionRunResponse,
} from '@/types/codexInspectionTask';
import { detectApiBaseFromLocation } from '@/utils/connection';
import {
  findMockCodexInspectionRunDetail,
  isCodexInspectionMockEnabled,
  MOCK_CODEX_INSPECTION_BASE,
  mockCodexInspectionTasks,
} from './codexInspectionMockData';
import styles from './CodexInspectionRunDetailPage.module.scss';

const PAGE_SIZE_OPTIONS = [
  { value: '10', label: '10 条/页' },
  { value: '20', label: '20 条/页' },
  { value: '50', label: '50 条/页' },
];

const STATUS_FILTER_OPTIONS = [
  { value: 'all', label: '全部' },
  { value: 'healthy', label: '正常' },
  { value: 'full_quota', label: '满额度' },
  { value: 'invalid', label: '失效' },
  { value: 'probe_failed', label: '巡检失败' },
];

const ACTION_FILTER_OPTIONS = [
  { value: 'all', label: '全部' },
  { value: 'none', label: '无' },
  { value: 'disable', label: '禁用' },
  { value: 'enable', label: '启用' },
  { value: 'delete', label: '删除' },
];

const pad = (value: number) => String(value).padStart(2, '0');

const formatDateTime = (value?: number) => {
  if (!value) return '--';
  const d = new Date(value);
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
};

const formatDuration = (value?: number) => {
  if (!value || value <= 0) return '--';
  const totalSec = Math.floor(value / 1000);
  if (totalSec < 60) return `${totalSec}秒`;
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  if (sec === 0) return `${min}分`;
  return `${min}分${sec}秒`;
};

const summaryNumber = (run: CodexInspectionRun | null | undefined, key: string) => {
  const value = run?.summary?.[key];
  return typeof value === 'number' ? value : 0;
};

const runStatusLabel = (status?: string) => {
  switch (status) {
    case 'running':
      return '运行中';
    case 'success':
      return '成功';
    case 'partial':
      return '部分异常';
    case 'failed':
      return '失败';
    case 'interrupted':
      return '已中断';
    case 'queued':
      return '排队中';
    case 'missed':
      return '已错过';
    default:
      return status || '未知';
  }
};

const runStatusToneClass = (status?: string) => {
  if (status === 'success') return styles.toneGood;
  if (status === 'partial') return styles.toneWarn;
  if (status === 'failed' || status === 'missed' || status === 'interrupted') return styles.toneBad;
  if (status === 'running' || status === 'queued') return styles.toneInfo;
  return styles.toneMuted;
};

const classificationToneClass = (classification?: string) => {
  if (classification === 'healthy' || classification === 'normal') return styles.toneGood;
  if (classification === 'full_quota' || classification === 'zero_quota') return styles.toneWarn;
  if (classification === 'invalid') return styles.toneBad;
  if (classification === 'probe_failed') return styles.toneBad;
  return styles.toneMuted;
};

const classificationLabel = (value?: string) => {
  switch (value) {
    case 'healthy':
    case 'normal':
      return '正常';
    case 'zero_quota':
      return '零额度';
    case 'full_quota':
      return '满额度';
    case 'invalid':
      return '失效';
    case 'probe_failed':
    case 'failed':
      return '巡检失败';
    case 'unknown':
      return '未知';
    default:
      return value || '--';
  }
};

const triggerLabel = (value?: string) => {
  switch (value) {
    case 'scheduled':
      return '定时任务';
    case 'manual':
      return '手动任务';
    default:
      return value || '--';
  }
};

const actionLabel = (value?: string) => {
  switch (value) {
    case 'disable':
      return '禁用';
    case 'enable':
      return '启用';
    case 'delete':
      return '删除';
    case 'retry':
      return '重试';
    case 'none':
    case '':
      return '无';
    default:
      return value || '无';
  }
};

const channelLabel = (value?: string) => {
  switch (value) {
    case 'telegram':
      return 'Telegram';
    case 'wecom':
      return '企业微信';
    case 'feishu':
      return '飞书';
    case 'webhook':
      return 'Webhook';
    default:
      return value || '--';
  }
};

const ChannelIcon = ({ channel }: { channel?: string }) => {
  switch (channel) {
    case 'telegram':
      return <IconSend size={16} />;
    case 'wecom':
    case 'feishu':
      return <IconUsers size={16} />;
    case 'webhook':
      return <IconWebhook size={16} />;
    default:
      return <IconSend size={16} />;
  }
};

const executionLabelForAccount = (
  account: CodexInspectionAccountResult,
  actions: CodexInspectionActionRecord[]
) => {
  if (account.classification === 'probe_failed') {
    return { label: '跳过', tone: styles.toneMuted };
  }
  const related = actions.find(
    (item) =>
      (account.fileName && item.fileName === account.fileName) ||
      (account.authIndex && item.authIndex === account.authIndex)
  );
  if (!related) return { label: '跳过', tone: styles.toneMuted };
  if (!related.success) return { label: '失败', tone: styles.toneBad };
  switch (related.action) {
    case 'disable':
      return { label: '已禁用', tone: styles.toneWarn };
    case 'enable':
      return { label: '已启用', tone: styles.toneGood };
    case 'delete':
      return { label: '已删除', tone: styles.toneBad };
    default:
      return { label: '已处理', tone: styles.toneInfo };
  }
};

const quotaToneClass = (percent?: number) => {
  if (typeof percent !== 'number') return styles.quotaMuted;
  if (percent >= 100) return styles.quotaDanger;
  if (percent >= 90) return styles.quotaWarn;
  return styles.quotaGood;
};

const accountSearchText = (account: CodexInspectionAccountResult) =>
  [
    account.displayAccount,
    account.accountId,
    account.authIndex,
    account.fileName,
    account.provider,
    account.error,
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase();

const percentText = (value: number, total: number) => {
  if (total <= 0) return '0.0%';
  return `${((value / total) * 100).toFixed(1)}%`;
};

export function CodexInspectionRunDetailPage() {
  const { runId = '' } = useParams();
  const location = useLocation();
  const apiBase = useAuthStore((state) => state.apiBase);
  const managementKey = useAuthStore((state) => state.managementKey);
  const usageServiceEnabled = useUsageServiceStore((state) => state.enabled);
  const usageServiceBase = useUsageServiceStore((state) => state.serviceBase);
  const showNotification = useNotificationStore((state) => state.showNotification);
  const mockModeEnabled = isCodexInspectionMockEnabled(location.search);
  const initialMockDetail = mockModeEnabled && runId ? findMockCodexInspectionRunDetail(runId) : null;

  const [serviceBase, setServiceBase] = useState(() => (mockModeEnabled ? MOCK_CODEX_INSPECTION_BASE : ''));
  const [detail, setDetail] = useState<CodexInspectionRunResponse | null>(() => initialMockDetail);
  const [loading, setLoading] = useState(!mockModeEnabled);
  const [statusFilter, setStatusFilter] = useState('all');
  const [actionFilter, setActionFilter] = useState('all');
  const [keyword, setKeyword] = useState('');
  const [pendingStatusFilter, setPendingStatusFilter] = useState('all');
  const [pendingActionFilter, setPendingActionFilter] = useState('all');
  const [pendingKeyword, setPendingKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const resolveServiceBase = useCallback(async () => {
    if (mockModeEnabled) return MOCK_CODEX_INSPECTION_BASE;
    if (usageServiceEnabled && usageServiceBase) return usageServiceBase;
    const candidates = Array.from(
      new Set(
        [apiBase, detectApiBaseFromLocation()]
          .map((value) => normalizeUsageServiceBase(value || ''))
          .filter(Boolean)
      )
    );
    for (const candidate of candidates) {
      try {
        const info = await usageServiceApi.getInfo(candidate);
        if (isUsageServiceId(info.service)) return candidate;
      } catch {
        // 主服务未代理 Usage Service 时继续尝试下一个候选地址。
      }
    }
    return '';
  }, [apiBase, mockModeEnabled, usageServiceBase, usageServiceEnabled]);

  const loadDetail = useCallback(async () => {
    setLoading(true);
    try {
      const base = await resolveServiceBase();
      setServiceBase(base);
      if (mockModeEnabled) {
        setDetail(runId ? findMockCodexInspectionRunDetail(runId) : null);
        return;
      }
      if (!base || !runId) {
        setDetail(null);
        return;
      }
      const response = await usageServiceApi.getCodexInspectionRun(base, runId, managementKey);
      setDetail(response);
    } catch (err) {
      showNotification(err instanceof Error ? err.message : String(err), 'error');
    } finally {
      setLoading(false);
    }
  }, [managementKey, mockModeEnabled, resolveServiceBase, runId, showNotification]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  const run = detail?.run ?? null;
  const accounts = useMemo(() => detail?.accounts ?? [], [detail?.accounts]);
  const actions = useMemo(() => detail?.actions ?? [], [detail?.actions]);
  const notifications = useMemo(() => detail?.notifications ?? [], [detail?.notifications]);

  const taskName = useMemo(() => {
    if (!run) return '';
    if (mockModeEnabled) {
      const matched = mockCodexInspectionTasks.find((item) => item.id === run.taskId);
      if (matched?.name) return matched.name;
    }
    return run.taskId;
  }, [mockModeEnabled, run]);

  const filteredAccounts = useMemo(() => {
    const normalizedKeyword = keyword.trim().toLowerCase();
    return accounts.filter((account) => {
      const classification = account.classification || account.status || 'unknown';
      const action = account.recommendedAction || 'none';
      if (statusFilter !== 'all' && classification !== statusFilter) return false;
      if (actionFilter !== 'all' && action !== actionFilter) return false;
      if (normalizedKeyword && !accountSearchText(account).includes(normalizedKeyword)) return false;
      return true;
    });
  }, [accounts, actionFilter, keyword, statusFilter]);

  useEffect(() => {
    setPage(1);
  }, [statusFilter, actionFilter, keyword, pageSize]);

  const totalPages = Math.max(1, Math.ceil(filteredAccounts.length / pageSize));
  const safePage = Math.min(page, totalPages);
  const pagedAccounts = useMemo(() => {
    const start = (safePage - 1) * pageSize;
    return filteredAccounts.slice(start, start + pageSize);
  }, [filteredAccounts, pageSize, safePage]);

  const applyFilters = useCallback(() => {
    setStatusFilter(pendingStatusFilter);
    setActionFilter(pendingActionFilter);
    setKeyword(pendingKeyword);
  }, [pendingActionFilter, pendingKeyword, pendingStatusFilter]);

  const resetFilters = useCallback(() => {
    setPendingStatusFilter('all');
    setPendingActionFilter('all');
    setPendingKeyword('');
    setStatusFilter('all');
    setActionFilter('all');
    setKeyword('');
  }, []);

  const handleCopyRunId = useCallback(async () => {
    if (!run?.id) return;
    try {
      await navigator.clipboard.writeText(run.id);
      showNotification('已复制运行 ID', 'success');
    } catch {
      showNotification('复制失败，请手动复制。', 'error');
    }
  }, [run?.id, showNotification]);

  const actionStats = useMemo(() => {
    const calc = (action: CodexInspectionActionRecord['action']) => {
      const subset = actions.filter((item) => item.action === action);
      const success = subset.filter((item) => item.success).length;
      const failed = subset.length - success;
      return { total: subset.length, success, failed };
    };
    const disable = calc('disable');
    const enable = calc('enable');
    const del = calc('delete');
    const failedTotal = actions.filter((item) => !item.success).length;
    return { disable, enable, delete: del, failed: failedTotal };
  }, [actions]);

  const notificationByChannel = useMemo(() => {
    const map = new Map<
      string,
      { success: number; failed: number; errors: string[] }
    >();
    notifications.forEach((record) => {
      const entry = map.get(record.channel) ?? { success: 0, failed: 0, errors: [] };
      if (record.status === 'success') entry.success += 1;
      else entry.failed += 1;
      if (record.error) entry.errors.push(record.error);
      map.set(record.channel, entry);
    });
    return Array.from(map.entries());
  }, [notifications]);

  const total = summaryNumber(run, 'total');
  const healthy = summaryNumber(run, 'healthy');
  const fullQuota = summaryNumber(run, 'fullQuota');
  const invalid = summaryNumber(run, 'invalid');
  const probeFailed = summaryNumber(run, 'probeFailed');

  const statCards = [
    {
      key: 'total',
      label: '巡检账号总数',
      value: total,
      meta: '所有被巡检的账号',
      tone: 'plain' as const,
    },
    {
      key: 'healthy',
      label: '正常',
      value: healthy,
      meta: percentText(healthy, total),
      tone: 'good' as const,
    },
    {
      key: 'fullQuota',
      label: '满额度',
      value: fullQuota,
      meta: percentText(fullQuota, total),
      tone: 'warn' as const,
    },
    {
      key: 'invalid',
      label: '失效',
      value: invalid,
      meta: percentText(invalid, total),
      tone: 'bad' as const,
    },
    {
      key: 'probeFailed',
      label: '巡检失败',
      value: probeFailed,
      meta: percentText(probeFailed, total),
      tone: 'muted' as const,
    },
  ];

  const auditCards = [
    {
      key: 'disable',
      label: '自动禁用',
      value: actionStats.disable.total,
      success: actionStats.disable.success,
      failed: actionStats.disable.failed,
      tone: styles.toneWarn,
    },
    {
      key: 'enable',
      label: '自动启用',
      value: actionStats.enable.total,
      success: actionStats.enable.success,
      failed: actionStats.enable.failed,
      tone: styles.toneGood,
    },
    {
      key: 'delete',
      label: '自动删除',
      value: actionStats.delete.total,
      success: actionStats.delete.success,
      failed: actionStats.delete.failed,
      tone: styles.toneBad,
    },
    {
      key: 'failed',
      label: '操作失败',
      value: actionStats.failed,
      success: 0,
      failed: actionStats.failed,
      tone: styles.toneMuted,
    },
  ];

  const pageNumbers = useMemo(() => {
    const pages: number[] = [];
    const start = Math.max(1, safePage - 2);
    const end = Math.min(totalPages, start + 4);
    for (let i = start; i <= end; i += 1) pages.push(i);
    return pages;
  }, [safePage, totalPages]);

  return (
    <div className={styles.page}>
      <nav className={styles.breadcrumb} aria-label="breadcrumb">
        <Link
          to={{
            pathname: '/monitoring',
            search: mockModeEnabled ? location.search : '',
          }}
        >
          监控中心
        </Link>
        <span className={styles.breadcrumbSep}>/</span>
        <Link
          to={{
            pathname: '/monitoring/codex-inspection-tasks',
            search: mockModeEnabled ? location.search : '',
          }}
        >
          Codex 巡检任务
        </Link>
        <span className={styles.breadcrumbSep}>/</span>
        <span className={styles.breadcrumbCurrent}>{runId || '--'}</span>
      </nav>

      <header className={styles.header}>
        <h1>执行日志详情</h1>
        <Button variant="secondary" onClick={loadDetail} loading={loading}>
          <IconRefreshCw size={16} />
          刷新
        </Button>
      </header>

      {mockModeEnabled ? (
        <Card className={styles.notice}>
          <IconFileText size={20} />
          <div>
            <strong>Mock 数据模式已启用</strong>
            <p>当前详情页使用本地执行日志假数据，不会访问 Usage Service。</p>
          </div>
        </Card>
      ) : null}

      {!serviceBase && !loading && !mockModeEnabled ? (
        <Card className={styles.notice}>
          <IconFileText size={20} />
          <div>
            <strong>Usage Service 未连接</strong>
            <p>执行日志详情需要从 Usage Service 读取，请先确认服务已启用。</p>
          </div>
        </Card>
      ) : null}

      {run ? (
        <>
          <Card className={styles.runSummary}>
            <RunField label="运行 ID">
              <div className={styles.copyRow}>
                <span className={styles.copyValue} title={run.id}>
                  {run.id}
                </span>
                <button
                  type="button"
                  className={styles.copyBtn}
                  onClick={handleCopyRunId}
                  aria-label="复制运行 ID"
                  title="复制运行 ID"
                >
                  <IconCopy size={14} />
                </button>
              </div>
            </RunField>
            <RunField label="任务名称">
              <Link
                to={{
                  pathname: '/monitoring/codex-inspection-tasks',
                  search: mockModeEnabled ? location.search : '',
                }}
                className={styles.taskLink}
                title={taskName}
              >
                <span>{taskName || '--'}</span>
                <IconExternalLink size={13} />
              </Link>
            </RunField>
            <RunField label="触发类型">
              <span>{triggerLabel(run.trigger)}</span>
            </RunField>
            <RunField label="状态">
              <span className={`${styles.statusPill} ${runStatusToneClass(run.status)}`}>
                {runStatusLabel(run.status)}
              </span>
            </RunField>
            <RunField label="开始时间">
              <span className={styles.monoText}>{formatDateTime(run.startedAtMs)}</span>
            </RunField>
            <RunField label="结束时间">
              <span className={styles.monoText}>{formatDateTime(run.endedAtMs)}</span>
            </RunField>
            <RunField label="耗时">
              <span>{formatDuration(run.durationMs)}</span>
            </RunField>
            <RunField label="执行模式">
              <span
                className={`${styles.modeChip} ${
                  run.autoActionSnapshot?.dryRun ? styles.modeChipDry : styles.modeChipReal
                }`}
              >
                {run.autoActionSnapshot?.dryRun ? 'DRY-RUN' : 'REAL'}
              </span>
            </RunField>
          </Card>

          <section className={styles.summaryGrid}>
            {statCards.map((card) => (
              <Card
                key={card.key}
                className={`${styles.metricCard} ${
                  card.tone !== 'plain' ? styles[`metric-${card.tone}`] : ''
                }`}
              >
                <span className={styles.metricLabel}>{card.label}</span>
                <strong className={styles.metricValue}>{card.value}</strong>
                <span className={styles.metricMeta}>{card.meta}</span>
              </Card>
            ))}
          </section>

          <section className={styles.contentGrid}>
            <Card className={styles.accountPanel}>
              <div className={styles.panelHeader}>
                <h2>账号明细</h2>
              </div>
              <div className={styles.filters}>
                <label className={styles.field}>
                  <span>状态</span>
                  <Select
                    value={pendingStatusFilter}
                    onChange={setPendingStatusFilter}
                    options={STATUS_FILTER_OPTIONS}
                    ariaLabel="状态筛选"
                  />
                </label>
                <label className={styles.field}>
                  <span>动作</span>
                  <Select
                    value={pendingActionFilter}
                    onChange={setPendingActionFilter}
                    options={ACTION_FILTER_OPTIONS}
                    ariaLabel="动作筛选"
                  />
                </label>
                <Input
                  label="关键词"
                  value={pendingKeyword}
                  onChange={(event) => setPendingKeyword(event.target.value)}
                  placeholder="邮箱 / 账号名 / IP"
                />
                <div className={styles.filterActions}>
                  <Button variant="secondary" size="sm" onClick={resetFilters}>
                    重置
                  </Button>
                  <Button variant="primary" size="sm" onClick={applyFilters}>
                    筛选
                  </Button>
                </div>
              </div>

              <div className={styles.tableWrap}>
                <table className={styles.table}>
                  <thead>
                    <tr>
                      <th>账号</th>
                      <th>当前状态</th>
                      <th>HTTP 状态</th>
                      <th>配额使用率</th>
                      <th>判定结果</th>
                      <th>建议动作</th>
                      <th>执行结果</th>
                      <th>错误摘要</th>
                      <th className={styles.actionCol}>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {pagedAccounts.length > 0 ? (
                      pagedAccounts.map((account) => {
                        const execution = executionLabelForAccount(account, actions);
                        const percent = account.usedPercent;
                        const errorText = account.error || account.actionReason || '';
                        return (
                          <tr
                            key={
                              account.id ??
                              `${account.runId}-${account.fileName}-${account.authIndex ?? ''}`
                            }
                          >
                            <td className={styles.accountCell}>
                              <strong>{account.displayAccount || account.accountId || account.fileName}</strong>
                              <small>{account.provider || '--'}</small>
                            </td>
                            <td>
                              <span
                                className={`${styles.stateChip} ${
                                  account.disabledBefore ? styles.stateDisabled : styles.stateEnabled
                                }`}
                              >
                                {account.disabledBefore ? '停用' : '启用'}
                              </span>
                            </td>
                            <td className={styles.monoCell}>{account.statusCode ?? '--'}</td>
                            <td>
                              {typeof percent === 'number' ? (
                                <div className={styles.quotaCell}>
                                  <span className={styles.quotaText}>{percent.toFixed(1)}%</span>
                                  <div className={styles.quotaTrack}>
                                    <span
                                      className={`${styles.quotaBar} ${quotaToneClass(percent)}`}
                                      style={{ width: `${Math.min(100, Math.max(0, percent))}%` }}
                                    />
                                  </div>
                                </div>
                              ) : (
                                <span className={styles.dim}>--</span>
                              )}
                            </td>
                            <td>
                              <span
                                className={`${styles.statusPill} ${classificationToneClass(
                                  account.classification
                                )}`}
                              >
                                {classificationLabel(account.classification || account.status)}
                              </span>
                            </td>
                            <td>{actionLabel(account.recommendedAction)}</td>
                            <td>
                              <span className={`${styles.statusPill} ${execution.tone}`}>
                                {execution.label}
                              </span>
                            </td>
                            <td className={styles.errorCell} title={errorText || ''}>
                              {errorText || '--'}
                            </td>
                            <td className={styles.actionCol}>
                              <button type="button" className={styles.detailBtn} disabled>
                                详情
                              </button>
                            </td>
                          </tr>
                        );
                      })
                    ) : (
                      <tr>
                        <td colSpan={9} className={styles.emptyRow}>
                          没有匹配的账号结果
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>

              <div className={styles.pagination}>
                <span className={styles.paginationCount}>共 {filteredAccounts.length} 条</span>
                <div className={styles.paginationControls}>
                  <button
                    type="button"
                    className={styles.pageBtn}
                    onClick={() => setPage((prev) => Math.max(1, prev - 1))}
                    disabled={safePage <= 1}
                    aria-label="上一页"
                  >
                    <IconChevronLeft size={14} />
                  </button>
                  {pageNumbers.map((p) => (
                    <button
                      key={p}
                      type="button"
                      className={`${styles.pageBtn} ${p === safePage ? styles.pageBtnActive : ''}`}
                      onClick={() => setPage(p)}
                    >
                      {p}
                    </button>
                  ))}
                  <button
                    type="button"
                    className={styles.pageBtn}
                    onClick={() => setPage((prev) => Math.min(totalPages, prev + 1))}
                    disabled={safePage >= totalPages}
                    aria-label="下一页"
                  >
                    <IconChevronRight size={14} />
                  </button>
                </div>
                <div className={styles.pageSize}>
                  <Select
                    value={String(pageSize)}
                    onChange={(value) => setPageSize(Number(value) || 10)}
                    options={PAGE_SIZE_OPTIONS}
                    ariaLabel="每页数量"
                  />
                </div>
              </div>
            </Card>

            <aside className={styles.sideColumn}>
              <Card className={styles.sideCard}>
                <h3>本次巡检结果分布</h3>
                <ResultDonut
                  total={total}
                  segments={[
                    { key: 'healthy', label: '正常', value: healthy, color: 'var(--success-color)' },
                    { key: 'fullQuota', label: '满额度', value: fullQuota, color: 'var(--warning-color)' },
                    { key: 'invalid', label: '失效', value: invalid, color: 'var(--error-color)' },
                    { key: 'probeFailed', label: '巡检失败', value: probeFailed, color: '#a855f7' },
                  ]}
                />
              </Card>

              <Card className={styles.sideCard}>
                <h3>通知发送结果</h3>
                <div className={styles.channelList}>
                  {notificationByChannel.length === 0 ? (
                    <div className={styles.emptyRow}>无通知记录</div>
                  ) : (
                    notificationByChannel.map(([channel, stats]) => {
                      const ok = stats.failed === 0;
                      return (
                        <div key={channel} className={styles.channelRow}>
                          <div className={styles.channelMain}>
                            <span className={styles.channelIcon}>
                              <ChannelIcon channel={channel} />
                            </span>
                            <span className={styles.channelName}>{channelLabel(channel)}</span>
                            <span
                              className={`${styles.statusPill} ${ok ? styles.toneGood : styles.toneBad}`}
                            >
                              {ok ? '成功' : '失败'}
                            </span>
                            <span className={styles.channelMeta}>已发送 {stats.success} 条</span>
                            <span className={styles.channelMetaMuted}>失败 {stats.failed} 条</span>
                            <IconChevronRight size={14} />
                          </div>
                          {stats.errors.length > 0 ? (
                            <p className={styles.channelError}>{stats.errors.join('；')}</p>
                          ) : null}
                        </div>
                      );
                    })
                  )}
                </div>
              </Card>

              <Card className={styles.sideCard}>
                <h3>自动操作审计</h3>
                <div className={styles.auditGrid}>
                  {auditCards.map((card) => (
                    <div key={card.key} className={`${styles.auditCard} ${card.tone}`}>
                      <span className={styles.auditLabel}>{card.label}</span>
                      <strong className={styles.auditValue}>{card.value}</strong>
                      <span className={styles.auditMeta}>
                        {card.key === 'failed'
                          ? `失败 ${card.failed} 条`
                          : `成功 ${card.success} / 失败 ${card.failed}`}
                      </span>
                    </div>
                  ))}
                </div>
              </Card>

              <Card className={styles.sideCard}>
                <h3>安全说明</h3>
                <div className={styles.safetyNote}>
                  <IconShield size={18} />
                  <p>
                    未决状态、网络异常和巡检失败账号不会进入自动删除分支，删除动作仍受 dry-run 和 allow delete
                    保护。
                  </p>
                </div>
              </Card>
            </aside>
          </section>
        </>
      ) : loading ? (
        <Card className={styles.emptyState}>
          <span className={styles.spinner} />
          <p>正在加载执行日志...</p>
        </Card>
      ) : (
        <Card className={styles.emptyState}>
          <IconEye size={24} />
          <p>未找到执行日志。</p>
        </Card>
      )}
    </div>
  );
}

function RunField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className={styles.runField}>
      <span className={styles.runFieldLabel}>{label}</span>
      <div className={styles.runFieldValue}>{children}</div>
    </div>
  );
}

interface DonutSegment {
  key: string;
  label: string;
  value: number;
  color: string;
}

function ResultDonut({ total, segments }: { total: number; segments: DonutSegment[] }) {
  const safeTotal = Math.max(
    total,
    segments.reduce((acc, item) => acc + item.value, 0),
    1
  );
  let cursor = 0;
  const stops = segments
    .map((segment) => {
      const start = cursor;
      const end = start + (segment.value / safeTotal) * 100;
      cursor = end;
      return `${segment.color} ${start}% ${end}%`;
    })
    .join(', ');
  const donutStyle = stops
    ? { background: `conic-gradient(${stops}, var(--bg-tertiary) ${cursor}% 100%)` }
    : { background: 'var(--bg-tertiary)' };

  return (
    <div className={styles.donut}>
      <div className={styles.donutRing} style={donutStyle}>
        <div className={styles.donutCenter}>
          <strong>{total}</strong>
          <span>巡检总数</span>
        </div>
      </div>
      <div className={styles.donutLegend}>
        {segments.map((segment) => (
          <div key={segment.key} className={styles.donutLegendRow}>
            <span className={styles.donutDot} style={{ background: segment.color }} />
            <span className={styles.donutLegendLabel}>{segment.label}</span>
            <span className={styles.donutLegendValue}>{segment.value}</span>
            <span className={styles.donutLegendPercent}>({percentText(segment.value, safeTotal)})</span>
          </div>
        ))}
      </div>
    </div>
  );
}
