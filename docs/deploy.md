---
layout: default
title: Deploying the docs
---

# Publishing these docs to GitHub Pages

These docs live in the `docs/` folder and use a Jekyll remote theme, so GitHub Pages can build them with zero configuration.

## One-time setup

1. Push the repository to GitHub (create a repo named e.g. `TracePoint`).
2. Open **Settings → Pages** for the repository.
3. Under **Build and deployment → Source**, choose **Deploy from a branch**.
4. Set the branch to `main` and the folder to `/docs`, then save.

GitHub will build the site with Jekyll and publish it at:

```
https://<user>.github.io/TracePoint/
```

## Editing locally

The theme is pulled in via `jekyll-remote-theme`. To preview locally:

```sh
gem install jekyll bundler jekyll-remote-theme
cd docs
bundle init
echo 'gem "jekyll-remote-theme"' >> Gemfile
bundle exec jekyll serve
```
