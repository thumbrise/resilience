import {defineConfig} from 'vitepress'

export default defineConfig({
  title: 'resilience',
  description: 'Composable resilience for Go function calls. Zero-dependency core with retry, backoff, and OpenTelemetry plugins.',
  base: '/resilience/',
  sitemap: {
    hostname: 'https://thumbrise.github.io/resilience',
  },
  head: [
    ['meta', {property: 'og:type', content: 'website'}],
    ['meta', {property: 'og:title', content: 'resilience — composable resilience for Go'}],
    ['meta', {property: 'og:description', content: 'Zero-dependency core. Retry, backoff, plugins. One primitive: func(ctx, call) error.'}],
    ['meta', {property: 'og:url', content: 'https://thumbrise.github.io/resilience/'}],
    ['meta', {name: 'twitter:card', content: 'summary'}],
    ['meta', {name: 'twitter:title', content: 'resilience — composable resilience for Go'}],
    ['meta', {name: 'twitter:description', content: 'Zero-dependency core. Retry, backoff, plugins. One primitive: func(ctx, call) error.'}],
  ],

  themeConfig: {
    nav: [
      {text: 'Guide', link: '/guide/getting-started'},
      {text: 'Internals', link: '/internals/multimod/'},
      {text: 'Devlog', link: '/devlog/'},
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Guide',
          items: [
            {text: 'Getting Started', link: '/guide/getting-started'},
          ],
        },
      ],
      '/internals/multimod/': [
        {
          text: 'multimod',
          items: [
            {text: 'Overview', link: '/internals/multimod/'},
            {text: 'Specification', link: '/internals/multimod/spec'},
            {text: 'Research', link: '/internals/multimod/research'},
            {text: 'Vision', link: '/internals/multimod/vision'},
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
          ],
        },
      ],
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
