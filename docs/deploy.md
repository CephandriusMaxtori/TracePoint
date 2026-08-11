---
layout: default
title: Deploying the docs
---

# Publishing these docs to GitHub Pages

These docs live in the `docs/` folder and are built with Jekyll using the
built-in `jekyll-theme-slate` theme. A GitHub Actions workflow (`.github/workflows/pages.yml`)
builds and deploys them automatically on every push that touches `docs/`.

## One-time setup

1. Push the repository to GitHub.
2. Open **Settings → Pages** for the repository.
3. Under **Build and deployment → Source**, choose **GitHub Actions**.
4. Push any change to the `docs/` folder (or run the
   **Deploy docs to GitHub Pages** workflow manually via the **Actions** tab).
5. The workflow builds the Jekyll site and deploys it.

The site is published at:

```
https://<user>.github.io/TracePoint/
```

Replace `<user>` with your GitHub username. If you get a 404 after enabling
Pages, wait for the workflow to finish, then check the **Actions → Deploy docs
to GitHub Pages → deploy** job logs — the `page_url` there is the canonical
address.

## Editing locally

To preview with the same theme locally:

```sh
gem install jekyll bundler
cd docs
bundle init
echo 'gem "jekyll-theme-slate"' >> Gemfile
bundle exec jekyll serve
```
