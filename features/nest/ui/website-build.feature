# Capture of the website build pipeline (ui/).
# Sources: ui/package.json, ui/.lemonseed/index.mjs, ui/src/vite.config.ts,
# ui/postbuild.sh, ui/src/personality/*.md, README.md (repo root, rsync deploy),
# www/setup.md (web host provisioning runbook).
# Captured 2026-11-03 from the prototype.

@ui @build
Feature: The website build compiles personalities, pages and service workers into a deployable tree
  As the site's publisher
  I want a deterministic build that emits the app, its agent personalities and its service workers
  So that one artifact tree can be rsynced to the web host

  Scenario: the build generates the agent personality manifest
    Given personality markdown files exist under the ui source
    When I run "npm run build" in the ui directory
    Then every personality file is parsed and validated
    And a per-agent JSON file and a manifest with sha256 hashes are emitted

  Scenario: the vite build emits the multi-page app with three service workers
    When I run "npm run build" in the ui directory
    Then the page entries are emitted under the base path /_._/
    And three service workers are emitted with the scopes /, /_._ and /nsole

  Scenario: postbuild injects the CSP nonce into the built index
    Given the vite build has emitted its index page
    When I run "bash ./postbuild.sh"
    Then a nonce attribute is injected into the script and stylesheet tags

  Scenario: deploy stages the build tree for rsync
    Given the build and postbuild steps have run
    When I run "npm run deploy"
    Then the built tree is copied to build/
    And the lemonseed manifest is staged under .well-known/appspecific/

  Scenario: the staged tree is published by rsync to the web host
    Given a web host prepared per www/setup.md
    When the README's rsync of ui/build/ to the host's web root runs
    Then the website is served from the host's web root
