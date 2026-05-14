import { renderToStaticMarkup } from 'react-dom/server';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { CodexInspectionRunDetailPage } from './CodexInspectionRunDetailPage';

const { mocks } = vi.hoisted(() => {
  (globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
    true;
  return {
    mocks: {
      showNotification: vi.fn(),
    },
  };
});

vi.mock('@/stores', () => ({
  useAuthStore: (selector: (state: { apiBase: string; managementKey: string }) => unknown) =>
    selector({ apiBase: 'http://localhost:3000', managementKey: 'mock-key' }),
  useUsageServiceStore: (selector: (state: { enabled: boolean; serviceBase: string }) => unknown) =>
    selector({ enabled: false, serviceBase: '' }),
  useNotificationStore: (
    selector: (state: { showNotification: typeof mocks.showNotification }) => unknown
  ) =>
    selector({
      showNotification: mocks.showNotification,
    }),
}));

describe('CodexInspectionRunDetailPage mock mode', () => {
  beforeEach(() => {
    mocks.showNotification.mockReset();
  });

  it('renders local mock run detail data when query flag is enabled', async () => {
    const html = renderToStaticMarkup(
      <MemoryRouter
        initialEntries={['/monitoring/codex-inspection-tasks/runs/cir_mock_partial_001?mockCodexInspection=1&mockCodexInspectionScenario=default']}
      >
        <Routes>
          <Route
            path="/monitoring/codex-inspection-tasks/runs/:runId"
            element={<CodexInspectionRunDetailPage />}
          />
        </Routes>
      </MemoryRouter>
    );

    expect(html).toContain('Mock 数据模式已启用');
    expect(html).toContain('部分异常');
    expect(html).toContain('账号明细');
    expect(html).toContain('ops-primary@example.com');
    expect(html).toContain('自动操作审计');
    expect(html).toContain('Webhook 返回 502 Bad Gateway');
    expect(mocks.showNotification).not.toHaveBeenCalled();
  });

  it('renders not-found state when missing-run scenario is enabled', async () => {
    const html = renderToStaticMarkup(
      <MemoryRouter
        initialEntries={['/monitoring/codex-inspection-tasks/runs/cir_mock_missing_001?mockCodexInspection=1&mockCodexInspectionScenario=missing-run']}
      >
        <Routes>
          <Route
            path="/monitoring/codex-inspection-tasks/runs/:runId"
            element={<CodexInspectionRunDetailPage />}
          />
        </Routes>
      </MemoryRouter>
    );

    expect(html).toContain('Mock 数据模式已启用');
    expect(html).toContain('未找到执行日志。');
  });
});
