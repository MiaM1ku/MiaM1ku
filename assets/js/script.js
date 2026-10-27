/* =========================================================================
   Nya 的学习笔记 · script.js
   -------------------------------------------------------------------------
   结构：每个功能一个 init 函数，各自查询自己的节点、各自加事件，
        找不到节点就安静退出（局部失效不会拖垮整页）。
   约定：文案一律走 t(key)；动态文案（随手记 / 运行时长）在语言切换时由
        DYN 列表重新渲染，而不是让 data-i18n 把内容覆盖掉。
   ========================================================================= */
(function () {
  'use strict';

  /* ======================= 站点配置：改这里就够了 ======================= */
  var SITE = {
    qq: '3428936776',
    github: 'MiaM1ku',
    site: 'nyaa.fun',
    url: 'https://nyaa.fun',
    since: '2026-10-27T00:00:00+08:00',
    timeZone: 'Asia/Shanghai'
  };

  /* ============================== 小工具 ============================== */
  var root = document.documentElement;
  var mqReduce = window.matchMedia('(prefers-reduced-motion: reduce)');
  var mqFine = window.matchMedia('(hover: hover) and (pointer: fine)');

  /* 语言切换时需要重算的动态内容 */
  var DYN = [];

  function $(sel, ctx) {
    return (ctx || document).querySelector(sel);
  }

  function $$(sel, ctx) {
    return Array.prototype.slice.call((ctx || document).querySelectorAll(sel));
  }

  function store(key, value) {
    try { localStorage.setItem(key, value); } catch (e) { /* 隐私模式忽略 */ }
  }

  function read(key) {
    try { return localStorage.getItem(key); } catch (e) { return null; }
  }

  function pad2(n) {
    return n < 10 ? '0' + n : String(n);
  }

  function getLang() {
    return root.getAttribute('data-lang') === 'en' ? 'en' : 'zh';
  }

  function t(key) {
    var d = i18n[getLang()] || i18n.zh;
    if (d && d[key] != null) return d[key];
    return i18n.zh[key] != null ? i18n.zh[key] : '';
  }

  function motionOK() {
    return !mqReduce.matches;
  }

  /* ============================== 主题 ============================== */
  var themeBtn = $('#themeBtn');
  var themeColorMeta = $('meta[name="theme-color"]');
  var THEME_COLOR = { dark: '#08080c', light: '#f4f5f9' };

  function getTheme() {
    return root.getAttribute('data-theme') === 'light' ? 'light' : 'dark';
  }

  /* 把主题相关的“副作用”集中在一处：属性、meta、按钮无障碍标签 */
  function syncThemeUi() {
    var theme = getTheme();
    if (themeColorMeta) themeColorMeta.setAttribute('content', THEME_COLOR[theme]);
    if (themeBtn) {
      var label = theme === 'dark' ? t('aria_theme_to_light') : t('aria_theme_to_dark');
      themeBtn.setAttribute('aria-label', label);
      themeBtn.setAttribute('title', label);
    }
  }

  function paintTheme(theme, persist) {
    root.setAttribute('data-theme', theme);
    if (persist !== false) store('theme', theme);
    syncThemeUi();
  }

  function toggleTheme() {
    var next = getTheme() === 'light' ? 'dark' : 'light';

    /* 降级：不支持 View Transition 或用户要求减少动效时直接切换 */
    if (!themeBtn || !motionOK() || typeof document.startViewTransition !== 'function') {
      paintTheme(next);
      return;
    }

    /* 圆形扩散从按钮中心展开（原来固定从屏幕角落扩散，方向是反的） */
    var rect = themeBtn.getBoundingClientRect();
    var cx = rect.left + rect.width / 2;
    var cy = rect.top + rect.height / 2;
    var radius = Math.ceil(Math.hypot(
      Math.max(cx, window.innerWidth - cx),
      Math.max(cy, window.innerHeight - cy)
    ));

    var styleEl = document.createElement('style');
    styleEl.textContent =
      '@keyframes theme-ripple{from{clip-path:circle(0px at ' + cx + 'px ' + cy + 'px)}' +
      'to{clip-path:circle(' + radius + 'px at ' + cx + 'px ' + cy + 'px)}}' +
      '::view-transition-new(root){animation:theme-ripple .48s cubic-bezier(.4,0,.2,1)}';
    document.head.appendChild(styleEl);

    var vt = document.startViewTransition(function () { paintTheme(next); });
    var cleanup = function () {
      if (styleEl.parentNode) styleEl.parentNode.removeChild(styleEl);
    };
    if (vt && vt.finished && vt.finished.then) vt.finished.then(cleanup, cleanup);
    else setTimeout(cleanup, 800);
  }

  /* ============================== 多语言 ============================== */
  var langToggle = $('#langToggle');
  var langCurrent = $('#langCurrent');
  var langSwitching = false;

  function applyLang(next) {
    var lang = next === 'en' ? 'en' : 'zh';
    var d = i18n[lang] || i18n.zh;

    root.setAttribute('data-lang', lang);
    root.setAttribute('lang', lang === 'zh' ? 'zh-CN' : 'en');
    store('lang', lang);

    /* 静态文案 */
    $$('[data-i18n]').forEach(function (el) {
      var val = d[el.getAttribute('data-i18n')];
      if (val == null) return;
      if (val.indexOf('<') === -1) el.textContent = val;
      else el.innerHTML = val;
    });

    $$('[data-i18n-alt]').forEach(function (el) {
      var val = d[el.getAttribute('data-i18n-alt')];
      if (val != null) el.setAttribute('alt', val);
    });

    $$('[data-i18n-aria]').forEach(function (el) {
      var val = d[el.getAttribute('data-i18n-aria')];
      if (val != null) el.setAttribute('aria-label', val);
    });

    if (langCurrent) langCurrent.textContent = lang === 'zh' ? 'EN' : '中文';
    if (langToggle) langToggle.setAttribute('aria-label', d.aria_lang_switch || t('aria_lang_switch'));

    /* 动态文案（随手记正文、运行时长单位等） */
    DYN.forEach(function (fn) {
      try { fn(); } catch (e) { /* 单个模块失败不影响其它 */ }
    });

    syncThemeUi();
  }

  function switchLang() {
    if (langSwitching) return;
    langSwitching = true;

    var targets = $$('[data-i18n], [data-i18n-alt], .cave-text');
    targets.forEach(function (el) { el.classList.add('lang-fade'); });

    setTimeout(function () {
      applyLang(getLang() === 'zh' ? 'en' : 'zh');
      requestAnimationFrame(function () {
        targets.forEach(function (el) { el.classList.remove('lang-fade'); });
        langSwitching = false;
      });
    }, 240);
  }

  /* ============================== 导航 / 滚动 ============================== */
  var SECTIONS = [];

  function initNav() {
    var navbar = $('#navbar');
    var backToTop = $('#backToTop');
    var progressRing = $('#progressRingFg');
    var RING = 2 * Math.PI * 21;
    var ticking = false;

    SECTIONS = $$('.nav-link').map(function (link) {
      var id = (link.getAttribute('href') || '').replace('#', '');
      return { id: id, link: link, el: id ? document.getElementById(id) : null };
    });

    /* 滚动监听合并成一次 rAF，避免每个事件都触发布局计算 */
    function update() {
      var y = window.scrollY || window.pageYOffset || 0;

      if (navbar) navbar.classList.toggle('scrolled', y > 20);

      var probe = window.innerHeight * 0.4;
      var current = SECTIONS.length ? SECTIONS[0] : null;
      SECTIONS.forEach(function (s) {
        if (s.el && s.el.getBoundingClientRect().top - probe <= 0) current = s;
      });

      /* 滚到底时强制高亮最后一节，否则「联系」永远点不亮 */
      var max = document.documentElement.scrollHeight - window.innerHeight;
      if (max > 0 && y >= max - 2 && SECTIONS.length) current = SECTIONS[SECTIONS.length - 1];

      SECTIONS.forEach(function (s) {
        s.link.classList.toggle('active', s === current);
      });

      if (backToTop) backToTop.classList.toggle('show', y > 240);

      if (progressRing) {
        var p = max > 0 ? Math.min(1, Math.max(0, y / max)) : 0;
        progressRing.style.strokeDashoffset = String(RING * (1 - p));
      }
    }

    function onScroll() {
      if (ticking) return;
      ticking = true;
      requestAnimationFrame(function () {
        ticking = false;
        update();
      });
    }

    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('resize', onScroll, { passive: true });

    /* 锚点：走 scrollIntoView，scroll-padding-top 会避开固定导航 */
    $$('.nav-link').forEach(function (link) {
      link.addEventListener('click', function (e) {
        var href = link.getAttribute('href') || '';
        if (href.charAt(0) !== '#') return;
        var target = document.getElementById(href.slice(1));
        if (!target) return;
        e.preventDefault();
        target.scrollIntoView({ behavior: motionOK() ? 'smooth' : 'auto', block: 'start' });
      });
    });

    if (backToTop) {
      backToTop.addEventListener('click', function () {
        window.scrollTo({ top: 0, behavior: motionOK() ? 'smooth' : 'auto' });
      });
    }

    var heroContact = $('#heroContact');
    if (heroContact) {
      heroContact.addEventListener('click', function () {
        var target = document.getElementById('contact');
        if (target) target.scrollIntoView({ behavior: motionOK() ? 'smooth' : 'auto', block: 'start' });
      });
    }

    update();
    return update;
  }

  /* ============================== 自定义光标 ============================== */
  function initCursor() {
    var dot = $('#cursorDot');
    var ring = $('#cursorRing');
    if (!dot || !ring || !mqFine.matches || !motionOK()) return;

    var HOVER_SEL = 'a,button,[role="button"],[data-tilt],.interest-tag';
    var pointerX = -1;
    var pointerY = -1;
    var ringX = -1;
    var ringY = -1;
    var started = false;
    var hidden = false;

    function render() {
      dot.style.transform = 'translate3d(' + pointerX + 'px,' + pointerY + 'px,0) translate(-50%,-50%)';
      ring.style.transform = 'translate3d(' + ringX + 'px,' + ringY + 'px,0) translate(-50%,-50%)';
    }

    function loop() {
      ringX += (pointerX - ringX) * 0.16;
      ringY += (pointerY - ringY) * 0.16;
      render();
      requestAnimationFrame(loop);
    }

    function setVisible(visible) {
      if (hidden === !visible) return;
      hidden = !visible;
      dot.style.opacity = visible ? '1' : '0';
      ring.style.opacity = visible ? '1' : '0';
    }

    document.addEventListener('mousemove', function (e) {
      pointerX = e.clientX;
      pointerY = e.clientY;

      if (!started) {
        /* 首次移动才显示，避免光标固定在左上角闪一下 */
        started = true;
        ringX = pointerX;
        ringY = pointerY;
        root.classList.add('has-cursor');
        loop();
      }
      setVisible(true);
    }, { passive: true });

    document.addEventListener('mouseleave', function () { setVisible(false); });
    document.addEventListener('mouseenter', function () { setVisible(true); });

    document.addEventListener('mouseover', function (e) {
      var target = e.target;
      if (target && target.closest && target.closest(HOVER_SEL)) ring.classList.add('hover');
    });

    document.addEventListener('mouseout', function (e) {
      var target = e.target;
      if (target && target.closest && target.closest(HOVER_SEL)) ring.classList.remove('hover');
    });
  }

  /* ============================== 卡片光斑 / 倾斜 ============================== */
  function initCardEffects() {
    if (!mqFine.matches) return;

    $$('.card').forEach(function (card) {
      var tilt = card.hasAttribute('data-tilt');
      var px = 0;
      var py = 0;
      var queued = false;

      card.addEventListener('mousemove', function (e) {
        px = e.clientX;
        py = e.clientY;
        if (queued) return;
        queued = true;
        requestAnimationFrame(function () {
          queued = false;
          var rect = card.getBoundingClientRect();
          if (!rect.width || !rect.height) return;

          var x = px - rect.left;
          var y = py - rect.top;
          card.style.setProperty('--mx', x + 'px');
          card.style.setProperty('--my', y + 'px');

          if (tilt && motionOK()) {
            var rx = ((y / rect.height) - 0.5) * -5;
            var ry = ((x / rect.width) - 0.5) * 5;
            card.style.transform =
              'perspective(900px) rotateX(' + rx.toFixed(2) + 'deg) rotateY(' + ry.toFixed(2) +
              'deg) translateY(-2px)';
          }
        });
      }, { passive: true });

      if (tilt) {
        card.addEventListener('mouseleave', function () {
          card.style.transform = '';
        });
      }
    });
  }

  /* ============================== 滚动揭示 ============================== */
  function initReveal() {
    var items = $$('[data-reveal]');
    if (!items.length) return;

    if (!('IntersectionObserver' in window) || !motionOK()) {
      items.forEach(function (el) { el.classList.add('in'); });
      return;
    }

    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry, i) {
        if (!entry.isIntersecting) return;
        var el = entry.target;
        io.unobserve(el);
        setTimeout(function () { el.classList.add('in'); }, Math.min(i, 5) * 80);
      });
    }, { threshold: 0.15, rootMargin: '0px 0px -40px 0px' });

    items.forEach(function (el) { io.observe(el); });
  }

  /* ============================== 时钟 ============================== */
  function initClock() {
    var clockEl = $('#clock');
    var dateEl = $('#clockDate');
    var barEl = $('#timeProgress');
    if (!clockEl) return;

    var timer = 0;

    function tick() {
      var now = new Date();
      clockEl.textContent =
        pad2(now.getHours()) + ':' + pad2(now.getMinutes()) + ':' + pad2(now.getSeconds());

      if (dateEl) {
        dateEl.textContent = now.getFullYear() + '.' + pad2(now.getMonth() + 1) + '.' + pad2(now.getDate());
      }

      if (barEl) {
        var sec = now.getHours() * 3600 + now.getMinutes() * 60 + now.getSeconds();
        barEl.style.width = (sec / 864).toFixed(2) + '%';
      }

      /* 对齐到整秒，避免 setInterval 累积漂移 */
      timer = setTimeout(tick, 1000 - (Date.now() % 1000) + 10);
    }

    tick();
  }

  /* ============================== 运行时长 ============================== */
  function formatSince(ts) {
    var date = new Date(ts);
    try {
      var parts = new Intl.DateTimeFormat('en-CA', {
        timeZone: SITE.timeZone,
        year: 'numeric',
        month: '2-digit',
        day: '2-digit'
      }).formatToParts(date);
      var map = {};
      parts.forEach(function (p) { map[p.type] = p.value; });
      if (map.year && map.month && map.day) return map.year + '.' + map.month + '.' + map.day;
    } catch (e) { /* 老浏览器不支持 Intl 选项 */ }
    return date.getFullYear() + '.' + pad2(date.getMonth() + 1) + '.' + pad2(date.getDate());
  }

  function initUptime() {
    var daysEl = $('#uptimeDays');
    if (!daysEl) return;

    var hmEl = $('#uptimeHm');
    var sinceEl = $('#uptimeSince');
    var startTs = Date.parse(SITE.since);
    if (isNaN(startTs)) startTs = Date.now();

    function render() {
      var diff = Math.max(0, Date.now() - startTs);
      var days = Math.floor(diff / 86400000);
      var hours = Math.floor((diff % 86400000) / 3600000);
      var mins = Math.floor((diff % 3600000) / 60000);

      daysEl.textContent = String(days);
      if (hmEl) hmEl.textContent = pad2(hours) + t('uptime_hour') + ' ' + pad2(mins) + t('uptime_min');
      if (sinceEl) sinceEl.textContent = t('uptime_since') + ' ' + formatSince(startTs);
    }

    render();
    setInterval(render, 60000);
    DYN.push(render);
  }

  /* ============================== 随手记 ============================== */
  function initNote() {
    var textEl = $('#hitokoto');
    var fromEl = $('#caveFrom');
    var card = $('#caveCard');
    if (!textEl) return;

    var data = (typeof echoCaveData !== 'undefined' && Array.isArray(echoCaveData)) ? echoCaveData : [];
    if (!data.length) return;

    var index = -1;
    var current = null;
    var timer = 0;
    var token = 0;      /* 每次切换自增，过期的打字任务自动失效 */

    function textOf(item) {
      if (typeof item === 'string') return item;
      if (!item) return '';
      return (getLang() === 'en' && item.en) ? item.en : (item.content || '');
    }

    function fromOf(item) {
      return (item && item.from) ? item.from : t('cave_title');
    }

    function showFrom(item) {
      if (!fromEl) return;
      fromEl.textContent = '— ' + fromOf(item);
      fromEl.classList.add('show');
    }

    function pick() {
      if (data.length === 1) return data[0];
      var i;
      do { i = Math.floor(Math.random() * data.length); } while (i === index);
      index = i;
      return data[i];
    }

    function type(text, item) {
      var mine = ++token;
      var chars = Array.from(text);
      var i = 0;
      textEl.textContent = '';

      (function step() {
        if (mine !== token) return;          /* 已被新的切换顶掉 */
        if (i < chars.length) {
          textEl.textContent += chars[i];
          i += 1;
          timer = setTimeout(step, 42);
        } else {
          showFrom(item);
        }
      })();
    }

    /* animate=true 时先闪一下再逐字打出 */
    function next(animate) {
      token += 1;
      clearTimeout(timer);

      var item = pick();
      current = item;
      var text = textOf(item);

      if (fromEl) fromEl.classList.remove('show');
      textEl.classList.remove('flash');

      if (!animate || !motionOK()) {
        textEl.textContent = text;
        showFrom(item);
        return;
      }

      var mine = token;
      textEl.classList.add('flash');

      timer = setTimeout(function () {
        if (mine !== token) return;
        textEl.classList.remove('flash');
        type(text, item);
      }, 240);
    }

    /* 语言切换：直接换成对应语言，不重放打字动画 */
    function refresh() {
      if (!current) return;
      token += 1;
      clearTimeout(timer);
      textEl.classList.remove('flash');
      textEl.textContent = textOf(current);
      showFrom(current);
    }

    next(false);

    if (card) {
      card.addEventListener('click', function () { next(true); });
      card.addEventListener('keydown', function (e) {
        if (e.key === 'Enter' || e.key === ' ' || e.key === 'Spacebar') {
          e.preventDefault();
          next(true);
        }
      });
    }

    DYN.push(refresh);
  }

  /* ============================== Toast / 复制 ============================== */
  var toastEl = $('#toast');
  var toastTimer = 0;

  function showToast(message) {
    if (!toastEl) return;
    toastEl.textContent = message;
    toastEl.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { toastEl.classList.remove('show'); }, 2000);
  }

  function legacyCopy(text) {
    try {
      var area = document.createElement('textarea');
      area.value = text;
      area.setAttribute('readonly', '');
      area.style.position = 'fixed';
      area.style.top = '-1000px';
      area.style.opacity = '0';
      document.body.appendChild(area);
      area.select();
      area.setSelectionRange(0, area.value.length);
      var ok = document.execCommand('copy');
      document.body.removeChild(area);
      return ok;
    } catch (e) {
      return false;
    }
  }

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(text).then(
        function () { return true; },
        function () { return legacyCopy(text); }
      );
    }
    return Promise.resolve(legacyCopy(text));
  }

  function copyQQ() {
    var qq = SITE.qq;
    copyText(qq).then(function (ok) {
      showToast((ok ? t('toast_qq_copied') : t('toast_copy_failed')) + qq);
    });
  }

  function initCopy() {
    $$('[data-copy-qq]').forEach(function (el) {
      el.addEventListener('click', function (e) {
        e.preventDefault();
        copyQQ();
      });
    });
  }

  /* ============================== 头像兜底 ============================== */
  function initAvatar() {
    var img = $('#avatarImg');
    if (!img) return;

    function placeholder() {
      var initial = (SITE.github || 'M').charAt(0).toUpperCase();
      var svg =
        '<svg xmlns="http://www.w3.org/2000/svg" width="640" height="640">' +
        '<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">' +
        '<stop offset="0" stop-color="#7c6cff"/><stop offset="1" stop-color="#ff6cb8"/>' +
        '</linearGradient></defs>' +
        '<rect width="640" height="640" fill="url(#g)"/>' +
        '<text x="50%" y="50%" dy=".35em" text-anchor="middle" ' +
        'font-family="system-ui,-apple-system,sans-serif" font-size="300" font-weight="700" ' +
        'fill="#ffffff">' + initial + '</text></svg>';
      img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg);
    }

    img.addEventListener('error', function () {
      var fallback = img.getAttribute('data-fallback');
      if (fallback) {
        img.removeAttribute('data-fallback');
        img.src = fallback;
        return;
      }
      placeholder();
    });

    if (img.complete && img.naturalWidth === 0) {
      var fallback = img.getAttribute('data-fallback');
      if (fallback) img.src = fallback;
    }
  }

  /* ============================== GitHub 星数 ============================== */
  function initStars() {
    var targets = $$('.proj-star[data-repo]');
    if (!targets.length || !window.fetch) return;

    var CACHE_KEY = 'gh-stars-v2';
    var CACHE_TTL = 6 * 60 * 60 * 1000;

    /* 1000 -> 1k，1200 -> 1.2k（原来正则多转义了一层，会输出 “1.0k”） */
    function fmt(n) {
      return n >= 1000 ? (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k' : String(n);
    }

    function apply(map) {
      targets.forEach(function (el) {
        var value = map[(el.getAttribute('data-repo') || '').toLowerCase()];
        if (typeof value === 'number') el.textContent = '★ ' + fmt(value);
      });
    }

    var cached = null;
    try { cached = JSON.parse(read(CACHE_KEY) || 'null'); } catch (e) { cached = null; }
    if (cached && cached.t && cached.m && Date.now() - cached.t < CACHE_TTL) {
      apply(cached.m);
      return;
    }

    if (navigator.onLine === false) return;

    var owners = [];
    targets.forEach(function (el) {
      var owner = (el.getAttribute('data-repo') || '').split('/')[0];
      if (owner && owners.indexOf(owner) === -1) owners.push(owner);
    });

    Promise.all(owners.map(function (owner) {
      return fetch('https://api.github.com/users/' + encodeURIComponent(owner) + '/repos?per_page=100', {
        headers: { Accept: 'application/vnd.github+json' }
      }).then(function (res) {
        return res.ok ? res.json() : [];
      }).catch(function () {
        return [];
      });
    })).then(function (lists) {
      var map = {};
      lists.forEach(function (list) {
        if (!Array.isArray(list)) return;
        list.forEach(function (repo) {
          if (!repo || !repo.name) return;
          var owner = (repo.owner && repo.owner.login ? repo.owner.login : '').toLowerCase();
          map[owner + '/' + String(repo.name).toLowerCase()] = repo.stargazers_count;
        });
      });

      /* 取不到就保留 HTML 里的静态值，不把卡片清空 */
      if (!Object.keys(map).length) return;
      apply(map);
      try { store(CACHE_KEY, JSON.stringify({ t: Date.now(), m: map })); } catch (e) { /* 超配额忽略 */ }
    });
  }

  /* ============================== 彩蛋 ============================== */
  function launchConfetti(count, cx, cy, circle) {
    if (!motionOK()) return;

    var colors = ['#ff6b6b', '#feca57', '#48dbfb', '#a55eea', '#ff6cb8', '#4ade80', '#7c6cff'];
    var originX = cx || window.innerWidth / 2;
    var originY = cy || window.innerHeight / 2;

    for (var i = 0; i < count; i += 1) {
      var piece = document.createElement('div');
      var size = 6 + Math.random() * 6;
      var color = colors[Math.floor(Math.random() * colors.length)];

      piece.style.cssText =
        'position:fixed;z-index:99997;pointer-events:none;' +
        'width:' + size + 'px;height:' + size + 'px;background:' + color + ';' +
        'border-radius:' + (Math.random() > 0.5 ? '50%' : '2px') + ';' +
        'left:' + originX + 'px;top:' + originY + 'px;';
      document.body.appendChild(piece);

      (function (el) {
        var angle = circle ? (i / count) * Math.PI * 2 : Math.random() * Math.PI * 2;
        var speed = circle ? 200 + Math.random() * 200 : 100 + Math.random() * 300;
        var vx = Math.cos(angle) * speed;
        var vy = Math.sin(angle) * speed - (circle ? 0 : 150);
        var rot = (Math.random() - 0.5) * 720;
        var duration = 1200 + Math.random() * 800;
        var start = performance.now();

        function fall(now) {
          var p = (now - start) / duration;
          if (p >= 1) {
            el.remove();
            return;
          }
          el.style.transform =
            'translate(' + (vx * p) + 'px,' + (vy * p + 600 * p * p) + 'px) rotate(' + (rot * p) + 'deg)';
          el.style.opacity = String(1 - p);
          requestAnimationFrame(fall);
        }
        requestAnimationFrame(fall);
      })(piece);
    }
  }

  function initEggs() {
    /* Konami Code */
    var KONAMI = ['ArrowUp', 'ArrowUp', 'ArrowDown', 'ArrowDown',
      'ArrowLeft', 'ArrowRight', 'ArrowLeft', 'ArrowRight', 'b', 'a'];
    var konamiIndex = 0;

    document.addEventListener('keydown', function (e) {
      var key = e.key.length === 1 ? e.key.toLowerCase() : e.key;
      if (key === KONAMI[konamiIndex]) {
        konamiIndex += 1;
        if (konamiIndex === KONAMI.length) {
          konamiIndex = 0;
          triggerKonami();
        }
      } else {
        konamiIndex = (key === KONAMI[0]) ? 1 : 0;
      }
    });

    function triggerKonami() {
      showToast(t('toast_konami'));
      if (!motionOK()) return;

      var flash = document.createElement('div');
      flash.style.cssText =
        'position:fixed;inset:0;z-index:99998;pointer-events:none;opacity:0;' +
        'transition:opacity .2s;mix-blend-mode:screen;' +
        'background:linear-gradient(45deg,#ff6b6b,#feca57,#48dbfb,#a55eea,#ff6cb8);';
      document.body.appendChild(flash);
      requestAnimationFrame(function () { flash.style.opacity = '0.4'; });
      setTimeout(function () { flash.style.opacity = '0'; }, 600);
      setTimeout(function () { flash.remove(); }, 900);

      launchConfetti(80);
      document.body.style.transition = 'transform 1.2s cubic-bezier(.2,.8,.2,1)';
      document.body.style.transform = 'rotate(360deg)';
      setTimeout(function () {
        document.body.style.transform = '';
        setTimeout(function () { document.body.style.transition = ''; }, 1300);
      }, 1200);
    }

    /* 点 Logo 撒花 */
    var logo = $('#navLogo');
    if (logo) {
      logo.style.cursor = 'pointer';
      logo.addEventListener('click', function () {
        var rect = logo.getBoundingClientRect();
        launchConfetti(40, rect.left + rect.width / 2, rect.top + rect.height / 2);
      });
    }

    /* 连点头像 5 次 */
    var avatar = $('#avatarImg');
    if (!avatar) return;

    var clicks = 0;
    var resetTimer = 0;

    avatar.addEventListener('click', function () {
      clicks += 1;
      clearTimeout(resetTimer);
      resetTimer = setTimeout(function () { clicks = 0; }, 1500);
      if (clicks >= 5) {
        clicks = 0;
        triggerAvatarEgg();
      }
    });

    function triggerAvatarEgg() {
      showToast(t('toast_avatar_egg'));
      if (!motionOK()) return;

      var ring = $('.avatar-ring');
      if (ring) {
        ring.style.transition = 'transform 1.5s cubic-bezier(.2,.8,.2,1)';
        ring.style.transform = 'rotate(720deg) scale(1.4)';
        setTimeout(function () {
          ring.style.transform = '';
          setTimeout(function () { ring.style.transition = ''; }, 1600);
        }, 1500);
      }

      var rect = avatar.getBoundingClientRect();
      launchConfetti(60, rect.left + rect.width / 2, rect.top + rect.height / 2, true);

      var inner = $('.avatar-inner');
      if (inner) {
        var hue = 0;
        var flash = setInterval(function () {
          inner.style.boxShadow = '0 0 24px 4px hsl(' + hue + ',90%,60%)';
          hue = (hue + 40) % 360;
        }, 80);
        setTimeout(function () {
          clearInterval(flash);
          inner.style.boxShadow = '';
        }, 2000);
      }
    }
  }

  /* ============================== 启动 ============================== */
  function boot() {
    if (themeBtn) themeBtn.addEventListener('click', toggleTheme);
    if (langToggle) langToggle.addEventListener('click', switchLang);

    initCursor();
    initCardEffects();
    initNav();
    initReveal();
    initCopy();
    initAvatar();
    initNote();
    initClock();
    initUptime();
    initEggs();
    initStars();

    /* 首屏统一落地一次文案（也会触发 DYN 里的动态内容） */
    applyLang(getLang());

    var yearEl = $('#footerYear');
    if (yearEl) yearEl.textContent = String(new Date().getFullYear());

    /* 刷新时回到顶部；带 hash 的深链保持原行为 */
    window.addEventListener('load', function () {
      if (!location.hash) window.scrollTo(0, 0);
    });

    if (window.console && console.info) {
      console.info(
        '%c MiaM1ku · nyaa.fun ',
        'background:#7c6cff;color:#fff;padding:3px 10px;border-radius:4px;font-weight:600'
      );
      console.info(
        [
          '',
          '  ╔══════════════════════════════════╗',
          '  ║  MiaM1ku · 学生 / 独立开发者      ║',
          '  ║  Python · Go · TS · Bun · Android ║',
          '  ╚══════════════════════════════════╝',
          '  nyaa.fun · github.com/MiaM1ku',
          '  你找到了这里 ✦ 试试：↑↑↓↓←→←→BA',
          ''
        ].join('\n')
      );
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
