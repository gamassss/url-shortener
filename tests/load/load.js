import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

const cacheHits = new Counter('cache_hits');
const cacheMisses = new Counter('cache_misses');
const redirectLatency = new Trend('redirect_latency');
const dbLatency = new Trend('db_latency');
const cacheLatency = new Trend('cache_latency');

const READ_RATIO = 0.8;

const hotUrls = Array.from({length: 100}, (_, i) =>
    `hot_${String(i + 1).padStart(6, '0')}`
);

const warmUrls = Array.from({length: 10000}, (_, i) =>
    `warm_${String(i + 1).padStart(6, '0')}`
);

function getRandomColdUrl() {
    const id = Math.floor(Math.random() * 9890000) + 1;
    return `cold_${String(id).padStart(7, '0')}`;
}

export const options = {
    stages: [
        { duration: '10s', target: 100 },
        // { duration: '3m', target: 500 },
        // { duration: '1m', target: 1000 },
        // { duration: '2m', target: 500 },
        // { duration: '1m', target: 0 },
    ],
    thresholds: {
        'http_req_duration{scenario:hot}': ['p(95)<50', 'p(99)<100'],
        'http_req_duration{scenario:warm}': ['p(95)<100', 'p(99)<200'],
        'http_req_duration{scenario:cold}': ['p(95)<500', 'p(99)<1000'],
        http_req_failed: ['rate<0.01'],
    },
};

export default function () {
    const isRead = Math.random() < READ_RATIO;

    if (isRead) {
        const rand = Math.random();
        let shortcode, scenario;

        if (rand < 0.3) {
            shortcode = hotUrls[Math.floor(Math.random() * hotUrls.length)];
            scenario = 'hot';
        } else if (rand < 0.8) {
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
        });

        check(res, {
            'redirect success': (r) => r.status === 301 || r.status === 302,
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
        });

        if (res.status !== 200 && res.status !== 201) {
            console.log('shorten error:', res.status, res.body);
        }

        check(res, {
            'shorten success': (r) => r.status === 200 || r.status === 201,
        });
    }

    sleep(0.1);
}

export function handleSummary(data) {
    const cacheHitRate = (data.metrics.cache_hits.values.count /
        (data.metrics.cache_hits.values.count + data.metrics.cache_misses.values.count) * 100).toFixed(2);

    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
        'summary.json': JSON.stringify(data),
    };
}