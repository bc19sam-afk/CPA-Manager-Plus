export type CodexInspectionTaskTrigger = 'manual' | 'scheduled';

export type CodexInspectionTaskStatus =
  | 'idle'
  | 'queued'
  | 'running'
  | 'success'
  | 'partial'
  | 'failed'
  | 'missed'
  | 'interrupted';

export type CodexInspectionTargetScope =
  | { type: 'all_codex' }
  | { type: 'files'; fileNames: string[] }
  | { type: 'auth_indices'; authIndices: string[] }
  | {
      type: 'metadata_filter';
      query?: string;
      noteIncludes?: string;
      priorityMin?: number;
      priorityMax?: number;
    };

export type CodexInspectionScheduleConfig =
  | { type: 'manual' }
  | { type: 'interval'; every: number; unit: 'minute' | 'hour' | 'day'; timezone?: string }
  | { type: 'daily_times'; times: string[]; timezone?: string };

export interface CodexInspectionExecutionConfig {
  concurrency: number;
  timeoutMs: number;
  retries: number;
}

export type CodexInspectionAutoAction = 'none' | 'disable' | 'enable' | 'delete';

export interface CodexInspectionAutoActionConfig {
  dryRun?: boolean;
  zeroQuotaAction: Exclude<CodexInspectionAutoAction, 'delete'>;
  fullQuotaAction: Exclude<CodexInspectionAutoAction, 'delete'>;
  invalidAction: CodexInspectionAutoAction;
  allowDelete: boolean;
  requireDeletePreview: boolean;
}

export type CodexInspectionNotificationChannel = 'telegram' | 'feishu' | 'wecom' | 'webhook';

export type CodexInspectionNotificationTrigger =
  | 'always'
  | 'abnormal'
  | 'auto_action'
  | 'manual_required';

export interface CodexInspectionNotificationConfig {
  enabled: boolean;
  channels: CodexInspectionNotificationChannel[];
  trigger: CodexInspectionNotificationTrigger;
  channelConfigs?: Record<string, Record<string, unknown>>;
}

export type CodexInspectionLogRetentionConfig =
  | { mode: 'days'; days: number }
  | { mode: 'latest'; count: number }
  | { mode: 'none' };

export interface CodexInspectionTaskPayload {
  name: string;
  description?: string;
  enabled?: boolean;
  targetScope?: CodexInspectionTargetScope;
  schedule?: CodexInspectionScheduleConfig;
  execution?: CodexInspectionExecutionConfig;
  autoAction?: CodexInspectionAutoActionConfig;
  notification?: CodexInspectionNotificationConfig;
  logRetention?: CodexInspectionLogRetentionConfig;
  saveLogs?: boolean;
  dryRun?: boolean;
}

export interface CodexInspectionTask {
  id: string;
  name: string;
  description?: string;
  note?: string;
  createdBy?: string;
  enabled: boolean;
  targetScope: CodexInspectionTargetScope;
  schedule: CodexInspectionScheduleConfig;
  execution: CodexInspectionExecutionConfig;
  autoAction: CodexInspectionAutoActionConfig;
  notification: CodexInspectionNotificationConfig;
  logRetention: CodexInspectionLogRetentionConfig;
  saveLogs: boolean;
  dryRun: boolean;
  status: CodexInspectionTaskStatus | string;
  lastRunId?: string;
  lastRunStatus?: CodexInspectionTaskStatus | string;
  lastRunAtMs?: number;
  nextRunAtMs?: number;
  createdAtMs: number;
  updatedAtMs: number;
}

export interface CodexInspectionRunSummary {
  total?: number;
  healthy?: number;
  fullQuota?: number;
  zeroQuota?: number;
  probeFailed?: number;
  invalid?: number;
  disableCount?: number;
  enableCount?: number;
  deleteCount?: number;
  [key: string]: number | undefined;
}

export interface CodexInspectionRun {
  id: string;
  taskId: string;
  batchId: string;
  trigger: CodexInspectionTaskTrigger | string;
  status: CodexInspectionTaskStatus | string;
  startedAtMs?: number;
  endedAtMs?: number;
  durationMs?: number;
  scheduleSnapshot?: CodexInspectionScheduleConfig;
  targetScopeSnapshot?: CodexInspectionTargetScope;
  executionSnapshot?: CodexInspectionExecutionConfig;
  autoActionSnapshot?: CodexInspectionAutoActionConfig;
  notificationSnapshot?: CodexInspectionNotificationConfig;
  summary?: CodexInspectionRunSummary;
  error?: string;
  createdAtMs: number;
}

export interface CodexInspectionAccountResult {
  id?: number;
  runId: string;
  taskId: string;
  fileName: string;
  authIndex?: string;
  accountId?: string;
  displayAccount?: string;
  provider?: string;
  disabledBefore: boolean;
  status: string;
  statusCode?: number;
  usedPercent?: number;
  classification?: string;
  recommendedAction?: string;
  actionReason?: string;
  error?: string;
  rateLimit?: Record<string, unknown>;
  rawResult?: Record<string, unknown>;
  createdAtMs: number;
}

export interface CodexInspectionActionRecord {
  id?: number;
  taskId: string;
  runId: string;
  fileName: string;
  authIndex?: string;
  action: CodexInspectionAutoAction;
  triggerReason?: string;
  beforeState?: Record<string, unknown>;
  afterState?: Record<string, unknown>;
  dryRun: boolean;
  success: boolean;
  error?: string;
  createdAtMs: number;
}

export interface CodexInspectionNotificationRecord {
  id?: number;
  taskId: string;
  runId: string;
  channel: CodexInspectionNotificationChannel | string;
  status: 'success' | 'failed' | string;
  error?: string;
  responseSummary?: string;
  createdAtMs: number;
}

export interface CodexInspectionTasksResponse {
  tasks: CodexInspectionTask[];
  total: number;
}

export interface CodexInspectionTaskResponse {
  task: CodexInspectionTask;
}

export interface CodexInspectionRunsQuery {
  taskId?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}

export interface CodexInspectionRunsResponse {
  runs: CodexInspectionRun[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CodexInspectionRunResponse {
  run: CodexInspectionRun;
  accounts?: CodexInspectionAccountResult[];
  actions?: CodexInspectionActionRecord[];
  notifications?: CodexInspectionNotificationRecord[];
}

export interface CodexInspectionSchedulerStatus {
  status: string;
  running: boolean;
  [key: string]: unknown;
}
