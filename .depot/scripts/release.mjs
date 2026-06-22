#!/usr/bin/env zx
import { $, chalk, echo } from 'zx'
import semver from 'semver' // @^7.6.3

$.verbose = false

const args = parseArgs(scriptArgs())
const dryRun = args.flags.has('dry-run')
const yes = args.flags.has('yes')
const fetchRefs = args.flags.has('fetch') || (!dryRun && !args.flags.has('no-fetch'))
const remote = args.values.remote || 'origin'

if (args.flags.has('help')) {
  printHelp()
  process.exit(0)
}

const branchName = args.values.branch || process.env.GITHUB_REF_NAME || await currentBranch()

if (!branchName || branchName === 'HEAD') {
  fail('Could not determine the current branch. Pass --branch <name>.')
}

if (fetchRefs) {
  await fetchReleaseRefs(remote)
}

const plan = await createPlan(branchName, remote)
printPlan(plan, dryRun)

if (dryRun) {
  await confirmPlan(yes)
}

await applyPlan(plan, { dryRun, remote })

if (dryRun) {
  echo(chalk.green('Applied locally. No refs were pushed because --dry-run is set.'))
} else {
  echo(chalk.green('Release ref pushed.'))
}

async function createPlan(branch, remoteName) {
  if (branch === 'main') {
    const releaseBranch = await nextReleaseBranch()

    return {
      action: 'branch',
      source: branch,
      target: releaseBranch,
      commands: [
        `git branch ${releaseBranch} HEAD`,
        `git push ${remoteName} refs/heads/${releaseBranch}:refs/heads/${releaseBranch}`,
      ],
    }
  }

  const releaseMatch = branch.match(/^release\/(\d+)-(\d+)$/)
  if (releaseMatch) {
    const major = Number(releaseMatch[1])
    const minor = Number(releaseMatch[2])
    const tag = await nextReleaseTag(major, minor)

    return {
      action: 'tag',
      source: branch,
      target: tag,
      commands: [
        `git tag ${tag} HEAD`,
        `git push ${remoteName} refs/tags/${tag}`,
      ],
    }
  }

  fail(`Release automation only runs on main or release/<major>-<minor> branches. Current branch: ${branch}`)
}

async function nextReleaseBranch() {
  const releaseLines = await releaseLinesFromBranches()
  const tagLines = await releaseLinesFromTags()
  const latest = maxLine([...releaseLines, ...tagLines])
  const next = latest ? { major: latest.major, minor: latest.minor + 1 } : { major: 0, minor: 1 }
  return `release/${next.major}-${next.minor}`
}

async function nextReleaseTag(major, minor) {
  const releaseTags = await getTags()
  const matching = releaseTags
    .map((tag) => semver.parse(normalizeTag(tag)))
    .filter((version) => version && version.major === major && version.minor === minor)

  if (matching.length === 0) {
    return `${major}.${minor}.0`
  }

  const latest = matching.sort(semver.rcompare)[0]
  return semver.inc(latest, 'patch')
}

async function releaseLinesFromBranches() {
  const refs = await $`git for-each-ref "--format=%(refname:short)" refs/heads refs/remotes`.quiet()
  const lines = refs.stdout
    .split('\n')
    .map((ref) => ref.trim().replace(/^origin\//, ''))
    .filter(Boolean)

  return unique(lines)
    .map((ref) => ref.match(/^release\/(\d+)-(\d+)$/))
    .filter(Boolean)
    .map((match) => ({ major: Number(match[1]), minor: Number(match[2]) }))
}

async function releaseLinesFromTags() {
  return (await getTags())
    .map((tag) => semver.parse(normalizeTag(tag)))
    .filter(Boolean)
    .map((version) => ({ major: version.major, minor: version.minor }))
}

async function getTags() {
  const result = await $`git tag --list`.quiet()
  return result.stdout.split('\n').map((tag) => tag.trim()).filter(Boolean)
}

function maxLine(lines) {
  return lines.sort((a, b) => b.major - a.major || b.minor - a.minor)[0]
}

function normalizeTag(tag) {
  return tag.startsWith('v') ? tag.slice(1) : tag
}

async function applyPlan(plan, options) {
  if (plan.action === 'branch') {
    await ensureRefMissing(`refs/heads/${plan.target}`, `branch ${plan.target}`)
    await $`git branch ${plan.target} HEAD`

    if (!options.dryRun) {
      await $`git push ${options.remote} refs/heads/${plan.target}:refs/heads/${plan.target}`
    }
    return
  }

  if (plan.action === 'tag') {
    await ensureRefMissing(`refs/tags/${plan.target}`, `tag ${plan.target}`)
    await $`git tag ${plan.target} HEAD`

    if (!options.dryRun) {
      await $`git push ${options.remote} refs/tags/${plan.target}`
    }
    return
  }

  fail(`Unknown release plan action: ${plan.action}`)
}

async function ensureRefMissing(ref, label) {
  const result = await $`git show-ref --verify --quiet ${ref}`.nothrow().quiet()
  if (result.exitCode === 0) {
    fail(`${label} already exists locally.`)
  }
}

async function fetchReleaseRefs(remoteName) {
  await $`git fetch --prune --tags ${remoteName} +refs/heads/*:refs/remotes/${remoteName}/*`
}

async function currentBranch() {
  const result = await $`git branch --show-current`.quiet()
  return result.stdout.trim()
}

async function confirmPlan(skipPrompt) {
  if (skipPrompt) {
    echo(chalk.yellow('Confirmation skipped because --yes was provided.'))
    return
  }

  if (!process.stdin.isTTY) {
    fail('--dry-run requires an interactive terminal. Pass --yes to apply the local dry run non-interactively.')
  }

  const { default: readline } = await import('node:readline/promises')
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout })
  const answer = await rl.question('Apply this plan locally without pushing? [y/N] ')
  rl.close()

  if (!['y', 'yes'].includes(answer.trim().toLowerCase())) {
    fail('Aborted.')
  }
}

function printPlan(plan, isDryRun) {
  echo(chalk.bold('Release plan'))
  echo(`Current branch: ${plan.source}`)
  echo(`Action: ${plan.action === 'branch' ? 'create release branch' : 'create release tag'}`)
  echo(`Target: ${plan.target}`)
  echo(`Mode: ${isDryRun ? 'dry run, apply locally, do not push' : 'CI, apply and push'}`)
  echo('')
  echo(chalk.bold('Commands'))
  for (const command of plan.commands) {
    if (isDryRun && command.startsWith('git push ')) {
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

  for (let index = 0; index < rawArgs.length; index += 1) {
    const arg = rawArgs[index]
    if (!arg.startsWith('--')) {
      fail(`Unexpected argument: ${arg}`)
    }

    const [rawName, inlineValue] = arg.slice(2).split('=', 2)
    if (['branch', 'remote'].includes(rawName)) {
      const value = inlineValue ?? rawArgs[index + 1]
      if (!value || value.startsWith('--')) {
        fail(`Missing value for --${rawName}`)
      }
      values[rawName] = value
      if (inlineValue === undefined) {
        index += 1
      }
    } else {
      flags.add(rawName)
    }
  }

  return { flags, values }
}

function scriptArgs() {
  const rawArgs = process.argv.slice(2)
  const scriptIndex = rawArgs.findIndex((arg) => !arg.startsWith('--') && arg.endsWith('release.mjs'))
  if (scriptIndex >= 0) {
    return rawArgs.slice(scriptIndex + 1)
  }
  return rawArgs
}

function unique(values) {
  return [...new Set(values)]
}

function printHelp() {
  echo(`Usage: zx --install scripts/release.mjs [options]

Options:
  --branch <name>  Branch to release from. Defaults to GITHUB_REF_NAME or current branch.
  --remote <name>  Git remote to push to. Defaults to origin.
  --dry-run        Print the plan, confirm, apply locally, and skip pushes.
  --yes            Skip confirmation for --dry-run.
  --fetch          Fetch release refs even in --dry-run mode.
  --no-fetch       Skip fetching refs in CI mode.
  --help           Show this help.
`)
}

function fail(message) {
  echo(chalk.red(message))
  process.exit(1)
}
