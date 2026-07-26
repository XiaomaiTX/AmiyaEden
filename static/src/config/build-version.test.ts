import assert from 'node:assert/strict'
import test from 'node:test'
import { resolveBuildVersion } from './build-version'

test('resolveBuildVersion returns a valid package version unchanged', () => {
  assert.equal(resolveBuildVersion('1.16.0'), '1.16.0')
  assert.equal(resolveBuildVersion('1.16.0-rc.1+build.42'), '1.16.0-rc.1+build.42')
})

test('resolveBuildVersion rejects missing and invalid package versions', () => {
  assert.throws(() => resolveBuildVersion(undefined), /valid semantic version/)
  assert.throws(() => resolveBuildVersion('1.16'), /valid semantic version/)
  assert.throws(() => resolveBuildVersion('01.16.0'), /valid semantic version/)
})
