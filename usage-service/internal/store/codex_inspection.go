package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultCodexInspectionTargetScopeJSON  = `{"type":"all_codex"}`
	defaultCodexInspectionScheduleJSON     = `{"type":"manual"}`
	defaultCodexInspectionExecutionJSON    = `{"concurrency":4,"timeoutMs":15000,"retries":0}`
	defaultCodexInspectionAutoActionJSON   = `{"dryRun":true,"zeroQuotaAction":"disable","fullQuotaAction":"disable","invalidAction":"disable","allowDelete":false,"requireDeletePreview":true}`
	defaultCodexInspectionNotificationJSON = `{"enabled":false,"channels":[],"trigger":"auto_action"}`
	defaultCodexInspectionRetentionJSON    = `{"mode":"days","days":30}`
)

var ErrCodexInspectionTaskRunning = errors.New("codex inspection task is already running")

type CodexInspectionTask struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	Enabled       bool            `json:"enabled"`
	TargetScope   json.RawMessage `json:"targetScope"`
	Schedule      json.RawMessage `json:"schedule"`
	Execution     json.RawMessage `json:"execution"`
	AutoAction    json.RawMessage `json:"autoAction"`
	Notification  json.RawMessage `json:"notification"`
	LogRetention  json.RawMessage `json:"logRetention"`
	SaveLogs      bool            `json:"saveLogs"`
	DryRun        bool            `json:"dryRun"`
	Status        string          `json:"status"`
	LastRunID     string          `json:"lastRunId,omitempty"`
	LastRunStatus string          `json:"lastRunStatus,omitempty"`
	LastRunAtMS   *int64          `json:"lastRunAtMs,omitempty"`
	NextRunAtMS   *int64          `json:"nextRunAtMs,omitempty"`
	CreatedAtMS   int64           `json:"createdAtMs"`
	UpdatedAtMS   int64           `json:"updatedAtMs"`
	DeletedAtMS   *int64          `json:"deletedAtMs,omitempty"`
}

type CodexInspectionTaskInput struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Enabled      *bool           `json:"enabled"`
	TargetScope  json.RawMessage `json:"targetScope"`
	Schedule     json.RawMessage `json:"schedule"`
	Execution    json.RawMessage `json:"execution"`
	AutoAction   json.RawMessage `json:"autoAction"`
	Notification json.RawMessage `json:"notification"`
	LogRetention json.RawMessage `json:"logRetention"`
	SaveLogs     *bool           `json:"saveLogs"`
	DryRun       *bool           `json:"dryRun"`
}

type CodexInspectionRun struct {
	ID                   string          `json:"id"`
	TaskID               string          `json:"taskId"`
	BatchID              string          `json:"batchId"`
	Trigger              string          `json:"trigger"`
	Status               string          `json:"status"`
	StartedAtMS          *int64          `json:"startedAtMs,omitempty"`
	EndedAtMS            *int64          `json:"endedAtMs,omitempty"`
	DurationMS           *int64          `json:"durationMs,omitempty"`
	ScheduleSnapshot     json.RawMessage `json:"scheduleSnapshot,omitempty"`
	TargetScopeSnapshot  json.RawMessage `json:"targetScopeSnapshot,omitempty"`
	ExecutionSnapshot    json.RawMessage `json:"executionSnapshot,omitempty"`
	AutoActionSnapshot   json.RawMessage `json:"autoActionSnapshot,omitempty"`
	NotificationSnapshot json.RawMessage `json:"notificationSnapshot,omitempty"`
	Summary              json.RawMessage `json:"summary,omitempty"`
	Error                string          `json:"error,omitempty"`
	CreatedAtMS          int64           `json:"createdAtMs"`
}

type CodexInspectionAccountResult struct {
	ID                int64           `json:"id,omitempty"`
	RunID             string          `json:"runId"`
	TaskID            string          `json:"taskId"`
	FileName          string          `json:"fileName"`
	AuthIndex         string          `json:"authIndex,omitempty"`
	AccountID         string          `json:"accountId,omitempty"`
	DisplayAccount    string          `json:"displayAccount,omitempty"`
	Provider          string          `json:"provider,omitempty"`
	DisabledBefore    bool            `json:"disabledBefore"`
	Status            string          `json:"status"`
	StatusCode        *int64          `json:"statusCode,omitempty"`
	UsedPercent       *float64        `json:"usedPercent,omitempty"`
	Classification    string          `json:"classification,omitempty"`
	RecommendedAction string          `json:"recommendedAction,omitempty"`
	ActionReason      string          `json:"actionReason,omitempty"`
	Error             string          `json:"error,omitempty"`
	RateLimit         json.RawMessage `json:"rateLimit,omitempty"`
	RawResult         json.RawMessage `json:"rawResult,omitempty"`
	CreatedAtMS       int64           `json:"createdAtMs"`
}

type CodexInspectionActionRecord struct {
	ID            int64           `json:"id,omitempty"`
	TaskID        string          `json:"taskId"`
	RunID         string          `json:"runId"`
	FileName      string          `json:"fileName"`
	AuthIndex     string          `json:"authIndex,omitempty"`
	Action        string          `json:"action"`
	TriggerReason string          `json:"triggerReason,omitempty"`
	BeforeState   json.RawMessage `json:"beforeState,omitempty"`
	AfterState    json.RawMessage `json:"afterState,omitempty"`
	DryRun        bool            `json:"dryRun"`
	Success       bool            `json:"success"`
	Error         string          `json:"error,omitempty"`
	CreatedAtMS   int64           `json:"createdAtMs"`
}

type CodexInspectionNotificationRecord struct {
	ID              int64  `json:"id,omitempty"`
	TaskID          string `json:"taskId"`
	RunID           string `json:"runId"`
	Channel         string `json:"channel"`
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
	ResponseSummary string `json:"responseSummary,omitempty"`
	CreatedAtMS     int64  `json:"createdAtMs"`
}

type CodexInspectionCleanupAudit struct {
	ID                    int64           `json:"id,omitempty"`
	TaskID                string          `json:"taskId,omitempty"`
	Policy                json.RawMessage `json:"policy,omitempty"`
	DeletedRuns           int64           `json:"deletedRuns"`
	DeletedAccountResults int64           `json:"deletedAccountResults"`
	DeletedActions        int64           `json:"deletedActions"`
	DeletedNotifications  int64           `json:"deletedNotifications"`
	Error                 string          `json:"error,omitempty"`
	CreatedAtMS           int64           `json:"createdAtMs"`
}

func (s *Store) ensureCodexInspectionTables() error {
	statements := []string{
		`create table if not exists codex_inspection_tasks (
			id text primary key,
			name text not null,
			description text,
			enabled integer not null default 0,
			target_scope_json text not null,
			schedule_json text not null,
			execution_json text not null,
			auto_action_json text not null,
			notification_json text not null,
			log_retention_json text not null,
			save_logs integer not null default 1,
			dry_run integer not null default 1,
			status text not null default 'idle',
			last_run_id text,
			last_run_status text,
			last_run_at_ms integer,
			next_run_at_ms integer,
			created_at_ms integer not null,
			updated_at_ms integer not null,
			deleted_at_ms integer
		)`,
		`create index if not exists idx_codex_inspection_tasks_enabled_next_run
			on codex_inspection_tasks(enabled, next_run_at_ms)
			where deleted_at_ms is null`,
		`create index if not exists idx_codex_inspection_tasks_updated
			on codex_inspection_tasks(updated_at_ms)
			where deleted_at_ms is null`,
		`create table if not exists codex_inspection_runs (
			id text primary key,
			task_id text not null,
			batch_id text not null unique,
			trigger text not null,
			status text not null,
			started_at_ms integer,
			ended_at_ms integer,
			duration_ms integer,
			schedule_snapshot_json text,
			target_scope_snapshot_json text,
			execution_snapshot_json text,
			auto_action_snapshot_json text,
			notification_snapshot_json text,
			summary_json text,
			error text,
			created_at_ms integer not null
		)`,
		`create index if not exists idx_codex_inspection_runs_task_created
			on codex_inspection_runs(task_id, created_at_ms desc)`,
		`create index if not exists idx_codex_inspection_runs_status_created
			on codex_inspection_runs(status, created_at_ms desc)`,
		`create table if not exists codex_inspection_run_accounts (
			id integer primary key autoincrement,
			run_id text not null,
			task_id text not null,
			file_name text not null,
			auth_index text,
			account_id text,
			display_account text,
			provider text,
			disabled_before integer not null default 0,
			status text not null,
			status_code integer,
			used_percent real,
			classification text,
			recommended_action text,
			action_reason text,
			error text,
			rate_limit_json text,
			raw_result_json text,
			created_at_ms integer not null
		)`,
		`create index if not exists idx_codex_inspection_run_accounts_run
			on codex_inspection_run_accounts(run_id)`,
		`create table if not exists codex_inspection_actions (
			id integer primary key autoincrement,
			task_id text not null,
			run_id text not null,
			file_name text not null,
			auth_index text,
			action text not null,
			trigger_reason text,
			before_state_json text,
			after_state_json text,
			dry_run integer not null default 1,
			success integer not null default 0,
			error text,
			created_at_ms integer not null
		)`,
		`create index if not exists idx_codex_inspection_actions_run
			on codex_inspection_actions(run_id)`,
		`create table if not exists codex_inspection_notifications (
			id integer primary key autoincrement,
			task_id text not null,
			run_id text not null,
			channel text not null,
			status text not null,
			error text,
			response_summary text,
			created_at_ms integer not null
		)`,
		`create index if not exists idx_codex_inspection_notifications_run
			on codex_inspection_notifications(run_id)`,
		`create table if not exists codex_inspection_cleanup_audit (
			id integer primary key autoincrement,
			task_id text,
			policy_json text,
			deleted_runs integer not null default 0,
			deleted_account_results integer not null default 0,
			deleted_actions integer not null default 0,
			deleted_notifications integer not null default 0,
			error text,
			created_at_ms integer not null
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	return s.ensureCodexInspectionCleanupAuditColumns()
}

func (s *Store) ensureCodexInspectionCleanupAuditColumns() error {
	rows, err := s.db.Query(`pragma table_info(codex_inspection_cleanup_audit)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	existing := map[string]struct{}{}
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, ok := existing["deleted_actions"]; ok {
		return nil
	}
	_, err = s.db.Exec(`alter table codex_inspection_cleanup_audit add column deleted_actions integer not null default 0`)
	return err
}

func (s *Store) CreateCodexInspectionTask(ctx context.Context, input CodexInspectionTaskInput) (CodexInspectionTask, error) {
	task, err := buildCodexInspectionTaskForSave("", input)
	if err != nil {
		return CodexInspectionTask{}, err
	}
	now := time.Now().UnixMilli()
	task.ID = newCodexInspectionID("cit")
	task.Status = "idle"
	task.CreatedAtMS = now
	task.UpdatedAtMS = now

	_, err = s.db.ExecContext(ctx, `insert into codex_inspection_tasks (
		id, name, description, enabled, target_scope_json, schedule_json, execution_json,
		auto_action_json, notification_json, log_retention_json, save_logs, dry_run,
		status, created_at_ms, updated_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID,
		task.Name,
		nullString(task.Description),
		boolInt(task.Enabled),
		string(task.TargetScope),
		string(task.Schedule),
		string(task.Execution),
		string(task.AutoAction),
		string(task.Notification),
		string(task.LogRetention),
		boolInt(task.SaveLogs),
		boolInt(task.DryRun),
		task.Status,
		task.CreatedAtMS,
		task.UpdatedAtMS,
	)
	if err != nil {
		return CodexInspectionTask{}, err
	}
	return s.GetCodexInspectionTask(ctx, task.ID)
}

func (s *Store) UpdateCodexInspectionTask(ctx context.Context, id string, input CodexInspectionTaskInput) (CodexInspectionTask, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CodexInspectionTask{}, errors.New("task id is required")
	}
	task, err := buildCodexInspectionTaskForSave(id, input)
	if err != nil {
		return CodexInspectionTask{}, err
	}
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `update codex_inspection_tasks set
		name = ?,
		description = ?,
		enabled = ?,
		target_scope_json = ?,
		schedule_json = ?,
		execution_json = ?,
		auto_action_json = ?,
		notification_json = ?,
		log_retention_json = ?,
		save_logs = ?,
		dry_run = ?,
		updated_at_ms = ?
		where id = ? and deleted_at_ms is null`,
		task.Name,
		nullString(task.Description),
		boolInt(task.Enabled),
		string(task.TargetScope),
		string(task.Schedule),
		string(task.Execution),
		string(task.AutoAction),
		string(task.Notification),
		string(task.LogRetention),
		boolInt(task.SaveLogs),
		boolInt(task.DryRun),
		now,
		id,
	)
	if err != nil {
		return CodexInspectionTask{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return CodexInspectionTask{}, sql.ErrNoRows
	}
	return s.GetCodexInspectionTask(ctx, id)
}

func (s *Store) SetCodexInspectionTaskEnabled(ctx context.Context, id string, enabled bool) (CodexInspectionTask, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CodexInspectionTask{}, errors.New("task id is required")
	}
	res, err := s.db.ExecContext(
		ctx,
		`update codex_inspection_tasks set enabled = ?, updated_at_ms = ? where id = ? and deleted_at_ms is null`,
		boolInt(enabled),
		time.Now().UnixMilli(),
		id,
	)
	if err != nil {
		return CodexInspectionTask{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return CodexInspectionTask{}, sql.ErrNoRows
	}
	return s.GetCodexInspectionTask(ctx, id)
}

func (s *Store) UpdateCodexInspectionTaskNextRun(ctx context.Context, id string, nextRunAtMS *int64) (CodexInspectionTask, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CodexInspectionTask{}, errors.New("task id is required")
	}
	res, err := s.db.ExecContext(
		ctx,
		`update codex_inspection_tasks set next_run_at_ms = ?, updated_at_ms = ? where id = ? and deleted_at_ms is null`,
		nullInt(nextRunAtMS),
		time.Now().UnixMilli(),
		id,
	)
	if err != nil {
		return CodexInspectionTask{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return CodexInspectionTask{}, sql.ErrNoRows
	}
	return s.GetCodexInspectionTask(ctx, id)
}

func (s *Store) DeleteCodexInspectionTask(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("task id is required")
	}
	res, err := s.db.ExecContext(
		ctx,
		`update codex_inspection_tasks set enabled = 0, deleted_at_ms = ?, updated_at_ms = ? where id = ? and deleted_at_ms is null`,
		time.Now().UnixMilli(),
		time.Now().UnixMilli(),
		id,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) GetCodexInspectionTask(ctx context.Context, id string) (CodexInspectionTask, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CodexInspectionTask{}, errors.New("task id is required")
	}
	row := s.db.QueryRowContext(ctx, `select
		id, name, description, enabled, target_scope_json, schedule_json, execution_json,
		auto_action_json, notification_json, log_retention_json, save_logs, dry_run,
		status, last_run_id, last_run_status, last_run_at_ms, next_run_at_ms,
		created_at_ms, updated_at_ms, deleted_at_ms
		from codex_inspection_tasks
		where id = ? and deleted_at_ms is null`, id)
	return scanCodexInspectionTask(row)
}

func (s *Store) ListCodexInspectionTasks(ctx context.Context) ([]CodexInspectionTask, error) {
	rows, err := s.db.QueryContext(ctx, `select
		id, name, description, enabled, target_scope_json, schedule_json, execution_json,
		auto_action_json, notification_json, log_retention_json, save_logs, dry_run,
		status, last_run_id, last_run_status, last_run_at_ms, next_run_at_ms,
		created_at_ms, updated_at_ms, deleted_at_ms
		from codex_inspection_tasks
		where deleted_at_ms is null
		order by updated_at_ms desc, created_at_ms desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]CodexInspectionTask, 0)
	for rows.Next() {
		task, err := scanCodexInspectionTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) ListDueCodexInspectionTasks(ctx context.Context, nowMS int64) ([]CodexInspectionTask, error) {
	rows, err := s.db.QueryContext(ctx, `select
		id, name, description, enabled, target_scope_json, schedule_json, execution_json,
		auto_action_json, notification_json, log_retention_json, save_logs, dry_run,
		status, last_run_id, last_run_status, last_run_at_ms, next_run_at_ms,
		created_at_ms, updated_at_ms, deleted_at_ms
		from codex_inspection_tasks
		where deleted_at_ms is null
		  and enabled = 1
		  and next_run_at_ms is not null
		  and next_run_at_ms <= ?
		order by next_run_at_ms asc, updated_at_ms asc`, nowMS)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]CodexInspectionTask, 0)
	for rows.Next() {
		task, err := scanCodexInspectionTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) ListCodexInspectionRuns(ctx context.Context, taskID string, status string, page int, pageSize int) ([]CodexInspectionRun, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	offset := (page - 1) * pageSize
	taskID = strings.TrimSpace(taskID)
	status = strings.TrimSpace(status)

	where := []string{"1 = 1"}
	args := []any{}
	if taskID != "" {
		where = append(where, "task_id = ?")
		args = append(args, taskID)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	whereSQL := strings.Join(where, " and ")

	var total int64
	if err := s.db.QueryRowContext(ctx, `select count(*) from codex_inspection_runs where `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(append([]any{}, args...), pageSize, offset)
	rows, err := s.db.QueryContext(ctx, `select
		id, task_id, batch_id, trigger, status, started_at_ms, ended_at_ms, duration_ms,
		schedule_snapshot_json, target_scope_snapshot_json, execution_snapshot_json,
		auto_action_snapshot_json, notification_snapshot_json, summary_json, error, created_at_ms
		from codex_inspection_runs
		where `+whereSQL+`
		order by created_at_ms desc
		limit ? offset ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	runs := make([]CodexInspectionRun, 0)
	for rows.Next() {
		run, err := scanCodexInspectionRun(rows)
		if err != nil {
			return nil, 0, err
		}
		runs = append(runs, run)
	}
	return runs, total, rows.Err()
}

func (s *Store) GetCodexInspectionRun(ctx context.Context, id string) (CodexInspectionRun, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CodexInspectionRun{}, errors.New("run id is required")
	}
	row := s.db.QueryRowContext(ctx, `select
		id, task_id, batch_id, trigger, status, started_at_ms, ended_at_ms, duration_ms,
		schedule_snapshot_json, target_scope_snapshot_json, execution_snapshot_json,
		auto_action_snapshot_json, notification_snapshot_json, summary_json, error, created_at_ms
		from codex_inspection_runs
		where id = ?`, id)
	return scanCodexInspectionRun(row)
}

func (s *Store) CreateCodexInspectionRun(ctx context.Context, task CodexInspectionTask, trigger string) (CodexInspectionRun, error) {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		trigger = "manual"
	}
	now := time.Now().UnixMilli()
	run := CodexInspectionRun{
		ID:                   newCodexInspectionID("cir"),
		TaskID:               task.ID,
		BatchID:              newCodexInspectionID("cib"),
		Trigger:              trigger,
		Status:               "running",
		StartedAtMS:          &now,
		ScheduleSnapshot:     task.Schedule,
		TargetScopeSnapshot:  task.TargetScope,
		ExecutionSnapshot:    task.Execution,
		AutoActionSnapshot:   task.AutoAction,
		NotificationSnapshot: task.Notification,
		CreatedAtMS:          now,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CodexInspectionRun{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `insert into codex_inspection_runs (
		id, task_id, batch_id, trigger, status, started_at_ms,
		schedule_snapshot_json, target_scope_snapshot_json, execution_snapshot_json,
		auto_action_snapshot_json, notification_snapshot_json, created_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID,
		run.TaskID,
		run.BatchID,
		run.Trigger,
		run.Status,
		now,
		nullString(string(run.ScheduleSnapshot)),
		nullString(string(run.TargetScopeSnapshot)),
		nullString(string(run.ExecutionSnapshot)),
		nullString(string(run.AutoActionSnapshot)),
		nullString(string(run.NotificationSnapshot)),
		run.CreatedAtMS,
	); err != nil {
		return CodexInspectionRun{}, err
	}
	res, err := tx.ExecContext(ctx, `update codex_inspection_tasks set
		status = 'running',
		last_run_id = ?,
		last_run_status = 'running',
		last_run_at_ms = ?,
		updated_at_ms = ?
		where id = ? and deleted_at_ms is null and status != 'running'`,
		run.ID,
		now,
		now,
		task.ID,
	)
	if err != nil {
		return CodexInspectionRun{}, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return CodexInspectionRun{}, ErrCodexInspectionTaskRunning
	}
	if err := tx.Commit(); err != nil {
		return CodexInspectionRun{}, err
	}
	return s.GetCodexInspectionRun(ctx, run.ID)
}

func (s *Store) FinishCodexInspectionRun(ctx context.Context, runID string, status string, summary json.RawMessage, errorMessage string) (CodexInspectionRun, error) {
	runID = strings.TrimSpace(runID)
	status = strings.TrimSpace(status)
	if runID == "" {
		return CodexInspectionRun{}, errors.New("run id is required")
	}
	if status == "" {
		return CodexInspectionRun{}, errors.New("run status is required")
	}
	if len(summary) > 0 && !json.Valid(summary) {
		return CodexInspectionRun{}, errors.New("invalid run summary")
	}

	current, err := s.GetCodexInspectionRun(ctx, runID)
	if err != nil {
		return CodexInspectionRun{}, err
	}
	now := time.Now().UnixMilli()
	var duration *int64
	if current.StartedAtMS != nil {
		value := now - *current.StartedAtMS
		duration = &value
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CodexInspectionRun{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `update codex_inspection_runs set
		status = ?,
		ended_at_ms = ?,
		duration_ms = ?,
		summary_json = ?,
		error = ?
		where id = ?`,
		status,
		now,
		nullInt(duration),
		nullString(string(summary)),
		nullString(errorMessage),
		runID,
	); err != nil {
		return CodexInspectionRun{}, err
	}
	if _, err := tx.ExecContext(ctx, `update codex_inspection_tasks set
		status = ?,
		last_run_status = ?,
		updated_at_ms = ?
		where id = ? and deleted_at_ms is null`,
		status,
		status,
		now,
		current.TaskID,
	); err != nil {
		return CodexInspectionRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return CodexInspectionRun{}, err
	}
	return s.GetCodexInspectionRun(ctx, runID)
}

func (s *Store) InsertCodexInspectionAccountResults(ctx context.Context, results []CodexInspectionAccountResult) error {
	if len(results) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `insert into codex_inspection_run_accounts (
		run_id, task_id, file_name, auth_index, account_id, display_account, provider,
		disabled_before, status, status_code, used_percent, classification, recommended_action,
		action_reason, error, rate_limit_json, raw_result_json, created_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UnixMilli()
	for _, result := range results {
		createdAt := result.CreatedAtMS
		if createdAt <= 0 {
			createdAt = now
		}
		if _, err := stmt.ExecContext(
			ctx,
			result.RunID,
			result.TaskID,
			result.FileName,
			nullString(result.AuthIndex),
			nullString(result.AccountID),
			nullString(result.DisplayAccount),
			nullString(result.Provider),
			boolInt(result.DisabledBefore),
			result.Status,
			nullInt(result.StatusCode),
			nullFloat(result.UsedPercent),
			nullString(result.Classification),
			nullString(result.RecommendedAction),
			nullString(result.ActionReason),
			nullString(result.Error),
			nullString(string(result.RateLimit)),
			nullString(string(result.RawResult)),
			createdAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListCodexInspectionAccountResults(ctx context.Context, runID string) ([]CodexInspectionAccountResult, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, errors.New("run id is required")
	}
	rows, err := s.db.QueryContext(ctx, `select
		id, run_id, task_id, file_name, auth_index, account_id, display_account, provider,
		disabled_before, status, status_code, used_percent, classification, recommended_action,
		action_reason, error, rate_limit_json, raw_result_json, created_at_ms
		from codex_inspection_run_accounts
		where run_id = ?
		order by file_name, display_account, id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]CodexInspectionAccountResult, 0)
	for rows.Next() {
		var result CodexInspectionAccountResult
		var authIndex, accountID, displayAccount, provider, classification, recommendedAction, actionReason, errorText, rateLimit, rawResult sql.NullString
		var disabledBefore int
		var statusCode sql.NullInt64
		var usedPercent sql.NullFloat64
		if err := rows.Scan(
			&result.ID,
			&result.RunID,
			&result.TaskID,
			&result.FileName,
			&authIndex,
			&accountID,
			&displayAccount,
			&provider,
			&disabledBefore,
			&result.Status,
			&statusCode,
			&usedPercent,
			&classification,
			&recommendedAction,
			&actionReason,
			&errorText,
			&rateLimit,
			&rawResult,
			&result.CreatedAtMS,
		); err != nil {
			return nil, err
		}
		result.AuthIndex = authIndex.String
		result.AccountID = accountID.String
		result.DisplayAccount = displayAccount.String
		result.Provider = provider.String
		result.DisabledBefore = disabledBefore != 0
		if statusCode.Valid {
			value := statusCode.Int64
			result.StatusCode = &value
		}
		if usedPercent.Valid {
			value := usedPercent.Float64
			result.UsedPercent = &value
		}
		result.Classification = classification.String
		result.RecommendedAction = recommendedAction.String
		result.ActionReason = actionReason.String
		result.Error = errorText.String
		result.RateLimit = rawJSONOrEmpty(rateLimit.String)
		result.RawResult = rawJSONOrEmpty(rawResult.String)
		results = append(results, result)
	}
	return results, rows.Err()
}

func (s *Store) InsertCodexInspectionActionRecords(ctx context.Context, records []CodexInspectionActionRecord) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `insert into codex_inspection_actions (
		task_id, run_id, file_name, auth_index, action, trigger_reason, before_state_json,
		after_state_json, dry_run, success, error, created_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UnixMilli()
	for _, record := range records {
		createdAt := record.CreatedAtMS
		if createdAt <= 0 {
			createdAt = now
		}
		if _, err := stmt.ExecContext(
			ctx,
			record.TaskID,
			record.RunID,
			record.FileName,
			nullString(record.AuthIndex),
			record.Action,
			nullString(record.TriggerReason),
			nullString(string(record.BeforeState)),
			nullString(string(record.AfterState)),
			boolInt(record.DryRun),
			boolInt(record.Success),
			nullString(record.Error),
			createdAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListCodexInspectionActionRecords(ctx context.Context, runID string) ([]CodexInspectionActionRecord, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, errors.New("run id is required")
	}
	rows, err := s.db.QueryContext(ctx, `select
		id, task_id, run_id, file_name, auth_index, action, trigger_reason,
		before_state_json, after_state_json, dry_run, success, error, created_at_ms
		from codex_inspection_actions
		where run_id = ?
		order by created_at_ms asc, id asc`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]CodexInspectionActionRecord, 0)
	for rows.Next() {
		var record CodexInspectionActionRecord
		var authIndex, triggerReason, beforeState, afterState, errorText sql.NullString
		var dryRun, success int
		if err := rows.Scan(
			&record.ID,
			&record.TaskID,
			&record.RunID,
			&record.FileName,
			&authIndex,
			&record.Action,
			&triggerReason,
			&beforeState,
			&afterState,
			&dryRun,
			&success,
			&errorText,
			&record.CreatedAtMS,
		); err != nil {
			return nil, err
		}
		record.AuthIndex = authIndex.String
		record.TriggerReason = triggerReason.String
		record.BeforeState = rawJSONOrEmpty(beforeState.String)
		record.AfterState = rawJSONOrEmpty(afterState.String)
		record.DryRun = dryRun != 0
		record.Success = success != 0
		record.Error = errorText.String
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) InsertCodexInspectionNotificationRecords(ctx context.Context, records []CodexInspectionNotificationRecord) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `insert into codex_inspection_notifications (
		task_id, run_id, channel, status, error, response_summary, created_at_ms
	) values (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UnixMilli()
	for _, record := range records {
		createdAt := record.CreatedAtMS
		if createdAt <= 0 {
			createdAt = now
		}
		if _, err := stmt.ExecContext(
			ctx,
			record.TaskID,
			record.RunID,
			record.Channel,
			record.Status,
			nullString(record.Error),
			nullString(record.ResponseSummary),
			createdAt,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListCodexInspectionNotificationRecords(ctx context.Context, runID string) ([]CodexInspectionNotificationRecord, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, errors.New("run id is required")
	}
	rows, err := s.db.QueryContext(ctx, `select
		id, task_id, run_id, channel, status, error, response_summary, created_at_ms
		from codex_inspection_notifications
		where run_id = ?
		order by created_at_ms asc, id asc`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]CodexInspectionNotificationRecord, 0)
	for rows.Next() {
		var record CodexInspectionNotificationRecord
		var errorText, responseSummary sql.NullString
		if err := rows.Scan(
			&record.ID,
			&record.TaskID,
			&record.RunID,
			&record.Channel,
			&record.Status,
			&errorText,
			&responseSummary,
			&record.CreatedAtMS,
		); err != nil {
			return nil, err
		}
		record.Error = errorText.String
		record.ResponseSummary = responseSummary.String
		records = append(records, record)
	}
	return records, rows.Err()
}

type codexInspectionRetentionPolicy struct {
	Mode  string `json:"mode"`
	Days  int    `json:"days"`
	Count int    `json:"count"`
}

func (s *Store) CleanupCodexInspectionLogs(ctx context.Context, taskID string, policyRaw json.RawMessage) (CodexInspectionCleanupAudit, error) {
	taskID = strings.TrimSpace(taskID)
	policy := parseCodexInspectionRetentionPolicy(policyRaw)
	audit := CodexInspectionCleanupAudit{
		TaskID:      taskID,
		Policy:      policyRaw,
		CreatedAtMS: time.Now().UnixMilli(),
	}
	if policy.Mode == "none" {
		return s.insertCodexInspectionCleanupAudit(ctx, audit)
	}
	runIDs, err := s.selectCodexInspectionRunsForCleanup(ctx, taskID, policy)
	if err != nil {
		audit.Error = err.Error()
		_, _ = s.insertCodexInspectionCleanupAudit(ctx, audit)
		return audit, err
	}
	if len(runIDs) == 0 {
		return s.insertCodexInspectionCleanupAudit(ctx, audit)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		audit.Error = err.Error()
		_, _ = s.insertCodexInspectionCleanupAudit(ctx, audit)
		return audit, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	where := "run_id in (" + sqlPlaceholders(len(runIDs)) + ")"
	args := stringsToAny(runIDs)
	if res, err := tx.ExecContext(ctx, `delete from codex_inspection_run_accounts where `+where, args...); err != nil {
		audit.Error = err.Error()
		return audit, err
	} else {
		audit.DeletedAccountResults, _ = res.RowsAffected()
	}
	if res, err := tx.ExecContext(ctx, `delete from codex_inspection_actions where `+where, args...); err != nil {
		audit.Error = err.Error()
		return audit, err
	} else {
		audit.DeletedActions, _ = res.RowsAffected()
	}
	if res, err := tx.ExecContext(ctx, `delete from codex_inspection_notifications where `+where, args...); err != nil {
		audit.Error = err.Error()
		return audit, err
	} else {
		audit.DeletedNotifications, _ = res.RowsAffected()
	}
	if res, err := tx.ExecContext(ctx, `delete from codex_inspection_runs where id in (`+sqlPlaceholders(len(runIDs))+`)`, args...); err != nil {
		audit.Error = err.Error()
		return audit, err
	} else {
		audit.DeletedRuns, _ = res.RowsAffected()
	}
	inserted, err := insertCodexInspectionCleanupAuditTx(ctx, tx, audit)
	if err != nil {
		return audit, err
	}
	if err := tx.Commit(); err != nil {
		return audit, err
	}
	return inserted, nil
}

func (s *Store) selectCodexInspectionRunsForCleanup(ctx context.Context, taskID string, policy codexInspectionRetentionPolicy) ([]string, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if taskID != "" {
		where = append(where, "task_id = ?")
		args = append(args, taskID)
	}
	switch policy.Mode {
	case "days":
		if policy.Days <= 0 {
			policy.Days = 30
		}
		cutoff := time.Now().Add(-time.Duration(policy.Days) * 24 * time.Hour).UnixMilli()
		where = append(where, "created_at_ms < ?")
		args = append(args, cutoff)
	case "latest":
		if policy.Count <= 0 {
			policy.Count = 100
		}
		if taskID != "" {
			rows, err := s.db.QueryContext(ctx, `select id from codex_inspection_runs
				where task_id = ?
				  and id not in (
					select id from codex_inspection_runs
					where task_id = ?
					order by created_at_ms desc
					limit ?
				  )
				order by created_at_ms asc`, taskID, taskID, policy.Count)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			return scanStringRows(rows)
		}
		rows, err := s.db.QueryContext(ctx, `select id from codex_inspection_runs
			where id not in (
				select id from codex_inspection_runs order by created_at_ms desc limit ?
			)
			order by created_at_ms asc`, policy.Count)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanStringRows(rows)
	default:
		return nil, fmt.Errorf("unsupported retention mode %q", policy.Mode)
	}
	rows, err := s.db.QueryContext(ctx, `select id from codex_inspection_runs where `+strings.Join(where, " and ")+` order by created_at_ms asc`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStringRows(rows)
}

func (s *Store) insertCodexInspectionCleanupAudit(ctx context.Context, audit CodexInspectionCleanupAudit) (CodexInspectionCleanupAudit, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return audit, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	inserted, err := insertCodexInspectionCleanupAuditTx(ctx, tx, audit)
	if err != nil {
		return audit, err
	}
	if err := tx.Commit(); err != nil {
		return audit, err
	}
	return inserted, nil
}

func insertCodexInspectionCleanupAuditTx(ctx context.Context, tx *sql.Tx, audit CodexInspectionCleanupAudit) (CodexInspectionCleanupAudit, error) {
	if audit.CreatedAtMS <= 0 {
		audit.CreatedAtMS = time.Now().UnixMilli()
	}
	res, err := tx.ExecContext(ctx, `insert into codex_inspection_cleanup_audit (
		task_id, policy_json, deleted_runs, deleted_account_results, deleted_actions,
		deleted_notifications, error, created_at_ms
	) values (?, ?, ?, ?, ?, ?, ?, ?)`,
		nullString(audit.TaskID),
		nullString(string(audit.Policy)),
		audit.DeletedRuns,
		audit.DeletedAccountResults,
		audit.DeletedActions,
		audit.DeletedNotifications,
		nullString(audit.Error),
		audit.CreatedAtMS,
	)
	if err != nil {
		return audit, err
	}
	audit.ID, _ = res.LastInsertId()
	return audit, nil
}

func parseCodexInspectionRetentionPolicy(raw json.RawMessage) codexInspectionRetentionPolicy {
	policy := codexInspectionRetentionPolicy{Mode: "days", Days: 30}
	_ = json.Unmarshal(raw, &policy)
	policy.Mode = strings.ToLower(strings.TrimSpace(policy.Mode))
	if policy.Mode == "" {
		policy.Mode = "days"
	}
	return policy
}

func scanStringRows(rows *sql.Rows) ([]string, error) {
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func sqlPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func stringsToAny(values []string) []any {
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

func (s *Store) MarkStaleCodexInspectionRunsInterrupted(ctx context.Context, reason string) (int64, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "service restarted while task was running"
	}
	now := time.Now().UnixMilli()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	res, err := tx.ExecContext(ctx, `update codex_inspection_runs set
		status = 'interrupted',
		ended_at_ms = ?,
		duration_ms = case
			when started_at_ms is null then null
			when ? >= started_at_ms then ? - started_at_ms
			else 0
		end,
		error = ?
		where status = 'running'`,
		now,
		now,
		now,
		reason,
	)
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	if _, err := tx.ExecContext(ctx, `update codex_inspection_tasks set
		status = 'interrupted',
		last_run_status = 'interrupted',
		updated_at_ms = ?
		where status = 'running' and deleted_at_ms is null`,
		now,
	); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}

type sqlScanner interface {
	Scan(dest ...any) error
}

func scanCodexInspectionTask(scanner sqlScanner) (CodexInspectionTask, error) {
	var task CodexInspectionTask
	var description, lastRunID, lastRunStatus sql.NullString
	var targetScope, schedule, execution, autoAction, notification, logRetention string
	var enabled, saveLogs, dryRun int
	var lastRunAt, nextRunAt, deletedAt sql.NullInt64
	if err := scanner.Scan(
		&task.ID,
		&task.Name,
		&description,
		&enabled,
		&targetScope,
		&schedule,
		&execution,
		&autoAction,
		&notification,
		&logRetention,
		&saveLogs,
		&dryRun,
		&task.Status,
		&lastRunID,
		&lastRunStatus,
		&lastRunAt,
		&nextRunAt,
		&task.CreatedAtMS,
		&task.UpdatedAtMS,
		&deletedAt,
	); err != nil {
		return CodexInspectionTask{}, err
	}
	task.Description = description.String
	task.Enabled = enabled != 0
	task.TargetScope = json.RawMessage(targetScope)
	task.Schedule = json.RawMessage(schedule)
	task.Execution = json.RawMessage(execution)
	task.AutoAction = json.RawMessage(autoAction)
	task.Notification = json.RawMessage(notification)
	task.LogRetention = json.RawMessage(logRetention)
	task.SaveLogs = saveLogs != 0
	task.DryRun = dryRun != 0
	task.LastRunID = lastRunID.String
	task.LastRunStatus = lastRunStatus.String
	if lastRunAt.Valid {
		task.LastRunAtMS = &lastRunAt.Int64
	}
	if nextRunAt.Valid {
		task.NextRunAtMS = &nextRunAt.Int64
	}
	if deletedAt.Valid {
		task.DeletedAtMS = &deletedAt.Int64
	}
	return task, nil
}

func scanCodexInspectionRun(scanner sqlScanner) (CodexInspectionRun, error) {
	var run CodexInspectionRun
	var startedAt, endedAt, duration sql.NullInt64
	var schedule, targetScope, execution, autoAction, notification, summary, errorText sql.NullString
	if err := scanner.Scan(
		&run.ID,
		&run.TaskID,
		&run.BatchID,
		&run.Trigger,
		&run.Status,
		&startedAt,
		&endedAt,
		&duration,
		&schedule,
		&targetScope,
		&execution,
		&autoAction,
		&notification,
		&summary,
		&errorText,
		&run.CreatedAtMS,
	); err != nil {
		return CodexInspectionRun{}, err
	}
	if startedAt.Valid {
		run.StartedAtMS = &startedAt.Int64
	}
	if endedAt.Valid {
		run.EndedAtMS = &endedAt.Int64
	}
	if duration.Valid {
		run.DurationMS = &duration.Int64
	}
	run.ScheduleSnapshot = rawJSONOrEmpty(schedule.String)
	run.TargetScopeSnapshot = rawJSONOrEmpty(targetScope.String)
	run.ExecutionSnapshot = rawJSONOrEmpty(execution.String)
	run.AutoActionSnapshot = rawJSONOrEmpty(autoAction.String)
	run.NotificationSnapshot = rawJSONOrEmpty(notification.String)
	run.Summary = rawJSONOrEmpty(summary.String)
	run.Error = errorText.String
	return run, nil
}

func buildCodexInspectionTaskForSave(id string, input CodexInspectionTaskInput) (CodexInspectionTask, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CodexInspectionTask{}, errors.New("task name is required")
	}
	if len([]rune(name)) > 120 {
		return CodexInspectionTask{}, errors.New("task name is too long")
	}
	description := strings.TrimSpace(input.Description)
	if len([]rune(description)) > 1000 {
		return CodexInspectionTask{}, errors.New("task description is too long")
	}

	targetScope, err := normalizeCodexInspectionJSON(input.TargetScope, defaultCodexInspectionTargetScopeJSON)
	if err != nil {
		return CodexInspectionTask{}, fmt.Errorf("invalid targetScope: %w", err)
	}
	schedule, err := normalizeCodexInspectionJSON(input.Schedule, defaultCodexInspectionScheduleJSON)
	if err != nil {
		return CodexInspectionTask{}, fmt.Errorf("invalid schedule: %w", err)
	}
	execution, err := normalizeCodexInspectionJSON(input.Execution, defaultCodexInspectionExecutionJSON)
	if err != nil {
		return CodexInspectionTask{}, fmt.Errorf("invalid execution: %w", err)
	}
	autoAction, err := normalizeCodexInspectionJSON(input.AutoAction, defaultCodexInspectionAutoActionJSON)
	if err != nil {
		return CodexInspectionTask{}, fmt.Errorf("invalid autoAction: %w", err)
	}
	notification, err := normalizeCodexInspectionJSON(input.Notification, defaultCodexInspectionNotificationJSON)
	if err != nil {
		return CodexInspectionTask{}, fmt.Errorf("invalid notification: %w", err)
	}
	logRetention, err := normalizeCodexInspectionJSON(input.LogRetention, defaultCodexInspectionRetentionJSON)
	if err != nil {
		return CodexInspectionTask{}, fmt.Errorf("invalid logRetention: %w", err)
	}

	return CodexInspectionTask{
		ID:           id,
		Name:         name,
		Description:  description,
		Enabled:      boolValue(input.Enabled, false),
		TargetScope:  targetScope,
		Schedule:     schedule,
		Execution:    execution,
		AutoAction:   autoAction,
		Notification: notification,
		LogRetention: logRetention,
		SaveLogs:     boolValue(input.SaveLogs, true),
		DryRun:       boolValue(input.DryRun, true),
	}, nil
}

func normalizeCodexInspectionJSON(raw json.RawMessage, fallback string) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		trimmed = []byte(fallback)
	}
	if !json.Valid(trimmed) {
		return nil, errors.New("invalid json")
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, err
	}
	return json.RawMessage(compacted.Bytes()), nil
}

func rawJSONOrEmpty(value string) json.RawMessage {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return json.RawMessage(value)
}

func normalizePage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func newCodexInspectionID(prefix string) string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buf[:])
}
