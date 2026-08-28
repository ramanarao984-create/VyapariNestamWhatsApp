<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { CalendarDays, Clock3, MessageSquare, Plus, Users } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { PageHeader, ErrorState } from '@/components/shared'
import { accountsService, contactsService, clinicService, type ClinicAppointment, type ClinicPractitioner, type ClinicProfile, type ClinicService, type ClinicSlot, type ClinicWaitlistEntry } from '@/services/api'
import { getErrorMessage } from '@/lib/api-utils'

const profile = ref<ClinicProfile | null>(null)
const appointments = ref<ClinicAppointment[]>([])
const waitlist = ref<ClinicWaitlistEntry[]>([])
const loading = ref(true); const failed = ref(false)
const date = ref(new Date().toISOString().slice(0, 10))
const practitioners = ref<ClinicPractitioner[]>([]); const services = ref<ClinicService[]>([])
const accounts = ref<any[]>([]); const contacts = ref<any[]>([]); const slots = ref<ClinicSlot[]>([])
const bookingOpen = ref(false); const practitionerOpen = ref(false); const serviceOpen = ref(false); const bookingSaving = ref(false)
const practitionerForm = ref({ display_name: '', department: '' }); const serviceForm = ref({ name: '', description: '', duration_minutes: 20, buffer_minutes: 0 })
const booking = ref({ whatsapp_account: '', contact_id: '', practitioner_id: '', service_id: '', starts_at: '' })
const unpack = <T,>(response: any): T => response.data?.data ?? response.data
const nextDay = (day: string) => { const value = new Date(`${day}T00:00:00`); value.setDate(value.getDate() + 1); return value.toISOString().slice(0, 10) }
const patient = (item: ClinicAppointment) => item.contact?.profile_name || item.contact?.phone_number || 'Patient'
const time = (value: string) => new Intl.DateTimeFormat(undefined, { hour: 'numeric', minute: '2-digit' }).format(new Date(value))
const confirmed = computed(() => appointments.value.filter(x => x.status === 'confirmed').length)
const pending = computed(() => appointments.value.filter(x => x.status === 'pending').length)
const activeWaitlist = computed(() => waitlist.value.filter(x => x.status === 'waiting' || x.status === 'offered').length)
const actions = computed(() => [
  ...appointments.value.filter(x => x.status === 'pending').map(x => ({ key: x.id, label: `Confirm ${patient(x)}`, detail: `${time(x.starts_at)} · WhatsApp booking` })),
  ...waitlist.value.filter(x => x.status === 'offered').map(x => ({ key: x.id, label: 'Waitlist offer awaiting reply', detail: x.offer_expires_at ? `Expires ${time(x.offer_expires_at)}` : 'Awaiting reply' })),
])
async function load() {
  loading.value = true; failed.value = false
  try {
    const [p, a, w] = await Promise.all([clinicService.getProfile(), clinicService.listAppointments({ from: date.value, to: nextDay(date.value) }), clinicService.listWaitlist()])
    profile.value = unpack<{ configured: boolean; profile?: ClinicProfile }>(p).profile || null
    appointments.value = unpack<{ appointments: ClinicAppointment[] }>(a).appointments || []
    waitlist.value = unpack<{ entries: ClinicWaitlistEntry[] }>(w).entries || []
    const [practitionerResult, serviceResult, accountResult, contactResult] = await Promise.all([clinicService.listPractitioners(), clinicService.listServices(), accountsService.list(), contactsService.list({ limit: 100 })])
    practitioners.value = unpack<{ practitioners: ClinicPractitioner[] }>(practitionerResult).practitioners || []
    services.value = unpack<{ services: ClinicService[] }>(serviceResult).services || []
    accounts.value = unpack<any>(accountResult).accounts || []
    contacts.value = unpack<any>(contactResult).contacts || []
  } catch (error) { failed.value = true; toast.error(getErrorMessage(error, 'Could not load clinic workspace')) } finally { loading.value = false }
}
onMounted(load)
watch(() => [booking.value.practitioner_id, booking.value.service_id, date.value], async () => {
  booking.value.starts_at = ''; slots.value = []
  if (!booking.value.practitioner_id) return
  try { slots.value = unpack<{ slots: ClinicSlot[] }>(await clinicService.listSlots(booking.value.practitioner_id, { date: date.value, service_id: booking.value.service_id || undefined })).slots || [] } catch { slots.value = [] }
})
async function addPractitioner() { try { await clinicService.createPractitioner(practitionerForm.value); practitionerOpen.value = false; practitionerForm.value = { display_name: '', department: '' }; await load(); toast.success('Practitioner added') } catch (e) { toast.error(getErrorMessage(e, 'Could not add practitioner')) } }
async function addService() { try { await clinicService.createService(serviceForm.value); serviceOpen.value = false; serviceForm.value = { name: '', description: '', duration_minutes: 20, buffer_minutes: 0 }; await load(); toast.success('Service added') } catch (e) { toast.error(getErrorMessage(e, 'Could not add service')) } }
async function bookAppointment() { bookingSaving.value = true; try { await clinicService.createAppointment({ ...booking.value, service_id: booking.value.service_id || undefined }); bookingOpen.value = false; booking.value = { whatsapp_account: '', contact_id: '', practitioner_id: '', service_id: '', starts_at: '' }; await load(); toast.success('Appointment confirmed') } catch (e) { toast.error(getErrorMessage(e, 'That slot is no longer available')) } finally { bookingSaving.value = false } }
</script>

<template>
  <div class="flex h-full flex-col bg-background">
    <PageHeader title="Clinic workspace" description="Today’s appointments, WhatsApp replies, and reception actions." :icon="CalendarDays" icon-gradient="bg-gradient-to-br from-teal-500 to-emerald-600" />
    <ErrorState v-if="failed && !loading" class="flex-1" title="Couldn’t load the clinic workspace" description="Please try again." retry-label="Try again" @retry="load" />
    <main v-else class="flex-1 overflow-auto p-5 md:p-6"><section class="mx-auto max-w-7xl space-y-5">
      <Card v-if="!profile" class="border-teal-500/30"><CardHeader><CardTitle>Set up your clinic first</CardTitle><CardDescription>Create the clinic profile through the existing clinic API before enabling WhatsApp booking. Practitioners, services, and availability are then added here.</CardDescription></CardHeader><CardContent><p class="text-sm text-muted-foreground">The profile endpoint is ready; this screen stays read-only until a valid profile exists to prevent incomplete booking setup.</p></CardContent></Card>
      <template v-else>
        <div class="flex flex-wrap items-end justify-between gap-3"><div><h2 class="text-lg font-semibold">{{ profile.display_name }}</h2><p class="text-sm text-muted-foreground">Reception view · {{ profile.timezone }}</p></div><div class="flex gap-2"><Button v-if="practitioners.length && accounts.length && contacts.length" @click="bookingOpen = true"><Plus class="mr-2 h-4 w-4" />Book appointment</Button><Input v-model="date" type="date" class="w-auto" @change="load" /><Button variant="outline" @click="load">Refresh</Button></div></div>
        <div class="grid gap-3 sm:grid-cols-3"><Card><CardContent class="flex gap-3 p-4"><Users class="h-5 w-5 text-emerald-600" /><div><p class="text-2xl font-semibold">{{ confirmed }}</p><p class="text-sm text-muted-foreground">Confirmed</p></div></CardContent></Card><Card><CardContent class="flex gap-3 p-4"><MessageSquare class="h-5 w-5 text-amber-600" /><div><p class="text-2xl font-semibold">{{ pending }}</p><p class="text-sm text-muted-foreground">Need confirmation</p></div></CardContent></Card><Card><CardContent class="flex gap-3 p-4"><Users class="h-5 w-5 text-teal-600" /><div><p class="text-2xl font-semibold">{{ activeWaitlist }}</p><p class="text-sm text-muted-foreground">Active waitlist</p></div></CardContent></Card></div>
        <div class="grid gap-5 lg:grid-cols-[1.45fr_0.8fr]"><Card><CardHeader><CardTitle>Today’s schedule</CardTitle><CardDescription>Live, tenant-scoped appointment calendar.</CardDescription></CardHeader><CardContent class="space-y-2"><div v-if="!appointments.length" class="rounded-xl border border-dashed p-8 text-center text-sm text-muted-foreground">No appointments for this day.</div><div v-for="item in appointments" :key="item.id" class="flex items-center gap-3 rounded-xl border p-3"><Clock3 class="h-4 w-4 text-muted-foreground" /><div class="min-w-0 flex-1"><p class="font-medium">{{ patient(item) }}</p><p class="text-sm text-muted-foreground">{{ time(item.starts_at) }} · {{ item.practitioner?.display_name || 'Practitioner' }}</p></div><Badge variant="secondary">{{ item.status }}</Badge></div></CardContent></Card><Card><CardHeader><CardTitle>Next best actions</CardTitle><CardDescription>Items needing reception attention.</CardDescription></CardHeader><CardContent class="space-y-3"><p v-if="!actions.length" class="rounded-xl bg-muted/40 p-5 text-sm text-muted-foreground">All clear.</p><div v-for="action in actions" :key="action.key" class="rounded-xl border p-3"><p class="font-medium">{{ action.label }}</p><p class="mt-1 text-sm text-muted-foreground">{{ action.detail }}</p></div><RouterLink to="/contacts"><Button variant="outline" class="w-full">Open patient contacts</Button></RouterLink></CardContent></Card></div>
        <div class="grid gap-5 lg:grid-cols-2"><Card><CardHeader><div class="flex items-center justify-between"><div><CardTitle>Practitioners</CardTitle><CardDescription>People whose availability controls booking.</CardDescription></div><Button size="sm" variant="outline" @click="practitionerOpen = true"><Plus class="mr-1 h-4 w-4" />Add</Button></div></CardHeader><CardContent class="space-y-2"><p v-if="!practitioners.length" class="text-sm text-muted-foreground">Add a practitioner, then configure recurring availability.</p><div v-for="person in practitioners" :key="person.id" class="rounded-lg border p-3"><p class="font-medium">{{ person.display_name }}</p><p class="text-sm text-muted-foreground">{{ person.department || 'General practice' }}</p></div></CardContent></Card><Card><CardHeader><div class="flex items-center justify-between"><div><CardTitle>Services</CardTitle><CardDescription>Service duration determines safe slots.</CardDescription></div><Button size="sm" variant="outline" @click="serviceOpen = true"><Plus class="mr-1 h-4 w-4" />Add</Button></div></CardHeader><CardContent class="space-y-2"><p v-if="!services.length" class="text-sm text-muted-foreground">Optional: default slot duration is used until services are added.</p><div v-for="service in services" :key="service.id" class="rounded-lg border p-3"><p class="font-medium">{{ service.name }}</p><p class="text-sm text-muted-foreground">{{ service.duration_minutes }} minutes<span v-if="service.buffer_minutes"> + {{ service.buffer_minutes }} minute buffer</span></p></div></CardContent></Card></div>
      </template>
    </section></main>
    <Dialog v-model:open="practitionerOpen"><DialogContent><DialogHeader><DialogTitle>Add practitioner</DialogTitle><DialogDescription>Only operational scheduling details are stored.</DialogDescription></DialogHeader><div class="space-y-3"><div><Label>Name</Label><Input v-model="practitionerForm.display_name" /></div><div><Label>Department</Label><Input v-model="practitionerForm.department" placeholder="Optional" /></div></div><DialogFooter><Button :disabled="!practitionerForm.display_name" @click="addPractitioner">Add practitioner</Button></DialogFooter></DialogContent></Dialog>
    <Dialog v-model:open="serviceOpen"><DialogContent><DialogHeader><DialogTitle>Add service</DialogTitle></DialogHeader><div class="space-y-3"><div><Label>Service name</Label><Input v-model="serviceForm.name" /></div><div><Label>Duration (minutes)</Label><Input v-model.number="serviceForm.duration_minutes" type="number" min="5" /></div><div><Label>Buffer (minutes)</Label><Input v-model.number="serviceForm.buffer_minutes" type="number" min="0" /></div></div><DialogFooter><Button :disabled="!serviceForm.name" @click="addService">Add service</Button></DialogFooter></DialogContent></Dialog>
    <Dialog v-model:open="bookingOpen"><DialogContent><DialogHeader><DialogTitle>Book appointment</DialogTitle><DialogDescription>Final availability is checked again by the server when you confirm.</DialogDescription></DialogHeader><div class="space-y-3"><div><Label>WhatsApp account</Label><select v-model="booking.whatsapp_account" class="w-full rounded-md border bg-background p-2"><option value="">Select account</option><option v-for="account in accounts" :key="account.id" :value="account.name">{{ account.name }}</option></select></div><div><Label>Patient</Label><select v-model="booking.contact_id" class="w-full rounded-md border bg-background p-2"><option value="">Select patient</option><option v-for="contact in contacts.filter((c:any) => !booking.whatsapp_account || c.whatsapp_account === booking.whatsapp_account)" :key="contact.id" :value="contact.id">{{ contact.profile_name || contact.phone_number }}</option></select></div><div><Label>Practitioner</Label><select v-model="booking.practitioner_id" class="w-full rounded-md border bg-background p-2"><option value="">Select practitioner</option><option v-for="person in practitioners" :key="person.id" :value="person.id">{{ person.display_name }}</option></select></div><div><Label>Service</Label><select v-model="booking.service_id" class="w-full rounded-md border bg-background p-2"><option value="">Default slot</option><option v-for="service in services" :key="service.id" :value="service.id">{{ service.name }}</option></select></div><div><Label>Available time on {{ date }}</Label><select v-model="booking.starts_at" class="w-full rounded-md border bg-background p-2" :disabled="!booking.practitioner_id"><option value="">Select a time</option><option v-for="slot in slots" :key="slot.starts_at" :value="slot.starts_at">{{ time(slot.starts_at) }}</option></select></div></div><DialogFooter><Button :disabled="bookingSaving || !booking.whatsapp_account || !booking.contact_id || !booking.practitioner_id || !booking.starts_at" @click="bookAppointment">{{ bookingSaving ? 'Confirming…' : 'Confirm appointment' }}</Button></DialogFooter></DialogContent></Dialog>
  </div>
</template>
