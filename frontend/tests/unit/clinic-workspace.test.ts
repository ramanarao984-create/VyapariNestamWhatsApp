import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { shallowMount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
const mocks = vi.hoisted(() => ({
  profile: vi.fn(), appointments: vi.fn(), waitlist: vi.fn(), practitioners: vi.fn(), services: vi.fn(),
  accounts: vi.fn(), contacts: vi.fn(), slots: vi.fn(), save: vi.fn(), create: vi.fn(),
  allowed: new Set(['clinic', 'contacts', 'accounts']),
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ hasPermission: (resource: string) => mocks.allowed.has(resource) }) }))
vi.mock('@/services/api', () => ({
  clinicService: { getProfile: mocks.profile, listAppointments: mocks.appointments, listWaitlist: mocks.waitlist,
    listPractitioners: mocks.practitioners, listServices: mocks.services, listSlots: mocks.slots,
    updateProfile: mocks.save, createAppointment: mocks.create },
  accountsService: { list: mocks.accounts }, contactsService: { list: mocks.contacts },
}))
vi.mock('@/components/shared', () => ({ CreateContactDialog: { template: '<div />' }, PageHeader: { template: '<header />' }, ErrorState: { props: ['title', 'description'], template: '<div role="alert">{{title}} {{description}}</div>' } }))
vi.mock('vue-sonner', () => ({ toast: { error: vi.fn(), success: vi.fn() } }))
import ClinicWorkspace from '../../src/views/clinic/ClinicWorkspaceView.vue'
const response = (data: unknown) => ({ data: { data } })
const profile = { id: 'clinic', display_name: 'Test clinic', timezone: 'Asia/Kolkata', default_slot_minutes: 20 }
let wrapper: VueWrapper<any>
const mount = () => wrapper = shallowMount(ClinicWorkspace, { global: { renderStubDefaultSlot: true, stubs: { RouterLink: { template: '<a><slot /></a>' } } } })
beforeEach(() => {
  vi.clearAllMocks(); mocks.allowed = new Set(['clinic', 'contacts', 'accounts'])
  mocks.profile.mockResolvedValue(response({ profile, configured: true }))
  mocks.appointments.mockResolvedValue(response({ appointments: [] }))
  mocks.waitlist.mockResolvedValue(response({ entries: [] }))
  mocks.practitioners.mockResolvedValue(response({ practitioners: [{ id: 'p1', display_name: 'Doctor', is_active: true }] }))
  mocks.services.mockResolvedValue(response({ services: [] }))
  mocks.accounts.mockResolvedValue(response({ accounts: [{ id: 'a', name: 'Clinic WhatsApp' }] }))
  mocks.contacts.mockResolvedValue(response({ contacts: [{ id: 'c1', whatsapp_account: 'Clinic WhatsApp' }] }))
  mocks.slots.mockResolvedValue(response({ slots: [] }))
  mocks.save.mockResolvedValue(response({ profile }))
})
afterEach(() => wrapper?.unmount())
describe('Clinic reception workflow', () => {
  it('shows a loading state without flashing first-time setup', async () => {
    let resolve!: (value: unknown) => void
    mocks.profile.mockReturnValue(new Promise(r => { resolve = r }))
    mount()
    expect(wrapper.text()).toContain('Loading your clinic workspace')
    expect(wrapper.text()).not.toContain('Set up your clinic first')
    resolve(response({ profile, configured: true })); await flushPromises()
    expect(wrapper.text()).toContain('Test clinic')
  })
  it('keeps the clinic usable when optional account information fails', async () => {
    mocks.accounts.mockRejectedValue(new Error('offline'))
    mount(); await flushPromises()
    expect(wrapper.text()).toContain('Test clinic')
    expect(wrapper.text()).toContain('WhatsApp accounts could not load')
    expect(wrapper.vm.failed).toBe(false)
  })
  it('does not request accounts or contacts a role cannot read', async () => {
    mocks.allowed = new Set(['clinic'])
    mount(); await flushPromises()
    expect(mocks.accounts).not.toHaveBeenCalled()
    expect(mocks.contacts).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Test clinic')
  })
  it('reloads daily data when the chosen calendar date changes', async () => {
    mount(); await flushPromises(); mocks.appointments.mockClear()
    wrapper.vm.date = '2026-10-12'; await nextTick(); await flushPromises()
    expect(mocks.appointments).toHaveBeenCalledWith({ from: '2026-10-12', to: '2026-10-13' })
  })
  it('keeps a newly created walk-in patient in the booking selector', async () => {
    mount(); await flushPromises()
    wrapper.vm.startWalkInBooking({ id: 'new', profile_name: 'Walk-in', whatsapp_account: 'Clinic WhatsApp' })
    await nextTick()
    expect(wrapper.vm.contacts.some((c: any) => c.id === 'new')).toBe(true)
    expect(wrapper.vm.booking.contact_id).toBe('new')
  })
  it('clears the patient selection when switching WhatsApp accounts', async () => {
    mount(); await flushPromises()
    wrapper.vm.booking.contact_id = 'c1'; wrapper.vm.booking.whatsapp_account = 'Another account'; await nextTick()
    expect(wrapper.vm.booking.contact_id).toBe('')
  })
  it('rejects invalid setup values before sending a save', async () => {
    mount(); await flushPromises()
    wrapper.vm.profileForm.default_slot_minutes = 1
    await wrapper.vm.saveProfile()
    expect(mocks.save).not.toHaveBeenCalled()
    expect(wrapper.vm.profileError).toContain('5 to 240')
  })
  it('keeps a failed profile save visible and preserves the form', async () => {
    mocks.save.mockRejectedValue({ isAxiosError: true, response: { status: 500 } })
    mount(); await flushPromises(); wrapper.vm.profileOpen = true; await nextTick()
    await wrapper.vm.saveProfile()
    expect(wrapper.vm.profileOpen).toBe(true)
    expect(wrapper.vm.profileError).toContain('Could not save clinic setup')
  })
  it('ignores outdated availability responses after changing practitioners', async () => {
    let resolveOld!: (value: unknown) => void
    mocks.slots.mockImplementation((id: string) => id === 'p1'
      ? new Promise(r => { resolveOld = r })
      : Promise.resolve(response({ slots: [{ starts_at: '2026-09-20T06:00:00Z' }] })))
    mount(); await flushPromises()
    wrapper.vm.bookingOpen = true; wrapper.vm.booking.practitioner_id = 'p1'; await nextTick()
    wrapper.vm.booking.practitioner_id = 'p2'; await nextTick(); await flushPromises()
    resolveOld(response({ slots: [{ starts_at: '2026-09-20T01:00:00Z' }] })); await flushPromises()
    expect(wrapper.vm.slots[0].starts_at).toBe('2026-09-20T06:00:00Z')
  })
})
