/* =========================================================================
   WebGL2 背景：点阵噪波
   -------------------------------------------------------------------------
   - 30fps 上限 + 标签页隐藏时暂停，避免后台空转烧电
   - 系统开启「减少动态效果」时只渲染一帧静态画面
   - 处理 webglcontextlost / restored，显卡休眠恢复后能自己重建资源
   ========================================================================= */
(function () {
  'use strict';

  var canvas = document.getElementById('shaderCanvas');
  if (!canvas) return;

  var gl = null;
  try {
    gl = canvas.getContext('webgl2', {
      antialias: false,
      alpha: false,
      depth: false,
      stencil: false,
      premultipliedAlpha: false,
      powerPreference: 'low-power'
    });
  } catch (err) {
    gl = null;
  }
  /* 没有 WebGL2 就交给 .aurora 的 CSS 渐变兜底 */
  if (!gl) return;

  var mqReduce = window.matchMedia('(prefers-reduced-motion: reduce)');
  var FRAME_MS = 1000 / 30;

  var program = null;
  var buffer = null;
  var uniforms = {};
  var rafId = 0;
  var running = false;
  var contextLost = false;
  var lastDraw = 0;
  var startedAt = performance.now();

  var mx = 0.5, my = 0.5;      // 平滑后的鼠标
  var tmx = 0.5, tmy = 0.5;    // 目标鼠标

  var VERT_SRC = '#version 300 es\n' +
    'in vec2 a_position;\n' +
    'void main() {\n' +
    '  gl_Position = vec4(a_position, 0.0, 1.0);\n' +
    '}\n';

  var FRAG_SRC = [
    '#version 300 es',
    'precision highp float;',
    'uniform vec2 u_resolution;',
    'uniform vec2 u_mouse;',
    'uniform float u_time;',
    'uniform float u_darkMode;',
    'uniform float u_scroll;',
    'out vec4 outputColor;',
    '',
    'vec2 mod289(vec2 value) { return value - floor(value * (1.0 / 289.0)) * 289.0; }',
    'vec3 mod289(vec3 value) { return value - floor(value * (1.0 / 289.0)) * 289.0; }',
    'vec3 permute(vec3 value) { return mod289(((value * 34.0) + 10.0) * value); }',
    '',
    'float simplexNoise(vec2 point) {',
    '  const vec4 c = vec4(0.211324865405187, 0.366025403784439, -0.577350269189626, 0.024390243902439);',
    '  vec2 lattice = floor(point + dot(point, c.yy));',
    '  vec2 origin = point - lattice + dot(lattice, c.xx);',
    '  vec2 corner = origin.x > origin.y ? vec2(1.0, 0.0) : vec2(0.0, 1.0);',
    '  vec4 offsets = origin.xyxy + c.xxzz;',
    '  offsets.xy -= corner;',
    '  lattice = mod289(lattice);',
    '  vec3 perm = permute(permute(lattice.y + vec3(0.0, corner.y, 1.0)) + lattice.x + vec3(0.0, corner.x, 1.0));',
    '  vec3 weight = max(0.5 - vec3(',
    '    dot(origin, origin),',
    '    dot(offsets.xy, offsets.xy),',
    '    dot(offsets.zw, offsets.zw)',
    '  ), 0.0);',
    '  weight *= weight;',
    '  weight *= weight;',
    '  vec3 gx = 2.0 * fract(perm * c.www) - 1.0;',
    '  vec3 gy = abs(gx) - 0.5;',
    '  vec3 gz = floor(gx + 0.5);',
    '  vec3 g = gx - gz;',
    '  weight *= 1.79284291400159 - 0.85373472095314 * (g * g + gy * gy);',
    '  vec3 contrib;',
    '  contrib.x = g.x * origin.x + gy.x * origin.y;',
    '  contrib.yz = g.yz * offsets.xz + gy.yz * offsets.yw;',
    '  return 130.0 * dot(weight, contrib);',
    '}',
    '',
    'float ditherFbm(vec2 p) {',
    '  float value = 0.0;',
    '  float amplitude = 0.52;',
    '  mat2 rotation = mat2(0.80, -0.60, 0.60, 0.80);',
    '  for (int i = 0; i < 5; i++) {',
    '    value += amplitude * clamp(simplexNoise(p) * 0.5 + 0.5, 0.0, 1.0);',
    '    p = rotation * p * 2.03 + 11.7;',
    '    amplitude *= 0.48;',
    '  }',
    '  return value;',
    '}',
    '',
    'float bayer4(vec2 p) {',
    '  int x = int(mod(p.x, 4.0));',
    '  int y = int(mod(p.y, 4.0));',
    '  int i = y * 4 + x;',
    '  float v = 0.0;',
    '  if (i == 0) v = 0.0; else if (i == 1) v = 8.0;',
    '  else if (i == 2) v = 2.0; else if (i == 3) v = 10.0;',
    '  else if (i == 4) v = 12.0; else if (i == 5) v = 4.0;',
    '  else if (i == 6) v = 14.0; else if (i == 7) v = 6.0;',
    '  else if (i == 8) v = 3.0; else if (i == 9) v = 11.0;',
    '  else if (i == 10) v = 1.0; else if (i == 11) v = 9.0;',
    '  else if (i == 12) v = 15.0; else if (i == 13) v = 7.0;',
    '  else if (i == 14) v = 13.0; else if (i == 15) v = 5.0;',
    '  return v / 16.0;',
    '}',
    '',
    'void main() {',
    '  const float pixelSize = 2.0;',
    '  vec2 ditherCoord = floor(gl_FragCoord.xy / pixelSize);',
    '  vec2 pixelCoord = (ditherCoord + 0.5) * pixelSize;',
    '  vec2 uv = pixelCoord / u_resolution;',
    '  vec2 p = uv - 0.5;',
    '  p.x *= u_resolution.x / max(u_resolution.y, 1.0);',
    '',
    '  vec3 dark = vec3(0.0235, 0.0235, 0.0275);',
    '  vec3 light = vec3(0.9686, 0.9686, 0.9725);',
    '  vec3 background = mix(light, dark, u_darkMode);',
    '  vec3 foreground = mix(dark, light, u_darkMode);',
    '',
    '  vec2 cursor = u_mouse - 0.5;',
    '  cursor.x *= u_resolution.x / max(u_resolution.y, 1.0);',
    '  float cursorField = exp(-4.2 * length(p - cursor));',
    '',
    '  vec2 q = p * 1.22;',
    '  float scrollOff = u_scroll * 0.005;',
    '  float field = ditherFbm(q * 2.1 + vec2(u_time * 0.045 + scrollOff, -u_time * 0.025));',
    '  field += 0.24 * sin((q.x - q.y) * 3.6 - u_time * 0.16);',
    '  field += cursorField * 0.18;',
    '',
    '  float threshold = bayer4(ditherCoord);',
    '  float dots = step(threshold, clamp(field * 0.9 - 0.12, 0.0, 1.0));',
    '',
    '  float dotStrength = mix(0.22, 0.34, u_darkMode);',
    '  vec3 color = mix(background, foreground, dots * dotStrength);',
    '',
    '  vec3 accent = vec3(0.388, 0.400, 0.945);',
    '  color = mix(color, accent, cursorField * dots * 0.25);',
    '',
    '  float vignette = smoothstep(1.12, 0.30, length(p * vec2(0.70, 1.0)));',
    '  color = mix(background, color, 0.74 + vignette * 0.26);',
    '',
    '  outputColor = vec4(color, 1.0);',
    '}'
  ].join('\n');

  function resize() {
    var w = Math.max(1, window.innerWidth);
    var h = Math.max(1, window.innerHeight);
    if (canvas.width !== w || canvas.height !== h) {
      canvas.width = w;
      canvas.height = h;
      canvas.style.width = w + 'px';
      canvas.style.height = h + 'px';
    }
    gl.viewport(0, 0, canvas.width, canvas.height);
  }

  function compileShader(type, source) {
    var shader = gl.createShader(type);
    gl.shaderSource(shader, source);
    gl.compileShader(shader);
    if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
      if (window.console) console.error('[shader] compile failed:', gl.getShaderInfoLog(shader));
      gl.deleteShader(shader);
      return null;
    }
    return shader;
  }

  function build() {
    var vs = compileShader(gl.VERTEX_SHADER, VERT_SRC);
    var fs = compileShader(gl.FRAGMENT_SHADER, FRAG_SRC);
    if (!vs || !fs) {
      if (vs) gl.deleteShader(vs);
      if (fs) gl.deleteShader(fs);
      return false;
    }

    program = gl.createProgram();
    gl.attachShader(program, vs);
    gl.attachShader(program, fs);
    gl.linkProgram(program);
    gl.deleteShader(vs);
    gl.deleteShader(fs);

    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
      if (window.console) console.error('[shader] link failed:', gl.getProgramInfoLog(program));
      return false;
    }
    gl.useProgram(program);

    buffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]), gl.STATIC_DRAW);

    var aPos = gl.getAttribLocation(program, 'a_position');
    gl.enableVertexAttribArray(aPos);
    gl.vertexAttribPointer(aPos, 2, gl.FLOAT, false, 0, 0);

    uniforms = {
      resolution: gl.getUniformLocation(program, 'u_resolution'),
      mouse: gl.getUniformLocation(program, 'u_mouse'),
      time: gl.getUniformLocation(program, 'u_time'),
      darkMode: gl.getUniformLocation(program, 'u_darkMode'),
      scroll: gl.getUniformLocation(program, 'u_scroll')
    };

    gl.disable(gl.BLEND);
    gl.disable(gl.DEPTH_TEST);
    return true;
  }

  function draw(now) {
    if (!program || contextLost) return;

    resize();

    /* 时间基于 startedAt，切换页面回来不会突然跳变 */
    var seconds = mqReduce.matches ? 8 : (now - startedAt) / 1000;
    var isLight = document.documentElement.getAttribute('data-theme') === 'light';

    mx += (tmx - mx) * 0.12;
    my += (tmy - my) * 0.12;

    gl.uniform2f(uniforms.resolution, canvas.width, canvas.height);
    gl.uniform2f(uniforms.mouse, mx, my);
    gl.uniform1f(uniforms.time, seconds);
    gl.uniform1f(uniforms.darkMode, isLight ? 0 : 1);
    gl.uniform1f(uniforms.scroll, window.scrollY || 0);
    gl.clearColor(0, 0, 0, 1);
    gl.clear(gl.COLOR_BUFFER_BIT);
    gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
  }

  function frame(now) {
    rafId = requestAnimationFrame(frame);
    if (now - lastDraw < FRAME_MS) return;
    lastDraw = now;
    draw(now);
  }

  function start() {
    if (running || contextLost || !program) return;
    running = true;
    lastDraw = 0;

    /* 减少动态效果：只画一帧，等 resize / 主题变化时再补一帧 */
    if (mqReduce.matches) {
      draw(performance.now());
      running = false;
      return;
    }
    rafId = requestAnimationFrame(frame);
  }

  function stop() {
    running = false;
    if (rafId) cancelAnimationFrame(rafId);
    rafId = 0;
  }

  function redrawOnce() {
    if (!running) {
      requestAnimationFrame(function (now) {
        draw(now);
      });
    }
  }

  /* --- 初始化 --- */
  if (!build()) return;
  resize();
  start();

  window.addEventListener('resize', function () {
    resize();
    redrawOnce();
  }, { passive: true });

  /* 主题切换后：运行中每帧都会重读属性，静态模式补一帧 */
  new MutationObserver(redrawOnce).observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme']
  });

  mqReduce.addEventListener('change', function () {
    stop();
    start();
  });

  document.addEventListener('visibilitychange', function () {
    if (document.hidden) stop();
    else start();
  });

  document.addEventListener('mousemove', function (e) {
    tmx = e.clientX / Math.max(1, window.innerWidth);
    tmy = 1 - e.clientY / Math.max(1, window.innerHeight);
  }, { passive: true });

  canvas.addEventListener('webglcontextlost', function (e) {
    e.preventDefault();
    contextLost = true;
    stop();
  }, false);

  canvas.addEventListener('webglcontextrestored', function () {
    contextLost = false;
    if (build()) start();
  }, false);
})();
