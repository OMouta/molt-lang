---
layout: home

hero:
  name: "@molt"
  tagline: Build, transform, and execute code as first-class runtime values.
  image:
    src: /logo.png
    alt: molt logo
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: Language Guide
      link: /guide/language-guide
    - theme: alt
      text: View on GitHub
      link: https://github.com/OMouta/molt-lang

features:
  - icon: 📦
    title: Code as Values
    details: Capture expressions with @{ ... }, transform them with mutation literals ~{ ... }, and execute with eval().
  - icon: ⚡
    title: First-class Errors
    details: Build structured errors with error(), raise them with throw(), and recover cleanly with try ... catch.
  - icon: 🔍
    title: Pattern Matching
    details: Match against literals, identifier bindings, and wildcards with a clean match expression.
  - icon: 🔗
    title: Modules
    details: Import and export bindings across .molt files. Modules run in isolated scopes with explicit exports only.
---
