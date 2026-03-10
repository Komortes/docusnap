function healthHandler(req, res) {
  return { ok: true, req, res };
}

module.exports = { healthHandler };
