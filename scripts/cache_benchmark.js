import http from 'k6/http';
import { check, group } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

const coldLatency = new Trend('cold_cache_latency_ms');
const warmLatency = new Trend('warm_cache_latency_ms');
const cacheMisses = new Counter('cache_misses');
const cacheHits = new Counter('cache_hits');

export const options = {
  scenarios: {
    cold: {
      executor: 'shared-iterations',
      vus: 100,
      iterations: 500,
      maxDuration: '60s',
      exec: 'coldCache',
    },
    warm: {
      executor: 'shared-iterations',
      vus: 100,
      iterations: 500,
      maxDuration: '60s',
      exec: 'warmCache',
      startTime: '60s',
    },
  },
};

const BASE = __ENV.PRISM_URL || 'http://localhost:8080';

export function coldCache() {
  const randomID = `test-asset-cold-${Math.random().toString(36).substring(7)}`;
  const res = http.get(`${BASE}/manifest/hls/${randomID}`);
  check(res, { 'cold status 200': (r) => r.status === 200 });
  coldLatency.add(res.timings.duration);
  cacheMisses.add(1);
}

export function warmCache() {
  const fixedID = 'test-asset-warm';
  const res = http.get(`${BASE}/manifest/hls/${fixedID}`);
  check(res, { 'warm status 200': (r) => r.status === 200 });
  warmLatency.add(res.timings.duration);
  cacheHits.add(1);
}

export function handleSummary(data) {
  const coldP95 = data.metrics.cold_cache_latency_ms.values['p(95)'];
  const warmP95 = data.metrics.warm_cache_latency_ms.values['p(95)'];
  const reduction = ((coldP95 - warmP95) / coldP95 * 100).toFixed(1);

  const formattedOutput = `Cold cache p95: ${coldP95.toFixed(2)}ms\n` +
                          `Warm cache p95: ${warmP95.toFixed(2)}ms\n` +
                          `Reduction: ${reduction}%\n` +
                          `Origin compute saved: ${reduction}%\n`;

  console.log(formattedOutput);

  return {
    "scripts/cache_benchmark_report.html": htmlReport(data),
    stdout: formattedOutput,
  };
}
