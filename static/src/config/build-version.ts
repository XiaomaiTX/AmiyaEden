const SEMVER_PATTERN =
  /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/

/** Returns the package version that is embedded in the frontend build. */
export function resolveBuildVersion(version: unknown): string {
  if (typeof version !== 'string' || !SEMVER_PATTERN.test(version)) {
    throw new Error('static/package.json must define a valid semantic version')
  }

  return version
}
