import { describe, expect, it } from 'vitest';
import { getDemoApiCallResult } from './demoFixtures';

const CLAUDE_USAGE_URL = 'https://api.anthropic.com/api/oauth/usage';

describe('Claude quota demo fixtures', () => {
  it('provides multiple model-scoped weekly limits for the team account', () => {
    const result = getDemoApiCallResult({
      authIndex: 'claude-team-01',
      url: CLAUDE_USAGE_URL,
    });

    expect(result.body).toMatchObject({
      five_hour: { utilization: 44 },
      seven_day: { utilization: 31 },
      limits: [
        {
          kind: 'weekly_scoped',
          group: 'weekly',
          percent: 78,
          scope: { model: { display_name: 'Fable 5 Max' } },
        },
        {
          kind: 'model_scoped',
          group: 'weekly',
          percent: 42,
          scope: { model: { displayName: 'Demo Model B' } },
        },
      ],
    });
  });

  it('keeps the research account on the legacy payload without limits', () => {
    const result = getDemoApiCallResult({
      authIndex: 'claude-research-02',
      url: CLAUDE_USAGE_URL,
    });

    expect(result.body).toMatchObject({
      five_hour: { utilization: 18 },
      seven_day: { utilization: 22 },
    });
    expect(result.body).not.toHaveProperty('limits');
  });
});
