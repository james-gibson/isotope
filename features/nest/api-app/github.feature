# The github service (api/src/github/): GithubController + the traced service
# that clones a private repository over SSH and reads a file from HEAD.
# Mounted on the main app (3001). Env var NAMES only: GITHUB_SSH_KEY_PATH,
# GITHUB_REPO_SSH_URL.
# Sources: api/src/github/github.controller.ts, api/src/github/github-traced.service.ts.
# Captured 2026-11-03 from the prototype.

@api @github
Feature: The github service serves the installer from a private repository over SSH
  As a machine being onboarded
  I want the installer script fetchable from the api
  So that setup needs nothing but curl

  Scenario: the installer script is served from the private repository
    Given the main app is running on port 3001
    And the github SSH key and repository URL are configured
    When a client requests GET /installer.sh
    Then the install.sh file at the repository's HEAD is returned

  Scenario: a repository without the file answers 404
    Given the github service is configured
    And the repository's HEAD holds no install.sh
    When a client requests GET /installer.sh
    Then the response status is 404
