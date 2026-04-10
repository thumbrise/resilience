import {defineConfig} from 'vitepress'  
  
export default defineConfig({  
  title: 'resilience',  
  description: 'Open source fault tolerance library for Go. Composable retry, backoff, circuit breaker, rate limiter middleware. Zero dependencies. One primitive: func(ctx, call) error.',  
  base: '/resilience/',  
  sitemap: {  
    hostname: 'https://thumbrise.github.io/resilience/',  
  },  
  transformHead({pageData, siteData}) {  
    const base = 'https://thumbrise.github.io/resilience'  
    const relativePath = pageData.relativePath.replace(/\.md$/, '.html').replace(/index\.html$/, '')  
    const url = relativePath ? `${base}/${relativePath}` : `${base}/`  
    const title = pageData.title || siteData.title  
    const description = pageData.description || siteData.description  
  
    return [  
      ['meta', {property: 'og:url', content: url}],  
      ['meta', {property: 'og:title', content: title}],  
      ['meta', {property: 'og:description', content: description}],  
      ['meta', {name: 'twitter:title', content: title}],  
      ['meta', {name: 'twitter:description', content: description}],  
    ]  
  },  
  head: [  
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/resilience/favicon.svg' }],  
    ['link', { rel: 'icon', type: 'image/png', sizes: '96x96', href: '/resilience/favicon-96x96.png' }],  
    ['link', { rel: 'apple-touch-icon', sizes: '180x180', href: '/resilience/apple-touch-icon.png' }],  
    ['meta', { property: 'og:image', content: 'https://thumbrise.github.io/resilience/og-image.png' }],  
    ['meta', {property: 'og:type', content: 'website'}],  
    ['meta', {name: 'twitter:card', content: 'summary'}],  
    ['meta', {name: 'keywords', content: 'go resilience library, golang retry, fault tolerance go, go circuit breaker, go backoff, go error handling, resilience middleware, rate limiter go, open source go library, zero dependency'}],  
  ],  
  
  themeConfig: {  
    nav: [  
      {text: 'Guide', link: '/guide/getting-started'},  
      {text: 'multimod ↗', link: 'https://thumbrise.github.io/multimod/'},  
      {text: 'Devlog', link: '/devlog/'},  
      {text: 'References', link: '/references/'},  
    ],  
  
    sidebar: {  
      '/guide/': [  
        {  
          text: 'Guide',  
          items: [  
            {text: 'Getting Started', link: '/guide/getting-started'},  
            {text: 'Retry', link: '/guide/retry'},  
            {text: 'Backoff', link: '/guide/backoff'},  
            {text: 'Observability (OTEL)', link: '/guide/otel'},  
            {text: 'Options & Plugins', link: '/guide/options-plugins'},  
            {text: 'Roadmap', link: '/guide/roadmap'},  
          ],  
        },  
      ],  
      '/internals/multimod/': [  
        {  
          text: 'multimod (moved)',  
          items: [  
            {text: 'Overview → multimod repo', link: '/internals/multimod/'},  
          ],  
        },  
      ],  
      '/devlog/': [  
        {  
          text: 'Devlog',  
          items: [  
            {text: 'About This Devlog', link: '/devlog/'},  
            {text: '#1 — Package Extracting', link: '/devlog/001-package-extracting'},  
            {text: '#2 — The Task Runner Lifecycle Gap', link: '/devlog/002-taskrunner-lifecycle-gap'},  
            {text: '#3 — The Multi-Module Gap', link: '/devlog/003-multimod-gap'},  
            {text: '#4 — Building multimod', link: '/devlog/004-multimod-implementation'},  
            {text: '#5 — Release design', link: '/devlog/005-multimod-release-design'},  
            {text: '#6 — autosolve migration', link: '/devlog/006-autosolve-migration'},  
            {text: '#7 — The architecture fight', link: '/devlog/007-adversarial-architecture-review'},  
            {text: '#8 — multimod extracted', link: '/devlog/008-multimod-extracted'},  
            {text: '#9 — The stress debate', link: '/devlog/009-stress-debate'},  
          ],  
        },  
      ],  
      '/references/': [  
        {  
          text: 'References',  
          items: [  
            {text: 'References', link: '/references/'},  
            {text: 'RFC-002 — Resilience Stress Debate', link: '/references/rfc-002-resilience-stress-debate'}  
          ]  
        }  
      ]  
    },  
  
    socialLinks: [  
      {icon: 'github', link: 'https://github.com/thumbrise/resilience'},  
    ],  
  
    editLink: {  
      pattern: 'https://github.com/thumbrise/resilience/edit/main/docs/:path',  
    },  
  
    footer: {  
      message: 'Apache 2.0 · Built in public · Contributions welcome',  
    },  
  },  
})
