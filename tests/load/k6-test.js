// k6 load test for SentinelFlow
// Install: https://k6.io/docs/getting-started/installation/

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const latency = new Trend('latency');

export const options = {
  stages: [
    { duration: '30s', target: 10 },    // Ramp up
    { duration: '1m', target: 50 },     // Stay at 50
    { duration: '1m', target: 100 },    // Peak 100
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],  // 95% under 2s
    errors: ['rate<0.01'],              // < 1% errors
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || '';

export default function () {
  // Test 1: Health check
  let res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health OK': (r) => r.status === 200,
  }) || errorRate.add(1);

  // Test 2: Chat completion
  const payload = JSON.stringify({
    model: 'gpt-3.5-turbo',
    messages: [{ role: 'user', content: 'Hello' }],
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${API_KEY}`,
    },
  };

  const start = Date.now();
  res = http.post(`${BASE_URL}/v1/chat/completions`, payload, params);
  latency.add(Date.now() - start);

  check(res, {
    'chat completions status 200 or 503': (r) => r.status === 200 || r.status === 503,
  }) || errorRate.add(1);

  sleep(1);
}

export function handleSummary(data) {
  return {
    'load-test-summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  return `
${indent}═══════════════════════════════════════
${indent}📊 SentinelFlow Load Test Summary
${indent}═══════════════════════════════════════
${indent}Total requests: ${data.metrics.http_reqs.values.count}
${indent}Request rate: ${data.metrics.http_reqs.values.rate.toFixed(2)}/s
${indent}Error rate: ${(data.metrics.errors?.values.rate * 100 || 0).toFixed(2)}%
${indent}
${indent}Latency:
${indent}  P50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms
${indent}  P95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms
${indent}  P99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms
${indent}═══════════════════════════════════════
`;
}
