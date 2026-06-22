#!/usr/bin/env zx
/* global $, chalk, echo */
import semver from 'semver' // @^7.6.3
import readline from 'readline/promises'

$.verbose = false

const args = parseArgs(scriptArgs(process.argv))

if (args.flags.has('help')) {
  printHelp()
  process.exit(0)
}

const dryRun = args.flags.has('dry-run')
const yes = args.flags.has('yes')
const remote = args.values.remote || 'origin'
const prNumber = args.values['pr-number'] || process.env.PR_NUMBER
const repo = args.values.repo || process.env.GITHUB_REPOSITORY || await inferRepository(remote)
const token = args.values['github-token'] || process.env.GITHUB_TOKEN || process.env.GH_TOKEN || ''
const slackWebhook = args.values['slack-webhook-url'] || process.env.SLACK_WEBHOOK_RELEASE_NEXT || ''
const fetchRefs = args.flags.has('fetch') || !args.flags.has('no-fetch')

if (!prNumber) {
  fail('Missing pull request number. Pass --pr-number <number> or set PR_NUMBER.')
}

if (fetchRefs) {
  await fetchReleaseRefs(remote)
}

class BackportConflictError extends Error {
  constructor(plan, cause) {
    const detail = cause instanceof Error ? cause.message : String(cause)
    super(`Cherry-pick failed for ${plan.mergeCommit} onto ${plan.releaseBranch}: ${detail}`)
    this.name = 'BackportConflictError'
    this.cause = cause
    this.plan = plan
  }
}

let activePlan

try {
  const pullRequest = await getPullRequest({ repo, prNumber, token })
  validatePullRequest(pullRequest)

  const plan = await createPlan({ pullRequest, remote, repo })
  activePlan = plan
  printPlan(plan, dryRun)

  if (dryRun) {
    await confirmPlan(yes)
  }

  await applyPlan(plan, { dryRun, remote, repo, token })

  if (dryRun) {
    echo(chalk.green('Applied locally. No refs were pushed and no pull request was created because --dry-run is set.'))
  } else {
    await notifySlack(slackWebhook, successMessage(plan))
    echo(chalk.green(`Backport pull request created: ${plan.backportPullRequest.html_url}`))
  }
} catch (error) {
  const message = error instanceof Error ? error.message : String(error)
  if (!dryRun) {
    await notifySlack(slackWebhook, failureMessage({ repo, prNumber, message, plan: activePlan, error }))
  }
  echo(chalk.red(message))
  process.exit(1)
}

async function createPlan({ pullRequest, remote: remoteName, repo: repository }) {
  const releaseBranch = await latestReleaseBranch(remoteName)
  const mergeCommit = pullRequest.merge_commit_sha
  const shortSha = mergeCommit.slice(0, 12)
  const parentCount = await commitParentCount(mergeCommit)
  const cherryPickCommand = parentCount > 1
    ? `git cherry-pick -x -m 1 ${mergeCommit}`
    : `git cherry-pick -x ${mergeCommit}`
  const backportBranch = `backport-pr-${pullRequest.number}-${shortSha}-to-${releaseBranch.replaceAll('/', '-')}`
  const title = `[${releaseBranch}] ${pullRequest.title}`
  const body = [
    `Backports #${pullRequest.number} to \`${releaseBranch}\`.`,
    '',
    `Source PR: ${pullRequest.html_url}`,
    `Cherry-pick commit: \`${mergeCommit}\``,
    '',
    'This pull request was created by `.depot/workflows/backport.yaml`.',
  ].join('\n')

  return {
    sourcePullRequest: pullRequest,
    releaseBranch,
    mergeCommit,
    backportBranch,
    title,
    body,
    repo: repository,
    commands: [
      `git checkout -B ${backportBranch} ${remoteName}/${releaseBranch}`,
      cherryPickCommand,
      `git push ${remoteName} refs/heads/${backportBranch}:refs/heads/${backportBranch}`,
      `POST /repos/${repository}/pulls`,
    ],
  }
}

async function applyPlan(plan, { dryRun: isDryRun, remote: remoteName, repo: repository, token: githubToken }) {
  await ensureBranchMissing(plan.backportBranch)
  await $`git checkout -B ${plan.backportBranch} ${`${remoteName}/${plan.releaseBranch}`}`
  try {
    await cherryPick(plan.mergeCommit)
  } catch (error) {
    throw new BackportConflictError(plan, error)
  }

  if (isDryRun) {
    return
  }

  await $`git push ${remoteName} ${`refs/heads/${plan.backportBranch}:refs/heads/${plan.backportBranch}`}`
  plan.backportPullRequest = await createPullRequest({
    repo: repository,
    token: githubToken,
    title: plan.title,
    head: plan.backportBranch,
    base: plan.releaseBranch,
    body: plan.body,
  })
}

async function cherryPick(commit) {
  const parentCount = await commitParentCount(commit)
  const command = parentCount > 1
    ? $`git cherry-pick -x -m 1 ${commit}`
    : $`git cherry-pick -x ${commit}`

  try {
    await command
  } catch (error) {
    if (await resolveCiWorkflowConflict()) {
      return
    }

    throw error
  }
}

async function resolveCiWorkflowConflict() {
  const conflicts = await conflictedFiles()
  if (conflicts.length !== 1 || conflicts[0] !== '.depot/workflows/ci.yml') {
    return false
  }

  echo(chalk.yellow('Resolving .depot/workflows/ci.yml conflict by taking the source PR version.'))
  await $`git checkout --theirs .depot/workflows/ci.yml`
  await $`git add .depot/workflows/ci.yml`
  await $`git cherry-pick --continue`
  return true
}

async function conflictedFiles() {
  const result = await $`git diff --name-only --diff-filter=U`.quiet()
  return result.stdout
    .split('\n')
    .map((file) => file.trim())
    .filter(Boolean)
}

async function commitParentCount(commit) {
  const result = await $`git rev-list --parents -n 1 ${commit}`.quiet()
  return Math.max(0, result.stdout.trim().split(/\s+/).length - 1)
}

async function latestReleaseBranch(remoteName) {
  const refs = await releaseBranches(remoteName)
  if (refs.length === 0) {
    fail('No release branches found. Expected branches named release/<major>-<minor>.')
  }

  return refs.sort((a, b) => semver.rcompare(a.version, b.version))[0].name
}

async function releaseBranches(remoteName) {
  const result = await $`git for-each-ref ${'--format=%(refname:short)'} refs/heads refs/remotes`.quiet()
  const branches = result.stdout
    .split('\n')
    .map((ref) => ref.trim())
    .filter(Boolean)
    .map((ref) => ref.replace(new RegExp(`^${escapeRegExp(remoteName)}/`), ''))

  return unique(branches)
    .map((branch) => {
      const match = branch.match(/^release\/(\d+)-(\d+)$/)
      if (!match) {
        return null
      }

      return {
        name: branch,
        version: semver.parse(`${match[1]}.${match[2]}.0`),
      }
    })
    .filter(Boolean)
}

async function fetchReleaseRefs(remoteName) {
  await $`git fetch --prune ${remoteName} +refs/heads/release/*:refs/remotes/${remoteName}/release/*`
  await $`git fetch ${remoteName} +refs/heads/main:refs/remotes/${remoteName}/main`.nothrow().quiet()
}

async function ensureBranchMissing(branch) {
  const local = await $`git rev-parse --verify --quiet ${`refs/heads/${branch}`}`.nothrow().quiet()
  if (local.exitCode === 0) {
    fail(`Branch ${branch} already exists locally.`)
  }
}

async function getPullRequest({ repo: repository, prNumber: number, token: githubToken }) {
  if (!githubToken) {
    fail('Missing GitHub token. Set GITHUB_TOKEN, GH_TOKEN, or pass --github-token.')
  }

  return githubRequest({
    repo: repository,
    token: githubToken,
    path: `/pulls/${number}`,
  })
}

function validatePullRequest(pullRequest) {
  if (!pullRequest.merged) {
    fail(`Pull request #${pullRequest.number} has not been merged.`)
  }
  if (!pullRequest.merge_commit_sha) {
    fail(`Pull request #${pullRequest.number} does not have a merge commit SHA.`)
  }
  if (!pullRequest.labels?.some((label) => label.name === 'backport')) {
    fail(`Pull request #${pullRequest.number} does not have the backport label.`)
  }
}

async function createPullRequest({ repo: repository, token: githubToken, title, head, base, body }) {
  if (!githubToken) {
    fail('Missing GitHub token. Set GITHUB_TOKEN, GH_TOKEN, or pass --github-token.')
  }

  return githubRequest({
    repo: repository,
    token: githubToken,
    path: '/pulls',
    method: 'POST',
    body: {
      title,
      head,
      base,
      body,
      maintainer_can_modify: true,
    },
  })
}

async function githubRequest({ repo: repository, token: githubToken, path, method = 'GET', body }) {
  const response = await fetch(`https://api.github.com/repos/${repository}${path}`, {
    method,
    headers: {
      accept: 'application/vnd.github+json',
      authorization: `Bearer ${githubToken}`,
      'content-type': 'application/json',
      'x-github-api-version': '2022-11-28',
    },
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!response.ok) {
    const text = await response.text()
    fail(`GitHub API ${method} ${path} failed with ${response.status}: ${text}`)
  }

  return response.json()
}

async function notifySlack(webhookUrl, message) {
  if (!webhookUrl) {
    echo(chalk.yellow('Slack notification skipped because SLACK_WEBHOOK_RELEASE_NEXT is not set.'))
    return
  }

  const response = await fetch(webhookUrl, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ text: message }),
  })

  if (!response.ok) {
    echo(chalk.yellow(`Slack notification failed with ${response.status}: ${await response.text()}`))
  }
}

function successMessage(plan) {
  return [
    `Backport PR created for #${plan.sourcePullRequest.number}: ${plan.backportPullRequest.html_url}`,
    `Target release branch: ${plan.releaseBranch}`,
  ].join('\n')
}

function failureMessage({ repo: repository, prNumber: number, message, plan, error }) {
  if (error instanceof BackportConflictError && plan) {
    return [
      `Automatic backport for ${repository}#${number} hit a cherry-pick conflict.`,
      `Original PR: ${plan.sourcePullRequest.html_url}`,
      `Target release branch: ${plan.releaseBranch}`,
      `Backport branch: ${plan.backportBranch}`,
      `Merge commit: ${plan.mergeCommit}`,
      '',
      'The release team needs to resolve the backport manually.',
      `Error: ${message}`,
    ].join('\n')
  }

  return [
    `Backport automation failed for ${repository}#${number}.`,
    `Release master should coordinate with the developer to resolve it.`,
    `Error: ${message}`,
  ].join('\n')
}

async function inferRepository(remoteName) {
  const result = await $`git remote get-url ${remoteName}`.quiet()
  const url = result.stdout.trim()
  const match = url.match(/github\.com[:/](?<owner>[^/]+)\/(?<repo>[^/.]+)(?:\.git)?$/)
  if (!match?.groups) {
    fail(`Could not infer GitHub repository from ${remoteName} remote URL. Pass --repo owner/name.`)
  }

  return `${match.groups.owner}/${match.groups.repo}`
}

async function confirmPlan(skipPrompt) {
  if (skipPrompt) {
    echo(chalk.yellow('Confirmation skipped because --yes was provided.'))
    return
  }

  if (!process.stdin.isTTY) {
    fail('--dry-run requires an interactive terminal. Pass --yes to apply the local dry run non-interactively.')
  }

  const rl = readline.createInterface({ input: process.stdin, output: process.stdout })
  const answer = await rl.question('Apply this backport locally without pushing or opening a PR? [y/N] ')
  rl.close()

  if (!['y', 'yes'].includes(answer.trim().toLowerCase())) {
    fail('Aborted.')
  }
}

function printPlan(plan, isDryRun) {
  echo(chalk.bold('Backport plan'))
  echo(`Source PR: #${plan.sourcePullRequest.number} ${plan.sourcePullRequest.title}`)
  echo(`Merge commit: ${plan.mergeCommit}`)
  echo(`Target release branch: ${plan.releaseBranch}`)
  echo(`Backport branch: ${plan.backportBranch}`)
  echo(`Mode: ${isDryRun ? 'dry run, apply locally, do not push or create PR' : 'CI, push branch and create PR'}`)
  echo('')
  echo(chalk.bold('Commands'))
  for (const command of plan.commands) {
    if (isDryRun && (command.startsWith('git push ') || command.startsWith('POST '))) {
      echo(chalk.dim(`# skipped by --dry-run: ${command}`))
    } else {
      echo(command)
    }
  }
  echo('')
}

function parseArgs(rawArgs) {
  const flags = new Set()
  const values = {}
  const valueFlags = new Set(['github-token', 'pr-number', 'remote', 'repo', 'slack-webhook-url'])

  for (let index = 0; index < rawArgs.length; index += 1) {
    const arg = rawArgs[index]
    if (!arg.startsWith('--')) {
      fail(`Unexpected argument: ${arg}`)
    }

    const [name, inlineValue] = arg.slice(2).split('=', 2)
    if (valueFlags.has(name)) {
      const value = inlineValue ?? rawArgs[index + 1]
      if (!value || value.startsWith('--')) {
        fail(`Missing value for --${name}`)
      }
      values[name] = value
      if (inlineValue === undefined) {
        index += 1
      }
    } else {
      flags.add(name)
    }
  }

  return { flags, values }
}

function scriptArgs(argv) {
  return argv
    .slice(2)
    .filter((arg) => arg !== '--install')
    .filter((arg) => !arg.endsWith('backport.mjs'))
}

function printHelp() {
  echo(`Usage:
  zx --install scripts/backport.mjs --pr-number <number> [options]

Options:
  --dry-run                  Show the plan, confirm, create a local branch, cherry-pick, and skip remote effects.
  --yes                      Skip interactive confirmation for --dry-run.
  --pr-number <number>       Merged pull request to backport. Defaults to PR_NUMBER.
  --repo <owner/name>        GitHub repository. Defaults to GITHUB_REPOSITORY or origin URL.
  --remote <name>            Git remote. Defaults to origin.
  --github-token <token>     GitHub token. Defaults to GITHUB_TOKEN or GH_TOKEN.
  --slack-webhook-url <url>  Slack incoming webhook for #release-next. Defaults to SLACK_WEBHOOK_RELEASE_NEXT.
  --fetch                    Fetch release refs before planning. This is the default.
  --no-fetch                 Skip fetching release refs.
`)
}

function unique(values) {
  return [...new Set(values)]
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function fail(message) {
  throw new Error(message)
}
