<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import {
  CalendarDays,
  CheckCircle2,
  Clock3,
  MessageSquare,
  Pencil,
  Plus,
  UserPlus,
  Users,
  Loader2,
  ChevronLeft,
  ChevronRight,
} from "lucide-vue-next";
import { toast } from "vue-sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  CreateContactDialog,
  PageHeader,
  ErrorState,
} from "@/components/shared";
import {
  accountsService,
  contactsService,
  clinicService,
  type ClinicAppointment,
  type ClinicPractitioner,
  type ClinicProfile,
  type ClinicService,
  type ClinicSlot,
  type ClinicWaitlistEntry,
} from "@/services/api";
import { getErrorMessage } from "@/lib/api-utils";
import { clinicDate, validCalendarDate } from "@/lib/clinic-date";
import { useAuthStore } from "@/stores/auth";
const auth = useAuthStore();
const canWrite = computed(() => auth.hasPermission('clinic', 'write'));
const canReadContacts = computed(() => auth.hasPermission('contacts', 'read'));
const canWriteContacts = computed(() => auth.hasPermission('contacts', 'write'));
const canReadAccounts = computed(() => auth.hasPermission('accounts', 'read'));
const zone = computed(() => profile.value?.timezone || 'Asia/Kolkata');
const loadError = ref('');
const warnings = ref<string[]>([]);
const calendarFailed = ref(false);
const waitlistFailed = ref(false);
const slotsLoading = ref(false);
const slotsError = ref('');
const rescheduleLoading = ref(false);
const rescheduleError = ref('');
const profileError = ref('');
const setupSaving = ref(false);
let loadRequest = 0;
let rescheduleRequest = 0;
let initialDateSet = false;

const profile = ref<ClinicProfile | null>(null);
const appointments = ref<ClinicAppointment[]>([]);
const weekAppointments = ref<ClinicAppointment[]>([]);
const waitlist = ref<ClinicWaitlistEntry[]>([]);
const loading = ref(true);
const failed = ref(false);
const date = ref(clinicDate(new Date(), 'Asia/Kolkata'));
const practitioners = ref<ClinicPractitioner[]>([]);
const services = ref<ClinicService[]>([]);
const accounts = ref<any[]>([]);
const contacts = ref<any[]>([]);
const slots = ref<ClinicSlot[]>([]);
const bookingOpen = ref(false);
const practitionerOpen = ref(false);
const serviceOpen = ref(false);
const bookingSaving = ref(false);
const walkInOpen = ref(false);
const profileOpen = ref(false);
const rescheduleOpen = ref(false);
const cancelOpen = ref(false);
const selectedAppointment = ref<ClinicAppointment | null>(null);
const appointmentSaving = ref(false);
const availabilityOpen = ref(false);
const availabilityPractitioner = ref<ClinicPractitioner | null>(null);
const availabilityRules = ref<
  Array<{
    id: string;
    day_of_week: number;
    start_minute: number;
    end_minute: number;
    slot_interval_minutes: number;
  }>
>([]);
const practitionerForm = ref({ display_name: "", department: "" });
const serviceForm = ref({
  name: "",
  description: "",
  duration_minutes: 20,
  buffer_minutes: 0,
});
const availabilityForm = ref({
  day_of_week: 1,
  start: "09:00",
  end: "17:00",
  slot_interval_minutes: 20,
});
const profileForm = ref({
  display_name: "",
  timezone: "Asia/Kolkata",
  address: "",
  reception_phone: "",
  booking_enabled: false,
  default_slot_minutes: 20,
  advance_booking_days: 30,
  minimum_notice_minutes: 30,
  cancellation_cutoff_minutes: 120,
});
const rescheduleDate = ref(date.value);
const rescheduleStartsAt = ref("");
const rescheduleSlots = ref<ClinicSlot[]>([]);
const cancellationReason = ref("");
const booking = ref({
  whatsapp_account: "",
  contact_id: "",
  practitioner_id: "",
  service_id: "",
  starts_at: "",
});
const unpack = <T,>(response: any): T => response.data?.data ?? response.data;
// Calendar requests use date-only values. UTC arithmetic prevents browser
// time zones from shifting either endpoint while a receptionist is viewing a
// week, keeping every request inside the API's 31-day safety boundary.
const utcDate = (day: string) => {
  const [year, month, date] = day.split("-").map(Number);
  return new Date(Date.UTC(year, month - 1, date));
};
const asDate = (value: Date) => value.toISOString().slice(0, 10);
const nextDay = (day: string) => {
  const value = utcDate(day);
  value.setUTCDate(value.getUTCDate() + 1);
  return asDate(value);
};
const weekStart = (day: string) => {
  const value = utcDate(day);
  value.setUTCDate(value.getUTCDate() - value.getUTCDay());
  return asDate(value);
};
const addDays = (day: string, amount: number) => {
  const value = utcDate(day);
  value.setUTCDate(value.getUTCDate() + amount);
  return asDate(value);
};
const selectedDateLabel = computed(() => new Intl.DateTimeFormat(undefined, {
  weekday: 'long', month: 'short', day: 'numeric', timeZone: 'UTC',
}).format(utcDate(date.value)));
function selectDate(day: string) { if (validCalendarDate(day)) date.value = day; }
const weekDays = computed(() =>
  Array.from({ length: 7 }, (_, index) => {
    const iso = addDays(weekStart(date.value), index);
    const value = new Date(`${iso}T00:00:00`);
    return {
      iso,
      short: new Intl.DateTimeFormat(undefined, { weekday: "short" }).format(
        value,
      ),
      day: value.getDate(),
    };
  }),
);
const appointmentsForDay = (day: string) =>
  weekAppointments.value.filter(
    (item) =>
      clinicDate(item.starts_at, zone.value) === day && item.status !== "cancelled",
  );
const isToday = (day: string) => day === clinicDate(new Date(), zone.value);
const patient = (item: ClinicAppointment | null) =>
  item?.contact?.profile_name || item?.contact?.phone_number || "Patient";
const time = (value: string) =>
  new Intl.DateTimeFormat(undefined, {
    timeZone: zone.value,
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(value));
const confirmed = computed(
  () => appointments.value.filter((x) => x.status === "confirmed").length,
);
const pending = computed(
  () => appointments.value.filter((x) => x.status === "pending").length,
);
const activeWaitlist = computed(
  () =>
    waitlist.value.filter(
      (x) => x.status === "waiting" || x.status === "offered",
    ).length,
);
const actions = computed(() => [
  ...appointments.value
    .filter((x) => x.status === "pending")
    .map((x) => ({
      key: x.id,
      label: `Confirm ${patient(x)}`,
      detail: `${time(x.starts_at)} · WhatsApp booking`,
    })),
  ...waitlist.value
    .filter((x) => x.status === "offered")
    .map((x) => ({
      key: x.id,
      label: "Waitlist offer awaiting reply",
      detail: x.offer_expires_at
        ? `Expires ${time(x.offer_expires_at)}`
        : "Awaiting reply",
    })),
]);
async function load() {
  if (!validCalendarDate(date.value)) return;
  const request = ++loadRequest;
  loading.value = true;
  failed.value = false;
  loadError.value = '';
  warnings.value = [];
  try {
    const response = await clinicService.getProfile();
    if (request !== loadRequest) return;
    profile.value = unpack<{ configured: boolean; profile?: ClinicProfile }>(response).profile || null;
    if (!profile.value) {
      appointments.value = []; weekAppointments.value = []; waitlist.value = [];
      return;
    }
    if (!initialDateSet) {
      initialDateSet = true;
      const today = clinicDate(new Date(), zone.value);
      if (date.value !== today) { date.value = today; return; }
    }
    Object.assign(profileForm.value, profile.value);
    const day = date.value;
    const results = await Promise.allSettled([
      clinicService.listAppointments({ from: day, to: nextDay(day) }),
      clinicService.listAppointments({ from: weekStart(day), to: addDays(weekStart(day), 7) }),
      clinicService.listWaitlist(), clinicService.listPractitioners(), clinicService.listServices(),
      canReadAccounts.value ? accountsService.list() : Promise.resolve({ data: { accounts: [] } }),
      canReadContacts.value ? contactsService.list({ limit: 100 }) : Promise.resolve({ data: { contacts: [] } }),
    ]);
    if (request !== loadRequest) return;
    const names = ['Daily schedule', 'Week calendar', 'Waitlist', 'Practitioners', 'Services', 'WhatsApp accounts', 'Patient contacts'];
    const keys = ['appointments', 'appointments', 'entries', 'practitioners', 'services', 'accounts', 'contacts'];
    const targets = [appointments, weekAppointments, waitlist, practitioners, services, accounts, contacts];
    results.forEach((result, index) => {
      targets[index].value = result.status === 'fulfilled' ? unpack<any>(result.value)[keys[index]] || [] : [];
      if (result.status === 'rejected') warnings.value.push(`${names[index]} could not load. ${getErrorMessage(result.reason, 'Please try again.')}`);
    });
    calendarFailed.value = results[1].status === 'rejected';
    waitlistFailed.value = results[2].status === 'rejected';
    if (results[0].status === 'rejected') throw results[0].reason;
  } catch (error) {
    if (request !== loadRequest) return;
    failed.value = true;
    loadError.value = getErrorMessage(error, 'Could not load clinic workspace. Please try again.');
  } finally {
    if (request === loadRequest) loading.value = false;
  }
}
onMounted(load);
watch(date, (day, previous) => {
  if (!validCalendarDate(day)) { date.value = previous; return; }
  void load();
});
watch(() => booking.value.whatsapp_account, () => {
  const selected = contacts.value.find(contact => contact.id === booking.value.contact_id);
  if (selected && selected.whatsapp_account !== booking.value.whatsapp_account) booking.value.contact_id = '';
});
watch(
  () => [booking.value.practitioner_id, booking.value.service_id, date.value, bookingOpen.value],
  async (_, __, onCleanup) => {
    let current = true;
    onCleanup(() => { current = false; });
    booking.value.starts_at = ''; slots.value = []; slotsError.value = ''; slotsLoading.value = false;
    if (!bookingOpen.value || !booking.value.practitioner_id || !validCalendarDate(date.value)) return;
    slotsLoading.value = true;
    try {
      const response = await clinicService.listSlots(booking.value.practitioner_id, { date: date.value, service_id: booking.value.service_id || undefined });
      if (current) slots.value = unpack<{ slots: ClinicSlot[] }>(response).slots || [];
    } catch (error) {
      if (current) slotsError.value = getErrorMessage(error, 'Could not load available times. Try another date or reopen this form.');
    } finally { if (current) slotsLoading.value = false; }
  },
);
watch(profileOpen, () => { profileError.value = ''; });
async function addPractitioner() {
  if (setupSaving.value || !canWrite.value || !practitionerForm.value.display_name.trim()) return;
  setupSaving.value = true;
  try {
    await clinicService.createPractitioner(practitionerForm.value);
    practitionerOpen.value = false;
    practitionerForm.value = { display_name: "", department: "" };
    await load();
    toast.success("Practitioner added");
  } catch (e) {
    toast.error(getErrorMessage(e, "Could not add practitioner"));
  } finally { setupSaving.value = false; }
}
async function addService() {
  if (setupSaving.value || !canWrite.value || !serviceForm.value.name.trim()) return;
  setupSaving.value = true;
  try {
    await clinicService.createService(serviceForm.value);
    serviceOpen.value = false;
    serviceForm.value = {
      name: "",
      description: "",
      duration_minutes: 20,
      buffer_minutes: 0,
    };
    await load();
    toast.success("Service added");
  } catch (e) {
    toast.error(getErrorMessage(e, "Could not add service"));
  } finally { setupSaving.value = false; }
}
async function bookAppointment() {
  if (bookingSaving.value || !canWrite.value || slotsLoading.value || !slots.value.some(slot => slot.starts_at === booking.value.starts_at)) return;
  bookingSaving.value = true;
  try {
    await clinicService.createAppointment({
      ...booking.value,
      service_id: booking.value.service_id || undefined,
    });
    bookingOpen.value = false;
    booking.value = {
      whatsapp_account: "",
      contact_id: "",
      practitioner_id: "",
      service_id: "",
      starts_at: "",
    };
    await load();
    toast.success("Appointment confirmed");
  } catch (e) {
    toast.error(getErrorMessage(e, "That slot is no longer available"));
  } finally {
    bookingSaving.value = false;
  }
}
function openBookingForDay(day: string) {
  date.value = day;
  bookingOpen.value = true;
}
function startWalkInBooking(contact: any) {
  if (!contacts.value.some(item => item.id === contact.id)) contacts.value.push(contact);
  booking.value.contact_id = contact.id;
  booking.value.whatsapp_account =
    contact.whatsapp_account || accounts.value[0]?.name || "";
  walkInOpen.value = false;
  bookingOpen.value = true;
}
const dayNames = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];
const asMinutes = (value: string) => {
  const [hour, minute] = value.split(":").map(Number);
  return hour * 60 + minute;
};
const asTime = (minutes: number) =>
  `${String(Math.floor(minutes / 60)).padStart(2, "0")}:${String(minutes % 60).padStart(2, "0")}`;
async function openAvailability(person: ClinicPractitioner) {
  availabilityRules.value = [];
  availabilityPractitioner.value = person;
  availabilityOpen.value = true;
  try {
    availabilityRules.value =
      unpack<{ availability_rules: typeof availabilityRules.value }>(
        await clinicService.listAvailability(person.id),
      ).availability_rules || [];
  } catch (e) {
    toast.error(getErrorMessage(e, "Could not load availability"));
  }
}
async function addAvailability() {
  if (!availabilityPractitioner.value || setupSaving.value || !canWrite.value) return;
  setupSaving.value = true;
  try {
    await clinicService.createAvailability(availabilityPractitioner.value.id, {
      day_of_week: availabilityForm.value.day_of_week,
      start_minute: asMinutes(availabilityForm.value.start),
      end_minute: asMinutes(availabilityForm.value.end),
      slot_interval_minutes: availabilityForm.value.slot_interval_minutes,
    });
    await openAvailability(availabilityPractitioner.value);
    toast.success("Availability saved");
  } catch (e) {
    toast.error(getErrorMessage(e, "Could not save availability"));
  } finally { setupSaving.value = false; }
}
async function saveProfile() {
  if (appointmentSaving.value || !canWrite.value) return;
  profileError.value = '';
  profileForm.value.display_name = profileForm.value.display_name.trim();
  profileForm.value.timezone = profileForm.value.timezone.trim();
  if (!profileForm.value.display_name) { profileError.value = 'Enter a clinic name.'; return; }
  try { new Intl.DateTimeFormat('en', { timeZone: profileForm.value.timezone }); }
  catch { profileError.value = 'Enter a valid time zone, such as Asia/Kolkata.'; return; }
  const minutes = Number(profileForm.value.default_slot_minutes);
  if (!Number.isInteger(minutes) || minutes < 5 || minutes > 240) {
    profileError.value = 'Appointment duration must be a whole number from 5 to 240 minutes.'; return;
  }
  profileForm.value.default_slot_minutes = minutes;
  appointmentSaving.value = true;
  try {
    await clinicService.updateProfile(profileForm.value);
    profileOpen.value = false;
    await load();
    toast.success("Clinic profile saved");
  } catch (e) {
    profileError.value = getErrorMessage(e, "Could not save clinic setup. Please try again.");
  } finally {
    appointmentSaving.value = false;
  }
}
async function loadRescheduleSlots() {
  const request = ++rescheduleRequest;
  rescheduleStartsAt.value = ''; rescheduleSlots.value = []; rescheduleError.value = ''; rescheduleLoading.value = false;
  if (!selectedAppointment.value || !validCalendarDate(rescheduleDate.value)) return;
  rescheduleLoading.value = true;
  try {
    const response = await clinicService.listSlots(selectedAppointment.value.practitioner_id, { date: rescheduleDate.value, service_id: selectedAppointment.value.service_id });
    if (request === rescheduleRequest) rescheduleSlots.value = unpack<{ slots: ClinicSlot[] }>(response).slots || [];
  } catch (error) {
    if (request === rescheduleRequest) rescheduleError.value = getErrorMessage(error, 'Could not load available times. Please try again.');
  } finally { if (request === rescheduleRequest) rescheduleLoading.value = false; }
}
async function openReschedule(item: ClinicAppointment) {
  if (!canWrite.value || !['pending', 'confirmed'].includes(item.status)) return;
  selectedAppointment.value = item;
  rescheduleDate.value = clinicDate(item.starts_at, zone.value);
  rescheduleOpen.value = true;
  await loadRescheduleSlots();
}
async function reschedule() {
  if (!selectedAppointment.value || !rescheduleStartsAt.value || appointmentSaving.value || rescheduleLoading.value || !canWrite.value) return;
  appointmentSaving.value = true;
  try {
    await clinicService.rescheduleAppointment(
      selectedAppointment.value.id,
      rescheduleStartsAt.value,
    );
    rescheduleOpen.value = false;
    await load();
    toast.success("Appointment rescheduled");
  } catch (e) {
    toast.error(getErrorMessage(e, "That slot is no longer available"));
  } finally {
    appointmentSaving.value = false;
  }
}
function openCancel(item: ClinicAppointment) {
  selectedAppointment.value = item;
  cancellationReason.value = "";
  cancelOpen.value = true;
}
async function cancel() {
  if (!selectedAppointment.value || appointmentSaving.value || !canWrite.value) return;
  appointmentSaving.value = true;
  try {
    await clinicService.cancelAppointment(
      selectedAppointment.value.id,
      cancellationReason.value,
    );
    cancelOpen.value = false;
    await load();
    toast.success("Appointment cancelled");
  } catch (e) {
    toast.error(getErrorMessage(e, "Could not cancel appointment"));
  } finally {
    appointmentSaving.value = false;
  }
}
</script>

<template>
  <div class="clinic-workspace flex h-full flex-col bg-background">
    <PageHeader
      title="Clinic workspace"
      description="Today’s appointments, WhatsApp replies, and reception actions."
      :icon="CalendarDays"
      icon-gradient="bg-gradient-to-br from-teal-500 to-emerald-600"
    />
    <ErrorState
      v-if="failed && !loading"
      class="flex-1"
      title="Couldn’t load the clinic workspace"
      :description="loadError"
      retry-label="Try again"
      @retry="load"
    />
    <div v-else-if="loading" class="flex flex-1 items-center justify-center gap-3 text-muted-foreground" role="status">
      <Loader2 class="h-5 w-5 animate-spin" /> Loading your clinic workspace…
    </div>
    <main v-else class="flex-1 overflow-auto p-5 md:p-8">
      <section class="clinic-shell mx-auto max-w-7xl space-y-6">
        <div v-if="warnings.length && profile" class="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm" role="alert">
          <p class="font-semibold">Some information is unavailable</p>
          <p v-for="warning in warnings" :key="warning" class="mt-1">{{ warning }}</p>
          <Button class="mt-3" variant="outline" size="sm" @click="load">Try again</Button>
        </div>
        <Card v-if="!profile" class="setup-card border-teal-500/30"
          ><CardHeader
            ><span class="eyebrow">Nestam AI clinic operations</span
            ><CardTitle class="text-2xl">Set up your clinic first</CardTitle
            ><CardDescription
              >Bring your team, appointment calendar, and WhatsApp enquiries into one calm workspace.</CardDescription
            ></CardHeader
          ><CardContent
            ><Button v-if="canWrite" class="premium-primary" @click="profileOpen = true"
              >Set up clinic</Button><p v-else class="text-sm text-muted-foreground">Ask your administrator to complete clinic setup.</p></CardContent
          ></Card
        >
        <template v-else>
          <div
            class="clinic-hero flex flex-wrap items-end justify-between gap-4"
          >
            <div>
              <span class="eyebrow">Reception command centre</span>
              <h2 class="mt-1 text-2xl font-semibold tracking-tight">
                {{ profile.display_name }}
              </h2>
              <p class="mt-1 text-sm text-muted-foreground">
                {{ selectedDateLabel }} · {{ profile.timezone }}
              </p>
            </div>
            <div class="flex flex-wrap gap-2">
              <Button v-if="canWrite" variant="outline" size="sm" @click="profileOpen = true"
                ><Pencil class="mr-1 h-4 w-4" />Clinic setup</Button
              ><Button
                v-if="
                  canWrite && practitioners.length && accounts.length && contacts.length
                "
                class="premium-primary"
                @click="bookingOpen = true"
                ><Plus class="mr-2 h-4 w-4" />Book appointment</Button
              ><Input
                v-model="date"
                type="date"
                class="date-control w-auto"
                aria-label="Schedule date"
              /><Button variant="outline" @click="load">Refresh</Button>
            </div>
          </div>
          <Card class="daily-desk"
            ><CardHeader
              ><span class="eyebrow">Your simple daily routine</span
              ><CardTitle class="mt-1"
                >A smoother day at reception.</CardTitle
              ><CardDescription
                >Keep conversations, patient records, and appointments in sync.</CardDescription
              ></CardHeader
            ><CardContent class="daily-steps"
              ><div class="daily-step">
                <span class="step-number">1</span>
                <div>
                  <p class="font-semibold">WhatsApp patient</p>
                  <p class="text-sm text-muted-foreground">
                    Open the inbox, answer the enquiry, then book if needed.
                  </p>
                </div>
                <RouterLink to="/chat"
                  ><Button variant="outline" size="sm"
                    ><MessageSquare class="mr-1.5 h-4 w-4" />Open inbox</Button
                  ></RouterLink
                >
              </div>
              <div class="daily-step">
                <span class="step-number">2</span>
                <div>
                  <p class="font-semibold">Walk-in patient</p>
                  <p class="text-sm text-muted-foreground">
                    Add name and phone once, choose a slot, and you are done.
                  </p>
                </div>
                <Button
                  size="sm"
                  class="premium-primary"
                  v-if="canWriteContacts" @click="walkInOpen = true"
                  ><UserPlus class="mr-1.5 h-4 w-4" />New walk-in</Button
                >
              </div>
              <div class="daily-step">
                <span class="step-number">3</span>
                <div>
                  <p class="font-semibold">Keep the day moving</p>
                  <p class="text-sm text-muted-foreground">
                    Use the queue below for confirmations, reschedules, and
                    follow-ups.
                  </p>
                </div>
                <CheckCircle2
                  class="h-5 w-5 text-emerald-600"
                /></div></CardContent
          ></Card>
          <div class="grid gap-4 sm:grid-cols-3">
            <Card class="metric-card metric-confirmed"
              ><CardContent class="flex gap-3 p-5"
                ><Users class="metric-icon h-5 w-5" />
                <div>
                  <p class="text-3xl font-semibold tracking-tight">
                    {{ confirmed }}
                  </p>
                  <p class="text-sm text-muted-foreground">Confirmed appointments</p>
                </div></CardContent
              ></Card
            ><Card class="metric-card metric-pending"
              ><CardContent class="flex gap-3 p-5"
                ><MessageSquare class="metric-icon h-5 w-5" />
                <div>
                  <p class="text-3xl font-semibold tracking-tight">
                    {{ pending }}
                  </p>
                  <p class="text-sm text-muted-foreground">Need confirmation</p>
                </div></CardContent
              ></Card
            ><Card class="metric-card metric-waitlist"
              ><CardContent class="flex gap-3 p-5"
                ><Users class="metric-icon h-5 w-5" />
                <div>
                  <p class="text-3xl font-semibold tracking-tight">
                    {{ waitlistFailed ? '—' : activeWaitlist }}
                  </p>
                  <p class="text-sm text-muted-foreground">Active waitlist</p>
                </div></CardContent
              ></Card
            >
          </div>
          <Card class="week-calendar-card"
            ><CardHeader
              ><div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <span class="eyebrow">Schedule</span
                  ><CardTitle class="mt-1">Week at a glance</CardTitle
                  ><CardDescription
                    >Choose a day to inspect its schedule, or use an open day to
                    start a booking.</CardDescription
                  >
                </div>
                <Badge variant="outline" class="calendar-legend"
                  >{{ calendarFailed ? 'Calendar unavailable' : `${weekAppointments.filter(item => item.status !== 'cancelled').length} appointments this week` }}</Badge
                >
              </div></CardHeader
            ><CardContent
              ><div class="mb-4 flex items-center justify-between gap-2">
                <Button variant="outline" size="sm" aria-label="Previous week" @click="selectDate(addDays(date, -7))"><ChevronLeft class="h-4 w-4" /></Button>
                <Button variant="ghost" size="sm" @click="selectDate(clinicDate(new Date(), zone))">Today</Button>
                <Button variant="outline" size="sm" aria-label="Next week" @click="selectDate(addDays(date, 7))"><ChevronRight class="h-4 w-4" /></Button>
              </div><div v-if="!calendarFailed" class="week-calendar">
                <div
                  v-for="day in weekDays"
                  :key="day.iso"
                  class="calendar-day"
                  :class="{
                    'calendar-day--selected': day.iso === date,
                    'calendar-day--today': isToday(day.iso),
                  }"
                  role="button"
                  tabindex="0"
                  :aria-label="`View schedule for ${day.iso}`"
                  :aria-pressed="day.iso === date"
                  @click="selectDate(day.iso)"
                  @keydown.enter.self="selectDate(day.iso)"
                  @keydown.space.self.prevent="selectDate(day.iso)"
                >
                  <div class="calendar-day-header">
                    <span>{{ day.short }}</span
                    ><strong>{{ day.day }}</strong>
                  </div>
                  <div class="calendar-events">
                    <div
                      v-for="item in appointmentsForDay(day.iso).slice(0, 4)"
                      :key="item.id"
                      class="calendar-event"
                      :class="`calendar-event--${item.status}`"
                      role="button" tabindex="0"
                    @keydown.enter.stop="openReschedule(item)"
                    @keydown.space.stop.prevent="openReschedule(item)"
                    @click.stop="openReschedule(item)"
                    >
                      <span>{{ time(item.starts_at) }}</span
                      ><b>{{ patient(item) }}</b
                      ><small>{{
                        item.practitioner?.display_name || "Practitioner"
                      }}</small>
                    </div>
                    <button
                      v-if="canWrite && !appointmentsForDay(day.iso).length"
                      type="button"
                      class="calendar-open-slot"
                      @click.stop="openBookingForDay(day.iso)"
                    >
                      <Plus class="h-3.5 w-3.5" />Book slot
                    </button>
                    <p
                      v-else-if="appointmentsForDay(day.iso).length > 4"
                      class="calendar-more"
                    >
                      +{{ appointmentsForDay(day.iso).length - 4 }} more
                    </p>
                  </div>
                </div>
              </div></CardContent
            ></Card
          >
          <div class="grid gap-5 lg:grid-cols-[1.45fr_0.8fr]">
            <Card
              ><CardHeader
                ><CardTitle>{{ isToday(date) ? "Today’s schedule" : selectedDateLabel }}</CardTitle
                ><CardDescription
                  >Appointments and patient arrivals for the selected day.</CardDescription
                ></CardHeader
              ><CardContent class="space-y-2"
                ><div
                  v-if="!appointments.length"
                  class="rounded-xl border border-dashed p-8 text-center text-sm text-muted-foreground"
                >
                  No appointments for this day.
                </div>
                <div
                  v-for="item in appointments"
                  :key="item.id"
                  class="flex items-center gap-3 rounded-xl border p-3"
                >
                  <Clock3 class="h-4 w-4 text-muted-foreground" />
                  <div class="min-w-0 flex-1">
                    <p class="font-medium">{{ patient(item) }}</p>
                    <p class="text-sm text-muted-foreground">
                      {{ time(item.starts_at) }} ·
                      {{ item.practitioner?.display_name || "Practitioner" }}
                    </p>
                  </div>
                  <Badge variant="secondary">{{ item.status }}</Badge>
                  <div
                    v-if="
                      canWrite && (item.status === 'pending' || item.status === 'confirmed')
                    "
                    class="flex gap-1"
                  >
                    <Button
                      size="sm"
                      variant="outline"
                      @click="openReschedule(item)"
                      >Move</Button
                    ><Button size="sm" variant="ghost" @click="openCancel(item)"
                      >Cancel</Button
                    >
                  </div>
                </div></CardContent
              ></Card
            ><Card
              ><CardHeader
                ><CardTitle>Next best actions</CardTitle
                ><CardDescription
                  >Items needing reception attention.</CardDescription
                ></CardHeader
              ><CardContent class="space-y-3"
                ><p
                  v-if="!actions.length"
                  class="rounded-xl bg-muted/40 p-5 text-sm text-muted-foreground"
                >
                  All clear.
                </p>
                <div
                  v-for="action in actions"
                  :key="action.key"
                  class="rounded-xl border p-3"
                >
                  <p class="font-medium">{{ action.label }}</p>
                  <p class="mt-1 text-sm text-muted-foreground">
                    {{ action.detail }}
                  </p>
                </div>
                <RouterLink to="/contacts"
                  ><Button variant="outline" class="w-full"
                    >Open patient contacts</Button
                  ></RouterLink
                ></CardContent
              ></Card
            >
          </div>
          <div class="grid gap-5 lg:grid-cols-2">
            <Card
              ><CardHeader
                ><div class="flex items-center justify-between">
                  <div>
                    <CardTitle>Practitioners</CardTitle
                    ><CardDescription
                      >People whose availability controls
                      booking.</CardDescription
                    >
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    v-if="canWrite" @click="practitionerOpen = true"
                    ><Plus class="mr-1 h-4 w-4" />Add</Button
                  >
                </div></CardHeader
              ><CardContent class="space-y-2"
                ><p
                  v-if="!practitioners.length"
                  class="text-sm text-muted-foreground"
                >
                  Add a practitioner, then configure recurring availability.
                </p>
                <div
                  v-for="person in practitioners"
                  :key="person.id"
                  class="flex items-center justify-between rounded-lg border p-3"
                >
                  <div>
                    <p class="font-medium">{{ person.display_name }}</p>
                    <p class="text-sm text-muted-foreground">
                      {{ person.department || "General practice" }}
                    </p>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    v-if="canWrite" @click="openAvailability(person)"
                    >Hours</Button
                  >
                </div></CardContent
              ></Card
            ><Card
              ><CardHeader
                ><div class="flex items-center justify-between">
                  <div>
                    <CardTitle>Services</CardTitle
                    ><CardDescription
                      >Service duration determines safe slots.</CardDescription
                    >
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    v-if="canWrite" @click="serviceOpen = true"
                    ><Plus class="mr-1 h-4 w-4" />Add</Button
                  >
                </div></CardHeader
              ><CardContent class="space-y-2"
                ><p
                  v-if="!services.length"
                  class="text-sm text-muted-foreground"
                >
                  Optional: default slot duration is used until services are
                  added.
                </p>
                <div
                  v-for="service in services"
                  :key="service.id"
                  class="rounded-lg border p-3"
                >
                  <p class="font-medium">{{ service.name }}</p>
                  <p class="text-sm text-muted-foreground">
                    {{ service.duration_minutes }} minutes<span
                      v-if="service.buffer_minutes"
                    >
                      + {{ service.buffer_minutes }} minute buffer</span
                    >
                  </p>
                </div></CardContent
              ></Card
            >
          </div>
        </template>
      </section>
    </main>
    <Dialog v-model:open="practitionerOpen"
      ><DialogContent
        ><DialogHeader
          ><DialogTitle>Add practitioner</DialogTitle
          ><DialogDescription
            >Only operational scheduling details are stored.</DialogDescription
          ></DialogHeader
        >
        <div class="space-y-3">
          <div>
            <Label for="field-practitionerForm-display-name">Name</Label><Input id="field-practitionerForm-display-name" v-model="practitionerForm.display_name" />
          </div>
          <div>
            <Label for="field-practitionerForm-department">Department</Label><Input id="field-practitionerForm-department"
              v-model="practitionerForm.department"
              placeholder="Optional"
            />
          </div>
        </div>
        <DialogFooter
          ><Button
            :disabled="setupSaving || !practitionerForm.display_name.trim()"
            @click="addPractitioner"
            >Add practitioner</Button
          ></DialogFooter
        ></DialogContent
      ></Dialog
    >
    <CreateContactDialog
      :open="walkInOpen"
      @update:open="walkInOpen = $event"
      @created="startWalkInBooking"
    />
    <Dialog v-model:open="serviceOpen"
      ><DialogContent
        ><DialogHeader><DialogTitle>Add service</DialogTitle></DialogHeader>
        <div class="space-y-3">
          <div>
            <Label for="field-serviceForm-name">Service name</Label><Input id="field-serviceForm-name" v-model="serviceForm.name" />
          </div>
          <div>
            <Label for="field-serviceForm-duration-minutes">Duration (minutes)</Label><Input id="field-serviceForm-duration-minutes"
              v-model.number="serviceForm.duration_minutes"
              type="number"
              min="5" max="240"
            />
          </div>
          <div>
            <Label for="field-serviceForm-buffer-minutes">Buffer (minutes)</Label><Input id="field-serviceForm-buffer-minutes"
              v-model.number="serviceForm.buffer_minutes"
              type="number"
              min="0"
            />
          </div>
        </div>
        <DialogFooter
          ><Button :disabled="setupSaving || !serviceForm.name.trim()" @click="addService"
            >Add service</Button
          ></DialogFooter
        ></DialogContent
      ></Dialog
    >
    <Dialog v-model:open="availabilityOpen"
      ><DialogContent
        ><DialogHeader
          ><DialogTitle
            >Weekly hours ·
            {{ availabilityPractitioner?.display_name }}</DialogTitle
          ><DialogDescription
            >These rules are evaluated in the clinic time zone for both
            reception and WhatsApp booking.</DialogDescription
          ></DialogHeader
        >
        <div class="space-y-3">
          <div class="grid grid-cols-3 gap-2">
            <div>
              <Label for="field-availabilityForm-day-of-week">Day</Label><select id="field-availabilityForm-day-of-week"
                v-model.number="availabilityForm.day_of_week"
                class="w-full rounded-md border bg-background p-2"
              >
                <option
                  v-for="(day, index) in dayNames"
                  :key="day"
                  :value="index"
                >
                  {{ day }}
                </option>
              </select>
            </div>
            <div>
              <Label for="field-availabilityForm-start">Start</Label><Input id="field-availabilityForm-start" v-model="availabilityForm.start" type="time" />
            </div>
            <div>
              <Label for="field-availabilityForm-end">End</Label><Input id="field-availabilityForm-end" v-model="availabilityForm.end" type="time" />
            </div>
          </div>
          <div>
            <Label for="field-availabilityForm-slot-interval-minutes">Slot interval</Label><Input id="field-availabilityForm-slot-interval-minutes"
              v-model.number="availabilityForm.slot_interval_minutes"
              type="number"
              min="5" max="240"
            />
          </div>
          <div class="max-h-40 space-y-2 overflow-auto rounded-lg border p-3">
            <p
              v-if="!availabilityRules.length"
              class="text-sm text-muted-foreground"
            >
              No weekly hours added yet.
            </p>
            <p v-for="rule in availabilityRules" :key="rule.id" class="text-sm">
              {{ dayNames[rule.day_of_week] }} ·
              {{ asTime(rule.start_minute) }}–{{ asTime(rule.end_minute) }} ·
              every {{ rule.slot_interval_minutes }} min
            </p>
          </div>
        </div>
        <DialogFooter
          ><Button
            :disabled="
              setupSaving || !availabilityPractitioner ||
              availabilityForm.start >= availabilityForm.end
            "
            @click="addAvailability"
            >Add hours</Button
          ></DialogFooter
        ></DialogContent
      ></Dialog
    >
    <Dialog v-model:open="bookingOpen"
      ><DialogContent
        ><DialogHeader
          ><DialogTitle>Book appointment</DialogTitle
          ><DialogDescription
            >Final availability is checked again by the server when you
            confirm.</DialogDescription
          ></DialogHeader
        >
        <div class="space-y-3">
          <div>
            <Label for="field-booking-whatsapp-account">WhatsApp account</Label><select id="field-booking-whatsapp-account"
              v-model="booking.whatsapp_account"
              class="w-full rounded-md border bg-background p-2"
            >
              <option value="">Select account</option>
              <option
                v-for="account in accounts"
                :key="account.id"
                :value="account.name"
              >
                {{ account.name }}
              </option>
            </select>
          </div>
          <div>
            <Label for="field-booking-contact-id">Patient</Label><select id="field-booking-contact-id"
              v-model="booking.contact_id"
              class="w-full rounded-md border bg-background p-2"
            >
              <option value="">Select patient</option>
              <option
                v-for="contact in contacts.filter(
                  (c: any) =>
                    !booking.whatsapp_account ||
                    c.whatsapp_account === booking.whatsapp_account,
                )"
                :key="contact.id"
                :value="contact.id"
              >
                {{ contact.profile_name || contact.phone_number }}
              </option>
            </select>
          </div>
          <div>
            <Label for="field-booking-practitioner-id">Practitioner</Label><select id="field-booking-practitioner-id"
              v-model="booking.practitioner_id"
              class="w-full rounded-md border bg-background p-2"
            >
              <option value="">Select practitioner</option>
              <option
                v-for="person in practitioners"
                :key="person.id"
                :value="person.id"
              >
                {{ person.display_name }}
              </option>
            </select>
          </div>
          <div>
            <Label for="field-booking-service-id">Service</Label><select id="field-booking-service-id"
              v-model="booking.service_id"
              class="w-full rounded-md border bg-background p-2"
            >
              <option value="">Default slot</option>
              <option
                v-for="service in services"
                :key="service.id"
                :value="service.id"
              >
                {{ service.name }}
              </option>
            </select>
          </div>
          <div>
            <Label for="field-booking-starts-at">Available time on {{ date }}</Label><select id="field-booking-starts-at"
              v-model="booking.starts_at"
              class="w-full rounded-md border bg-background p-2"
              :disabled="!booking.practitioner_id || slotsLoading"
            >
              <option value="">Select a time</option>
              <option
                v-for="slot in slots"
                :key="slot.starts_at"
                :value="slot.starts_at"
              >
                {{ time(slot.starts_at) }}
              </option>
            </select>
          </div>
        </div>
        <DialogFooter
          ><p v-if="slotsError" role="alert" class="text-sm text-destructive">{{ slotsError }}</p>
          <p v-else-if="slotsLoading" role="status" class="text-sm text-muted-foreground">Checking available times…</p>
          <p v-else-if="booking.practitioner_id && !slots.length" class="text-sm text-muted-foreground">No times available. Choose another day or check the practitioner’s hours.</p>
          <Button
            :disabled="
              bookingSaving || slotsLoading ||
              !booking.whatsapp_account ||
              !booking.contact_id ||
              !booking.practitioner_id ||
              !booking.starts_at
            "
            @click="bookAppointment"
            >{{ bookingSaving ? "Confirming…" : "Confirm appointment" }}</Button
          ></DialogFooter
        ></DialogContent
      ></Dialog
    >
    <Dialog v-model:open="profileOpen"
      ><DialogContent
        ><DialogHeader
          ><DialogTitle>Clinic setup</DialogTitle
          ><DialogDescription
            >Set your clinic’s name, location, and appointment preferences.</DialogDescription
          ></DialogHeader
        >
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <Label for="field-profileForm-display-name">Clinic name</Label><Input id="field-profileForm-display-name" v-model="profileForm.display_name" />
          </div>
          <div>
            <Label for="field-profileForm-timezone">Time zone</Label><Input id="field-profileForm-timezone" v-model="profileForm.timezone" placeholder="Asia/Kolkata" />
          </div>
          <div>
            <Label for="field-profileForm-reception-phone">Reception phone</Label><Input id="field-profileForm-reception-phone" v-model="profileForm.reception_phone" />
          </div>
          <div>
            <Label for="field-profileForm-default-slot-minutes">Default slot minutes</Label><Input id="field-profileForm-default-slot-minutes"
              v-model.number="profileForm.default_slot_minutes"
              type="number"
              min="5" max="240"
            />
          </div>
          <div class="sm:col-span-2">
            <Label for="field-profileForm-address">Address</Label><Input id="field-profileForm-address" v-model="profileForm.address" />
          </div>
          <label class="flex items-center gap-2 sm:col-span-2"
            ><input v-model="profileForm.booking_enabled" type="checkbox" />
            Enable WhatsApp appointment booking</label
          >
        </div>
        <DialogFooter
          ><p v-if="profileError" role="alert" class="text-sm text-destructive">{{ profileError }}</p>
          <Button
            :disabled="
              appointmentSaving ||
              !profileForm.display_name ||
              !profileForm.timezone
            "
            @click="saveProfile"
            >{{ appointmentSaving ? "Saving…" : "Save clinic setup" }}</Button
          ></DialogFooter
        ></DialogContent
      ></Dialog
    >
    <Dialog v-model:open="rescheduleOpen"
      ><DialogContent
        ><DialogHeader
          ><DialogTitle>Move appointment</DialogTitle
          ><DialogDescription
            >{{ patient(selectedAppointment!) }} · fresh availability is checked
            before saving.</DialogDescription
          ></DialogHeader
        >
        <div class="space-y-3">
          <div>
            <Label for="field-rescheduleDate">Date</Label><Input id="field-rescheduleDate"
              v-model="rescheduleDate"
              type="date"
              @change="loadRescheduleSlots"
            />
          </div>
          <div>
            <Label for="field-rescheduleStartsAt">Available time</Label><select id="field-rescheduleStartsAt"
              v-model="rescheduleStartsAt"
              class="w-full rounded-md border bg-background p-2"
            >
              <option value="">Select a time</option>
              <option
                v-for="slot in rescheduleSlots"
                :key="slot.starts_at"
                :value="slot.starts_at"
              >
                {{ time(slot.starts_at) }}
              </option>
            </select>
          </div>
        </div>
        <p v-if="rescheduleError" role="alert" class="text-sm text-destructive">{{ rescheduleError }}</p>
        <p v-else-if="rescheduleLoading" role="status" class="text-sm text-muted-foreground">Checking available times…</p>
        <p v-else-if="!rescheduleSlots.length" class="text-sm text-muted-foreground">No available times for this day.</p>
        <DialogFooter
          ><Button
            :disabled="appointmentSaving || rescheduleLoading || !rescheduleStartsAt"
            @click="reschedule"
            >{{ appointmentSaving ? "Saving…" : "Confirm new time" }}</Button
          ></DialogFooter
        ></DialogContent
      ></Dialog
    >
    <Dialog v-model:open="cancelOpen"
      ><DialogContent
        ><DialogHeader
          ><DialogTitle>Cancel appointment</DialogTitle
          ><DialogDescription
            >This is retained as an auditable cancellation and may trigger an
            eligible waitlist offer.</DialogDescription
          ></DialogHeader
        >
        <div>
          <Label for="field-cancellationReason">Reason (optional)</Label><Input id="field-cancellationReason"
            v-model="cancellationReason"
            placeholder="e.g. Patient requested cancellation"
          />
        </div>
        <DialogFooter
          ><Button
            variant="destructive"
            :disabled="appointmentSaving"
            @click="cancel"
            >{{
              appointmentSaving ? "Cancelling…" : "Cancel appointment"
            }}</Button
          ></DialogFooter
        ></DialogContent
      ></Dialog
    >
  </div>
</template>

<style scoped>
.clinic-workspace {
  background:
    radial-gradient(
      circle at 82% -10%,
      rgb(20 184 166 / 0.13),
      transparent 31%
    ),
    linear-gradient(180deg, hsl(var(--background)), hsl(var(--muted) / 0.32));
}
.clinic-shell {
  padding-bottom: 2rem;
}
.clinic-hero {
  border: 1px solid rgb(45 212 191 / 0.2);
  border-radius: 1rem;
  padding: 1.35rem;
  background: linear-gradient(120deg, rgb(15 118 110 / 0.09), transparent 55%);
  box-shadow: 0 12px 32px rgb(15 23 42 / 0.05);
}
.eyebrow {
  color: rgb(13 148 136);
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}
.premium-primary {
  background: linear-gradient(135deg, rgb(13 148 136), rgb(5 150 105));
  box-shadow: 0 8px 18px rgb(13 148 136 / 0.22);
}
.premium-primary:hover {
  filter: brightness(1.06);
  transform: translateY(-1px);
}
.date-control {
  border-color: rgb(20 184 166 / 0.3);
  background: hsl(var(--background) / 0.8);
}
.metric-card {
  overflow: hidden;
  position: relative;
  border-color: rgb(148 163 184 / 0.18);
  box-shadow: 0 8px 22px rgb(15 23 42 / 0.045);
}
.metric-card::after {
  content: "";
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  background: currentColor;
}
.metric-confirmed {
  color: rgb(5 150 105);
}
.metric-pending {
  color: rgb(217 119 6);
}
.metric-waitlist {
  color: rgb(13 148 136);
}
.metric-card :deep(p) {
  color: inherit;
}
.metric-card :deep(.text-muted-foreground) {
  color: hsl(var(--muted-foreground));
}
.metric-icon {
  margin-top: 0.25rem;
}
.setup-card {
  background: linear-gradient(135deg, rgb(240 253 250 / 0.9), hsl(var(--card)));
  box-shadow: 0 20px 45px rgb(15 118 110 / 0.08);
}
:global(.dark) .setup-card {
  background: linear-gradient(135deg, rgb(15 118 110 / 0.13), hsl(var(--card)));
}
.week-calendar-card {
  border-color: rgb(20 184 166 / 0.2);
  box-shadow: 0 12px 28px rgb(15 23 42 / 0.045);
}
.daily-desk {
  border-color: rgb(20 184 166 / 0.24);
  background: linear-gradient(
    110deg,
    rgb(240 253 250 / 0.78),
    hsl(var(--card))
  );
  box-shadow: 0 12px 28px rgb(15 23 42 / 0.04);
}
.daily-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
}
.daily-step {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 0.65rem;
  border: 1px solid rgb(148 163 184 / 0.2);
  border-radius: 0.8rem;
  background: hsl(var(--background) / 0.62);
  padding: 0.85rem;
}
.daily-step > :last-child {
  grid-column: 2;
  justify-self: start;
}
.step-number {
  display: grid;
  grid-row: span 2;
  height: 2rem;
  width: 2rem;
  place-items: center;
  border-radius: 999px;
  background: rgb(13 148 136 / 0.13);
  color: rgb(13 148 136);
  font-weight: 800;
}
.calendar-legend {
  border-color: rgb(20 184 166 / 0.28);
  color: rgb(13 148 136);
}
.week-calendar {
  display: grid;
  grid-template-columns: repeat(7, minmax(142px, 1fr));
  gap: 0.65rem;
  overflow-x: auto;
  padding-bottom: 0.35rem;
}
.calendar-day {
  min-height: 190px;
  border: 1px solid hsl(var(--border));
  border-radius: 0.85rem;
  background: hsl(var(--background) / 0.65);
  padding: 0.65rem;
  text-align: left;
  transition: 0.18s ease;
}
.calendar-day:hover {
  border-color: rgb(20 184 166 / 0.55);
  background: rgb(240 253 250 / 0.58);
  transform: translateY(-2px);
}
.calendar-day--selected {
  border-color: rgb(13 148 136);
  box-shadow: 0 0 0 2px rgb(13 148 136 / 0.13);
}
.calendar-day--today .calendar-day-header strong {
  background: rgb(13 148 136);
  color: white;
}
.calendar-day-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: hsl(var(--muted-foreground));
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
}
.calendar-day-header strong {
  display: grid;
  height: 1.8rem;
  width: 1.8rem;
  place-items: center;
  border-radius: 999px;
  color: hsl(var(--foreground));
  font-size: 0.85rem;
}
.calendar-events {
  margin-top: 0.75rem;
  display: grid;
  gap: 0.38rem;
}
.calendar-event {
  display: grid;
  gap: 0.05rem;
  border-left: 3px solid rgb(13 148 136);
  border-radius: 0.4rem;
  background: rgb(240 253 250);
  padding: 0.35rem 0.42rem;
  font-size: 0.68rem;
  cursor: pointer;
}
.calendar-event:hover {
  filter: brightness(0.97);
}
.calendar-event--pending {
  border-color: rgb(217 119 6);
  background: rgb(255 251 235);
}
.calendar-event span {
  color: hsl(var(--muted-foreground));
}
.calendar-event b {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.72rem;
}
.calendar-event small {
  overflow: hidden;
  color: hsl(var(--muted-foreground));
  text-overflow: ellipsis;
  white-space: nowrap;
}
.calendar-open-slot {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  border: 1px dashed rgb(20 184 166 / 0.5);
  border-radius: 0.5rem;
  color: rgb(13 148 136);
  padding: 0.55rem 0.3rem;
  font-size: 0.72rem;
  font-weight: 600;
}
.calendar-more {
  color: hsl(var(--muted-foreground));
  font-size: 0.7rem;
  text-align: center;
}
:global(.dark) .calendar-day:hover {
  background: rgb(13 148 136 / 0.1);
}
.calendar-day :deep(*) {
  color: inherit;
}
@media (max-width: 900px) {
  .daily-steps {
    grid-template-columns: 1fr;
  }
}
.calendar-day {
  cursor: pointer;
}
@media (max-width: 640px) {
  .clinic-hero {
    padding: 1rem;
  }
}
</style>

<style scoped>
.setup-card, .daily-desk { background: linear-gradient(120deg, hsl(var(--primary) / .09), hsl(var(--card)) 70%); }
.calendar-day:hover { background: hsl(var(--primary) / .07); }
.calendar-event { background: hsl(var(--primary) / .1); color: hsl(var(--foreground)); }
.calendar-event--pending { background: hsl(38 92% 50% / .12); }
.calendar-day:focus-visible, .calendar-event:focus-visible, .calendar-open-slot:focus-visible { outline: 2px solid hsl(var(--ring)); outline-offset: 3px; }
.clinic-shell { width: 100%; min-width: 0; }
.eyebrow { color: hsl(var(--primary)); }
</style>
