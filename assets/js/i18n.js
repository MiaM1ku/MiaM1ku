/* =========================================================================
   文案表：只放文本，逻辑一律在 script.js
   -------------------------------------------------------------------------
   约定
   - 中英两份必须保持相同的 key 集合（script.js 里有缺 key 的兜底）
   - 值里允许出现 <br/> 之类的简单行内标签，纯文本会用 textContent 写入
   - aria_* 是给无文字按钮用的无障碍标签
   ========================================================================= */
const i18n = {
  zh: {
    /* 品牌 / 导航 */
    brand: "Nya 的学习笔记",
    nav_home: "首页",
    nav_about: "关于",
    nav_stack: "技术栈",
    nav_projects: "项目",
    nav_contact: "联系",
    footer_brand: "Nya 的学习笔记",

    /* Hero */
    hero_name: "MiaM1ku",
    avatar_alt: "MiaM1ku",
    hero_tag: "✦ 学生 · 独立开发者",
    hero_hello: "你好，我是",
    hero_desc: "来自中国的学生，喜欢全栈开发与运维，也长期折腾 AI 和自动化。<br/>正在学习 Python、Golang、TypeScript (Bun) 与 Android。",
    hero_btn_contact: "联系我",
    avatar_loc: "📍 中国 · UTC+8",

    /* 统计卡 */
    time_label: "本地时间",
    uptime_label: "本站已运行",
    github_hint: "50+ 公开仓库 ↗",
    qq_hint: "点击复制 📋",
    interests_label: "兴趣领域",
    tag_coding: "💻 编程",
    tag_tech: "🛰️ 科技前沿",
    tag_ai: "🧠 人工智能",
    tag_python: "🐍 Python",
    tag_go: "🐹 Golang",
    tag_ts: "🟦 TypeScript",
    tag_bun: "🍞 Bun",
    tag_android: "🤖 Android",
    tag_opensource: "🌱 开源",
    tag_tinker: "🛠️ 折腾",
    tag_automation: "⚙️ 自动化",
    tag_network: "🌐 网络",

    /* 运行时长 */
    uptime_day: "天",
    uptime_hour: "时",
    uptime_min: "分",
    uptime_since: "since",

    /* 社交 */
    social_label: "社交",
    social_site: "本站",
    social_repos: "仓库",

    /* 随手记 */
    cave_title: "随手记",
    cave_hint: "点击换一句",
    cave_from: "— 随手记",

    /* 关于 */
    about_title: "关于我",
    about_h1: "编程",
    about_p1: "喜欢把想法写成能跑起来的东西：脚本、小工具、服务端、界面，能亲手做出来就很开心。",
    about_h2: "科技前沿",
    about_p2: "对新技术保持雷达常开，看到有意思的东西就想拆开看看它是怎么跑起来的。",
    about_h3: "人工智能",
    about_p3: "关注大模型与 Agent 工程，更在意把 AI 接到真实工具和工作流里，让它真的干活。",
    about_h4: "持续学习",
    about_p4: "Python、Golang、TypeScript (Bun)、Android 都在路上——边学边写，边写边拆。",

    /* 技术栈 */
    stack_title: "技术栈 · 正在学习",
    stack_py_sub: "脚本 · 工具 · AI",
    stack_py_desc: "最顺手的一门。写脚本、做小工具、跑数据处理和 AI 实验，几乎什么都先用它试一遍。",
    stack_go_sub: "服务 · 并发 · 命令行",
    stack_go_desc: "在学习中。喜欢它部署简单、并发省心，写网络服务和命令行工具的顺手程度越来越高。",
    stack_ts_sub: "BUN · 前端 · 工具链",
    stack_ts_desc: "在学习中，用 Bun 跑脚本和服务很舒服。前端页面、插件、小工具链都靠它撑着。",
    stack_android_sub: "KOTLIN · 系统 · 模块",
    stack_android_desc: "刚上路。从 Kotlin 基础到系统能力和模块化折腾，想弄明白一个 App 到底是怎么跑在手机上的。",
    level_fluent: "顺手",
    level_advancing: "进阶中",
    level_beginner: "入门中",

    /* 项目 */
    proj_title: "项目",
    proj_d1: "全球网测 / 泰尔测速的 Linux 客户端，把测速这件事搬到命令行里。",
    proj_d2: "支付聚合面板，多通道统一接入，把收款这件事做成能自己掌控的样子。",
    proj_d3: "LSPosed 模块：把小米手环 / 手表接入 ColorOS 设备中心与 OPPO 健康。",
    proj_d4: "给 AI 一把 SSH 钥匙：Termius Cloud 的 stdio MCP 服务器。",
    proj_d5: "Koishi 的哔哩哔哩私信适配器，让机器人和 B 站私信互通。",
    proj_d6: "给 QPlayer 写的一串音源插件：酷狗、酷我、咪咕、汽水、Apple Music。",
    proj_d7: "Rust 写的 LLM API 网关，和 Ikaleio 一起折腾的项目。",
    proj_more: "更多仓库",
    proj_more_tag: "ALL",
    proj_more_desc: "GitHub 上还有一堆在折腾的小东西，慢慢翻。",

    /* 联系 */
    contact_title: "联系我",
    contact_headline: "聊两句？",
    contact_sub: "技术、AI、开源，或者只是路过想打个招呼——都欢迎。",

    /* 友情链接（页面最下面） */
    friends_title: "友情链接",
    friend_ikaleio: "一起做过 monoize",
    friend_guimc: "朋友的小站",
    friend_weqiu: "朋友的小站",

    /* Toast */
    toast_qq_copied: "QQ号已复制：",
    toast_copy_failed: "复制失败，请手动复制：",
    toast_konami: "🎮 Konami! 彩蛋已激活 ✦",
    toast_avatar_egg: "✦ 哇！你发现了头像彩蛋！",

    /* 无障碍标签 */
    aria_lang_switch: "切换语言",
    aria_theme_to_light: "切换到浅色主题",
    aria_theme_to_dark: "切换到深色主题",
    aria_back_top: "回到顶部",
    aria_copy_qq: "复制 QQ 号",
    aria_note_next: "换一句随手记",
  },

  en: {
    brand: "Nya's Study Notes",
    nav_home: "Home",
    nav_about: "About",
    nav_stack: "Stack",
    nav_projects: "Projects",
    nav_contact: "Contact",
    footer_brand: "Nya's Study Notes",

    hero_name: "MiaM1ku",
    avatar_alt: "MiaM1ku",
    hero_tag: "✦ STUDENT · INDIE DEV",
    hero_hello: "Hi, I'm",
    hero_desc: "A student in China, into full-stack development and ops — and always tinkering with AI and automation.<br/>Currently learning Python, Golang, TypeScript (Bun) and Android.",
    hero_btn_contact: "Contact Me",
    avatar_loc: "📍 China · UTC+8",

    time_label: "Local Time",
    uptime_label: "Site Uptime",
    github_hint: "50+ public repos ↗",
    qq_hint: "Click to copy 📋",
    interests_label: "Interests",
    tag_coding: "💻 Coding",
    tag_tech: "🛰️ Frontier Tech",
    tag_ai: "🧠 AI",
    tag_python: "🐍 Python",
    tag_go: "🐹 Golang",
    tag_ts: "🟦 TypeScript",
    tag_bun: "🍞 Bun",
    tag_android: "🤖 Android",
    tag_opensource: "🌱 Open Source",
    tag_tinker: "🛠️ Tinkering",
    tag_automation: "⚙️ Automation",
    tag_network: "🌐 Network",

    uptime_day: "d",
    uptime_hour: "h",
    uptime_min: "m",
    uptime_since: "since",

    social_label: "Social",
    social_site: "This Site",
    social_repos: "Repos",

    cave_title: "Notes",
    cave_hint: "Click for another",
    cave_from: "— Notes",

    about_title: "About Me",
    about_h1: "Coding",
    about_p1: "I like turning ideas into things that run: scripts, small tools, services, interfaces. Building it by hand is the fun part.",
    about_h2: "Frontier Tech",
    about_p2: "Always watching new tech with the radar on. When something looks interesting, I want to take it apart and see how it runs.",
    about_h3: "Artificial Intelligence",
    about_p3: "Following LLMs and agent engineering, and caring more about wiring AI into real tools and workflows so it actually gets work done.",
    about_h4: "Always Learning",
    about_p4: "Python, Golang, TypeScript (Bun), Android — all on the road. Learn while writing, write while taking things apart.",

    stack_title: "Stack · Now Learning",
    stack_py_sub: "SCRIPT · TOOLING · AI",
    stack_py_desc: "The one I reach for first. Scripts, small tools, data crunching and AI experiments all start here.",
    stack_go_sub: "SERVICE · CONCURRENCY · CLI",
    stack_go_desc: "Still learning it. Simple to deploy, painless concurrency — writing services and CLI tools gets smoother every week.",
    stack_ts_sub: "BUN · WEB · TOOLCHAIN",
    stack_ts_desc: "Learning it, and Bun makes running scripts and services a pleasure. Pages, plugins and small toolchains live here.",
    stack_android_sub: "KOTLIN · SYSTEM · MODULE",
    stack_android_desc: "Just getting started. From Kotlin basics to system capabilities and modules — I want to know how an app really runs on a phone.",
    level_fluent: "Comfortable",
    level_advancing: "Advancing",
    level_beginner: "Beginner",

    proj_title: "Projects",
    proj_d1: "A Linux client for the global network speed test — moving speed testing into the terminal.",
    proj_d2: "A payment aggregation panel with unified multi-channel access.",
    proj_d3: "An LSPosed module bridging Xiaomi bands and watches into ColorOS Device Center and OPPO Health.",
    proj_d4: "An SSH key for AI: a stdio MCP server for Termius Cloud.",
    proj_d5: "A Bilibili private-message adapter for Koishi — let your bot talk through B站 DMs.",
    proj_d6: "A set of music-source plugins for QPlayer: Kugou, Kuwo, Migu, Soda and Apple Music.",
    proj_d7: "An LLM API gateway in Rust, built together with Ikaleio.",
    proj_more: "More repos",
    proj_more_tag: "ALL",
    proj_more_desc: "Plenty more half-finished experiments on GitHub — take a look.",

    contact_title: "Contact",
    contact_headline: "Let's talk",
    contact_sub: "Tech, AI, open source — or just dropping by to say hi. All welcome.",

    friends_title: "Friends",
    friend_ikaleio: "Built monoize together",
    friend_guimc: "A friend's site",
    friend_weqiu: "A friend's site",

    toast_qq_copied: "QQ copied: ",
    toast_copy_failed: "Copy failed, please copy manually: ",
    toast_konami: "🎮 Konami! Easter egg activated ✦",
    toast_avatar_egg: "✦ Wow! You found the avatar easter egg!",

    aria_lang_switch: "Switch language",
    aria_theme_to_light: "Switch to light theme",
    aria_theme_to_dark: "Switch to dark theme",
    aria_back_top: "Back to top",
    aria_copy_qq: "Copy QQ number",
    aria_note_next: "Show another note",
  },
};
