// k6 load test for yaver-api. Run:
//   k6 run deploy/k6/loadtest.js
//   BASE_URL=https://api.yaver.app PUBLISHABLE_KEY=yvr_pk_… API_KEY=sk_… \
//     k6 run deploy/k6/loadtest.js
//
// Health is always exercised. The public widget and event ingest are exercised
// only when their keys are provided, so the script is safe to run against any
// environment without credentials.
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const PUBLISHABLE_KEY = __ENV.PUBLISHABLE_KEY || '';
const API_KEY = __ENV.API_KEY || '';

export const options = {
	stages: [
		{ duration: '30s', target: 20 }, // ramp up
		{ duration: '1m', target: 20 }, // sustain
		{ duration: '15s', target: 0 } // ramp down
	],
	thresholds: {
		http_req_failed: ['rate<0.01'], // <1% errors
		http_req_duration: ['p(95)<500'] // 95% under 500ms
	}
};

export default function () {
	const health = http.get(`${BASE_URL}/healthz`);
	check(health, { 'healthz 200': (r) => r.status === 200 });

	if (PUBLISHABLE_KEY) {
		const cfg = http.get(`${BASE_URL}/public/chat/config`, {
			headers: { 'X-Yaver-Key': PUBLISHABLE_KEY }
		});
		check(cfg, { 'chat config 200': (r) => r.status === 200 });

		const msg = http.post(
			`${BASE_URL}/public/chat/messages`,
			JSON.stringify({ text: 'load test: where is my order?' }),
			{ headers: { 'Content-Type': 'application/json', 'X-Yaver-Key': PUBLISHABLE_KEY } }
		);
		check(msg, { 'chat send 200': (r) => r.status === 200 });
	}

	if (API_KEY) {
		const ev = http.post(
			`${BASE_URL}/v1/events`,
			JSON.stringify({ type: 'order.placed', phone: '01712345678', order: { id: 'k6', amount: 1000 } }),
			{ headers: { 'Content-Type': 'application/json', 'X-API-Key': API_KEY } }
		);
		check(ev, { 'ingest 2xx': (r) => r.status >= 200 && r.status < 300 });
	}

	sleep(1);
}
