import { defineConfig } from 'vitepress'

export default defineConfig({
  title: '@molt',
  description: 'A language where code is data.',
  head: [['link', { rel: 'icon', href: '/logo.png' }]],
  cleanUrls: true,

  themeConfig: {
    logo: '/logo.png',
    siteTitle: '@molt',

    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Standard Library', link: '/reference/standard-library' },
      { text: 'Examples', link: '/reference/examples' },
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'Getting Started', link: '/guide/getting-started' },
          { text: 'Language Guide', link: '/guide/language-guide' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'Standard Library', link: '/reference/standard-library' },
          { text: 'Examples', link: '/reference/examples' },
        ],
      },
      {
        text: 'Internals',
        items: [
          { text: 'Architecture', link: '/internals/architecture' },
          { text: 'Editor Support', link: '/internals/editor-support' },
        ],
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/OMouta/molt-lang' },
    ],

    search: { provider: 'local' },

    editLink: {
      pattern: 'https://github.com/OMouta/molt-lang/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },

    footer: {
      message: 'Released under the MIT License.',
    },
  },

  markdown: {
    theme: { light: 'github-light', dark: 'github-dark' },
  },
})
