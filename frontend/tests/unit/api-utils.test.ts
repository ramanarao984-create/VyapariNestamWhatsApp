import { describe, expect, it } from 'vitest'
import { getErrorMessage } from '../../src/lib/api-utils'
describe('Actionable form errors', () => {
  it('preserves server validation messages', () => {
    expect(getErrorMessage({ isAxiosError: true, response: { data: { message: 'Phone number is invalid' }, status: 400 } })).toBe('Phone number is invalid')
  })
  it('never renders an object from a malformed error array', () => {
    expect(getErrorMessage({ isAxiosError: true, response: { data: { errors: [{ field: 'name' }] }, status: 400 } }, 'Could not save')).toBe('Could not save')
  })
  it('explains timeouts and connection failures', () => {
    expect(getErrorMessage({ isAxiosError: true, code: 'ECONNABORTED' })).toContain('timed out')
    expect(getErrorMessage({ isAxiosError: true })).toContain('reach the server')
  })
  it('gives useful permission and server error feedback', () => {
    expect(getErrorMessage({ isAxiosError: true, response: { status: 403 } })).toContain('permission')
    expect(getErrorMessage({ isAxiosError: true, response: { status: 500 } }, 'Could not save clinic setup')).toBe('Could not save clinic setup')
  })
})
