/**
 * k6 Load Test Script - Simple Fixed Concurrency
 * 
 * Usage:
 *   k6 run loadtest.js
 *   k6 run -e TARGET_URL=http://localhost:8080/api loadtest.js
 *   k6 run -e VUS=50 -e DURATION=2m loadtest.js
 */

import http from 'k6/http';
import { sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const TARGET_URL = __ENV.TARGET_URL || 'http://localhost:8080';
const VUS = parseInt(__ENV.VUS) || 100;
const DURATION = __ENV.DURATION || '1m';

export const options = {
    vus: VUS,
    duration: DURATION,
    // 
};

export default function () {
    const headers = {
        'x-request-id': uuidv4(),
    };
    http.get(TARGET_URL, { headers: headers });
    sleep(1);
}
