import { describe, expect, it } from 'vitest'
import { clinicDate, validCalendarDate } from '../../src/lib/clinic-date'
describe('Clinic date boundaries', () => {
  it('groups an early-morning India booking on its clinic day', () => {
    expect(clinicDate('2026-09-18T20:00:00Z', 'Asia/Kolkata')).toBe('2026-09-19')
  })
  it('groups a late-night American booking on the previous UTC day', () => {
    expect(clinicDate('2026-09-19T02:00:00Z', 'America/New_York')).toBe('2026-09-18')
  })
  it('handles the daylight-saving transition', () => {
    expect(clinicDate('2026-11-01T05:30:00Z', 'America/New_York')).toBe('2026-11-01')
    expect(clinicDate('2026-11-01T06:30:00Z', 'America/New_York')).toBe('2026-11-01')
  })
  it.each(['', '2026-02-30', 'not-a-date', '2026-9-1'])('rejects an invalid calendar value: %s', value => {
    expect(validCalendarDate(value)).toBe(false)
  })
  it('accepts leap days only in leap years', () => {
    expect(validCalendarDate('2028-02-29')).toBe(true)
    expect(validCalendarDate('2026-02-29')).toBe(false)
  })
})
