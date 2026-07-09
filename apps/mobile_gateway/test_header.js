const http = require('http');
const path = require('path');

const projectRoot = process.cwd();
console.log('Project Root:', projectRoot);

try {
    const res = new http.ServerResponse({ method: 'GET' });
    res.setHeader('X-Test-Header', projectRoot);
    console.log('Success: Header set correctly.');
} catch (e) {
    console.error('Error setting header:', e.message);
}
