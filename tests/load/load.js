import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

const cacheHits = new Counter('cache_hits');
const cacheMisses = new Counter('cache_misses');
const redirectLatency = new Trend('redirect_latency');
const dbLatency = new Trend('db_latency');
const cacheLatency = new Trend('cache_latency');

const READ_RATIO = 0.9;

const hotUrls = Array.from({length: 100}, (_, i) =>
    `hot_${String(i + 1).padStart(6, '0')}`
);

const warmUrls = Array.from({length: 5000}, (_, i) =>  // Reduced from 10k
    `warm_${String(i + 1).padStart(6, '0')}`
);

function getRandomColdUrl() {
    const id = Math.floor(Math.random() * 9890000) + 1;
    return `cold_${String(id).padStart(7, '0')}`;
}

export const options = {
    stages: [
        { duration: '1m', target: 100 },
        { duration: '2m', target: 500 },
        { duration: '2m', target: 1000 },
        { duration: '2m', target: 250 },
        { duration: '1m', target: 0 },
    ],

    thresholds: {
        'http_req_duration{scenario:hot}': ['p(95)<200', 'p(99)<400'],
        'http_req_duration{scenario:warm}': ['p(95)<600', 'p(99)<1200'],
        'http_req_duration{scenario:cold}': ['p(95)<2000', 'p(99)<3500'],
        'http_req_failed': ['rate<0.08'],
        'http_req_duration': ['p(95)<1500'],
    },

    noConnectionReuse: false,
    userAgent: 'K6LoadTest/1.0',
    batch: 10,
    batchPerHost: 5,
};

export default function () {
    const isRead = Math.random() < READ_RATIO;

    if (isRead) {
        const rand = Math.random();
        let shortcode, scenario;

        if (rand < 0.5) {
            shortcode = hotUrls[Math.floor(Math.random() * hotUrls.length)];
            scenario = 'hot';
        } else if (rand < 0.85) {
            shortcode = warmUrls[Math.floor(Math.random() * warmUrls.length)];
            scenario = 'warm';
        } else {
            shortcode = getRandomColdUrl();
            scenario = 'cold';
        }

        const res = http.get(`http://localhost:8080/${shortcode}`, {
            redirects: 0,
            tags: {
                name: 'redirect',
                scenario: scenario
            },
            timeout: '5s',
        });

        check(res, {
            'redirect success': (r) => r.status === 301 || r.status === 302,
            'not timeout': (r) => r.status !== 0,
        });

        const cacheHit = res.headers['X-Cache-Hit'] === 'true';
        if (cacheHit) {
            cacheHits.add(1);
            cacheLatency.add(res.timings.duration);
        } else {
            cacheMisses.add(1);
            dbLatency.add(res.timings.duration);
        }

        redirectLatency.add(res.timings.duration, { scenario: scenario });

    } else {
        const payload = JSON.stringify({
            url: `https://example.com/page/${Date.now()}-${Math.random()}`,
        });

        const res = http.post('http://localhost:8080/api/shorten', payload, {
            headers: { 'Content-Type': 'application/json' },
            tags: { name: 'shorten' },
            timeout: '10s',
        });

        check(res, {
            'shorten success': (r) => r.status === 200 || r.status === 201,
            'not server error': (r) => r.status < 500,
        });
    }

    sleep(Math.random() * 0.4 + 0.2);  // 0.2-0.6s
}

export function handleSummary(data) {
    const totalRequests = data.metrics.cache_hits.values.count +
        data.metrics.cache_misses.values.count;
    const cacheHitRate = totalRequests > 0
        ? (data.metrics.cache_hits.values.count / totalRequests * 100).toFixed(2)
        : '0.00';

    const avgRedirectLatency = data.metrics.redirect_latency?.values?.avg?.toFixed(2) || 'N/A';
    const p95RedirectLatency = data.metrics.redirect_latency?.values?.['p(95)']?.toFixed(2) || 'N/A';

    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
        'summary.json': JSON.stringify(data, null, 2),
    };
}