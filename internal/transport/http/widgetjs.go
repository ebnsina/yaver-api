package http

// widgetJS is the embeddable chat widget. Merchants add:
//
//	<script src="https://api.yaver.../widget.js" data-key="yvr_pk_..."></script>
//
// It renders a floating launcher + chat panel inside a Shadow DOM (isolated
// from the host page's CSS) and talks to POST /public/chat/messages using the
// publishable key. No dependencies; no backticks (Go raw string).
const widgetJS = `(function () {
  var script = document.currentScript;
  if (!script) return;
  var key = script.getAttribute('data-key');
  if (!key) { console.error('[Yaver] widget: missing data-key'); return; }
  var base = new URL(script.src).origin;
  var convId = null;
  var open = false;
  var cfg = { title: 'Chat with us', welcome: 'Hi! 👋 How can I help you today?', accent: '#111827' };

  var host = document.createElement('div');
  document.body.appendChild(host);
  var root = host.attachShadow({ mode: 'open' });

  var css =
    ':host{all:initial}' +
    '*{box-sizing:border-box;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif}' +
    '.launcher{position:fixed;bottom:20px;right:20px;width:56px;height:56px;border-radius:9999px;background:var(--accent,#111827);color:#fff;border:none;cursor:pointer;display:flex;align-items:center;justify-content:center;box-shadow:0 6px 20px rgba(0,0,0,.18);z-index:2147483000}' +
    '.launcher svg{width:24px;height:24px}' +
    '.panel{position:fixed;bottom:88px;right:20px;width:370px;max-width:calc(100vw - 40px);height:540px;max-height:calc(100vh - 120px);background:#fff;border:1px solid #e5e7eb;border-radius:16px;display:none;flex-direction:column;overflow:hidden;z-index:2147483000;box-shadow:0 12px 40px rgba(0,0,0,.16)}' +
    '.panel.open{display:flex}' +
    '.hd{display:flex;align-items:center;gap:10px;padding:14px 16px;border-bottom:1px solid #f0f0f0}' +
    '.hd .av{width:30px;height:30px;border-radius:8px;background:var(--accent,#111827);color:#fff;font-weight:700;font-size:13px;display:flex;align-items:center;justify-content:center}' +
    '.hd .t{font-size:14px;font-weight:600;color:#111827}' +
    '.hd .s{font-size:12px;color:#9ca3af}' +
    '.msgs{flex:1;overflow-y:auto;padding:16px;display:flex;flex-direction:column;gap:12px;background:#fafafa}' +
    '.b{max-width:82%;padding:9px 12px;font-size:14px;line-height:1.45;border-radius:14px;white-space:pre-wrap}' +
    '.u{align-self:flex-end;background:var(--accent,#111827);color:#fff;border-bottom-right-radius:5px}' +
    '.a{align-self:flex-start;background:#fff;color:#111827;border:1px solid #e5e7eb;border-bottom-left-radius:5px}' +
    '.dots{align-self:flex-start;display:flex;gap:4px;padding:12px}' +
    '.dots i{width:6px;height:6px;border-radius:9999px;background:#9ca3af;animation:bl 1.3s infinite both}' +
    '.dots i:nth-child(2){animation-delay:.2s}.dots i:nth-child(3){animation-delay:.4s}' +
    '@keyframes bl{0%,80%,100%{opacity:.2}40%{opacity:1}}' +
    '.cp{display:flex;gap:8px;padding:10px;border-top:1px solid #f0f0f0}' +
    '.cp input{flex:1;border:1px solid #d1d5db;border-radius:10px;padding:9px 12px;font-size:14px;outline:none}' +
    '.cp input:focus{border-color:#9ca3af}' +
    '.cp button{width:38px;height:38px;border:none;border-radius:10px;background:var(--accent,#111827);color:#fff;cursor:pointer;display:flex;align-items:center;justify-content:center}' +
    '.cp button:disabled{opacity:.4}';

  var chatIcon = '<svg viewBox="0 0 24 24" fill="none"><path d="M4 5h16v11H9l-4 3v-3H4z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/></svg>';
  var sendIcon = '<svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8 13V3m0 0L4 7m4-4 4 4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg>';

  root.innerHTML =
    '<style>' + css + '</style>' +
    '<div class="panel" id="p">' +
      '<div class="hd"><div class="av">Y</div><div><div class="t">Chat with us</div><div class="s">Typically replies instantly</div></div></div>' +
      '<div class="msgs" id="m"></div>' +
      '<form class="cp" id="f"><input id="i" placeholder="Type a message..." autocomplete="off"/><button type="submit" id="b" disabled>' + sendIcon + '</button></form>' +
    '</div>' +
    '<button class="launcher" id="l">' + chatIcon + '</button>';

  var panel = root.getElementById('p');
  var msgs = root.getElementById('m');
  var input = root.getElementById('i');
  var sendBtn = root.getElementById('b');
  var titleEl = root.querySelector('.hd .t');

  function applyCfg() {
    host.style.setProperty('--accent', cfg.accent);
    titleEl.textContent = cfg.title;
  }
  applyCfg();

  // Pull the merchant's branding (title, welcome, accent).
  fetch(base + '/public/chat/config', { headers: { 'X-Yaver-Key': key } })
    .then(function (r) { return r.json(); })
    .then(function (d) {
      if (d && d.title) { cfg = { title: d.title, welcome: d.welcome, accent: d.accent }; applyCfg(); }
    }).catch(function () {});

  function scroll() { msgs.scrollTop = msgs.scrollHeight; }
  function add(role, text) {
    var d = document.createElement('div');
    d.className = 'b ' + (role === 'user' ? 'u' : 'a');
    d.textContent = text;
    msgs.appendChild(d); scroll(); return d;
  }
  function typing() {
    var d = document.createElement('div');
    d.className = 'dots'; d.innerHTML = '<i></i><i></i><i></i>';
    msgs.appendChild(d); scroll(); return d;
  }

  root.getElementById('l').addEventListener('click', function () {
    open = !open;
    panel.classList.toggle('open', open);
    if (open && msgs.childElementCount === 0) {
      add('assistant', cfg.welcome);
      input.focus();
    }
  });
  input.addEventListener('input', function () { sendBtn.disabled = !input.value.trim(); });

  root.getElementById('f').addEventListener('submit', function (e) {
    e.preventDefault();
    var text = input.value.trim();
    if (!text) return;
    input.value = ''; sendBtn.disabled = true;
    add('user', text);
    var t = typing();
    fetch(base + '/public/chat/messages', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Yaver-Key': key },
      body: JSON.stringify({ conversation_id: convId, text: text })
    }).then(function (r) { return r.json(); }).then(function (data) {
      t.remove();
      if (data && data.reply) { convId = data.conversation_id; add('assistant', data.reply); }
      else { add('assistant', 'Sorry, something went wrong.'); }
    }).catch(function () { t.remove(); add('assistant', 'Sorry, I could not reach the server.'); });
  });
})();
`
