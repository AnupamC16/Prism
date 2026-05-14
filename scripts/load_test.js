import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Trend } from 'k6/metrics';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

const manifestLatency = new Trend('manifest_latency_ms');
const tokenLatency = new Trend('token_validation_latency_ms');
const licenseLatency = new Trend('license_latency_ms');

export const options = {
  scenarios: {
    ramp_up: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 200 },
        { duration: '30s', target: 600 },
        { duration: '60s', target: 600 },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    'http_req_duration{endpoint:manifest}':           ['p(95)<90'],
    'http_req_duration{endpoint:token_validation}':   ['p(95)<18'],
    'http_req_failed':                                ['rate<0.01'],
    'manifest_latency_ms':                            ['p(95)<90'],
    'token_validation_latency_ms':                    ['p(95)<18'],
  },
};

const BASE = __ENV.PRISM_URL || 'http://localhost:8080';

export default function () {
  const assetID = 'test-asset-001';
  
  // Issue token via POST /token (setup at iteration start)
  const tokenReq = http.post(`${BASE}/token`, JSON.stringify({ asset_id: assetID, viewer_id: 'load-test', ttl: 3600 }), {
    headers: { 'Content-Type': 'application/json' },
  });
  check(tokenReq, { 'token issued': (r) => r.status === 200 || r.status === 201 });

  let token = '';
  try {
    const body = tokenReq.json();
    token = (body && body.data && body.data.token) || '';
  } catch (e) {}

  group('manifest', function () {
    const res = http.get(`${BASE}/manifest/hls/${assetID}`, {
      tags: { endpoint: 'manifest' },
    });
    check(res, { 'manifest status is 200': (r) => r.status === 200 });
    manifestLatency.add(res.timings.duration);
  });

  group('license', function () {
    const res = http.post(`${BASE}/license/widevine`, JSON.stringify({ challenge: 'test-challenge' }), {
      headers: {
        'Content-Type': 'application/json',
        'X-DRM-Token': token,
        'X-Asset-ID': assetID,
      },
      tags: { endpoint: 'token_validation' },
    });
    check(res, { 'license status is 200': (r) => r.status === 200 });
    licenseLatency.add(res.timings.duration);
    tokenLatency.add(res.timings.duration); // proxy for token validation latency
  });

  sleep(0.5);
}

export function handleSummary(data) {
  return {
    "scripts/load_test_report.html": htmlReport(data),
    "scripts/summary.json": JSON.stringify(data),
  };
}
