const express = require('express');
const { healthHandler } = require('./handlers.js');

const app = express();
app.get('/web/health', healthHandler);
